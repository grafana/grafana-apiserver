package options

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/spf13/pflag"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
	"k8s.io/apiserver/pkg/util/openapi"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	netutils "k8s.io/utils/net"

	v1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	"github.com/grafana/grafana-apiserver/pkg/apiserver"
	"github.com/grafana/grafana-apiserver/pkg/client/clientset/clientset/scheme"
	generatedopenapi "github.com/grafana/grafana-apiserver/pkg/client/openapi"
)

const defaultEtcdPathPrefix = "/registry/kind.grafana.com"

// GrafanaAPIServerOptions contains state for master/api server
type GrafanaAPIServerOptions struct {
	ServerRunOptions   *genericoptions.ServerRunOptions
	RecommendedOptions *genericoptions.RecommendedOptions
	APIEnablement      *genericoptions.APIEnablementOptions

	StdOut io.Writer
	StdErr io.Writer

	AlternateDNS []string
}

// NewGrafanaAPIServerOptions returns a new GrafanaAPIServerOptions
func NewGrafanaAPIServerOptions(out, errOut io.Writer) *GrafanaAPIServerOptions {
	o := &GrafanaAPIServerOptions{
		RecommendedOptions: genericoptions.NewRecommendedOptions(
			defaultEtcdPathPrefix,
			apiserver.Codecs.LegacyCodec(v1.SchemeGroupVersion),
		),
		ServerRunOptions: genericoptions.NewServerRunOptions(),
		APIEnablement:    genericoptions.NewAPIEnablementOptions(),

		StdOut: out,
		StdErr: errOut,
	}
	return o
}

// AddFlags adds the grafana-apiserver flags to the flagset.
func (o GrafanaAPIServerOptions) AddFlags(fs *pflag.FlagSet) {
	o.ServerRunOptions.AddUniversalFlags(fs)
	o.RecommendedOptions.AddFlags(fs)
	o.APIEnablement.AddFlags(fs)
}

// Validate validates the kinds-apiserver options.
func (o GrafanaAPIServerOptions) Validate() error {
	// set etcd options to nil if servers are not specified
	if o.RecommendedOptions.Etcd != nil && len(o.RecommendedOptions.Etcd.StorageConfig.Transport.ServerList) == 0 {
		o.RecommendedOptions.Etcd = nil
	}

	errors := []error{}
	errors = append(errors, o.ServerRunOptions.Validate()...)
	errors = append(errors, o.RecommendedOptions.Validate()...)
	errors = append(errors, o.APIEnablement.Validate(apiserver.Scheme)...)
	return utilerrors.NewAggregate(errors)
}

// Complete fills in fields required to have valid data
func (o GrafanaAPIServerOptions) Complete() error {
	return nil
}

// Config returns config for the api server given GrafanaAPIServerOptions
func (o GrafanaAPIServerOptions) Config() (*apiserver.Config, error) {
	// TODO have a "real" external address
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", o.AlternateDNS, []net.IP{netutils.ParseIPSloppy("127.0.0.1")}); err != nil {
		return nil, fmt.Errorf("error creating self-signed certificates: %v", err)
	}

	serverConfig := genericapiserver.NewRecommendedConfig(apiserver.Codecs)

	// when grafana runs this apiserver in standalone mode, it will not have a CoreAPI options. we need
	// to fake some pieces of the config to avoid errors when applying the generic recommended options to the config.
	if o.RecommendedOptions.CoreAPI == nil {
		serverConfig.SharedInformerFactory = informers.NewSharedInformerFactory(fake.NewSimpleClientset(), 10*time.Minute)
		serverConfig.ClientConfig = &rest.Config{}
	}

	if err := o.ServerRunOptions.ApplyTo(&serverConfig.Config); err != nil {
		return nil, err
	}
	if err := o.RecommendedOptions.ApplyTo(serverConfig); err != nil {
		return nil, err
	}
	if err := o.APIEnablement.ApplyTo(&serverConfig.Config, apiserver.DefaultAPIResourceConfigSource(), apiserver.Scheme); err != nil {
		return nil, err
	}

	serverConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(openapi.GetOpenAPIDefinitionsWithoutDisabledFeatures(generatedopenapi.GetOpenAPIDefinitions), openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme))
	serverConfig.OpenAPIV3Config = genericapiserver.DefaultOpenAPIV3Config(openapi.GetOpenAPIDefinitionsWithoutDisabledFeatures(generatedopenapi.GetOpenAPIDefinitions), openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme))
	serverConfig.SkipOpenAPIInstallation = false

	coreAPIConfig, err := o.getCoreAPIRESTConfig()
	if err != nil {
		return nil, err
	}
	extraConfig := apiserver.ExtraConfig{RemoteKubeConfig: coreAPIConfig}

	// if etcd config is specified, use it for GRD storage by default
	if o.RecommendedOptions.Etcd != nil {
		extraConfig.RESTOptionsGetter, err = apiserver.NewGRDRESTOptionsGetter(*o.RecommendedOptions.Etcd)
		if err != nil {
			return nil, err
		}
	}

	config := &apiserver.Config{
		GenericConfig: serverConfig,
		ExtraConfig:   extraConfig,
	}

	return config, nil
}

// RunGrafanaAPIServer starts a new GrafanaAPIServer given GrafanaAPIServerOptions
func (o GrafanaAPIServerOptions) Run(stopCh <-chan struct{}) error {
	config, err := o.Config()
	if err != nil {
		return err
	}

	server, err := config.Complete().New(genericapiserver.NewEmptyDelegate())
	if err != nil {
		return err
	}

	return server.GenericAPIServer.PrepareRun().Run(stopCh)
}

func (o GrafanaAPIServerOptions) getCoreAPIRESTConfig() (*rest.Config, error) {
	// attempt to load kubeconfig file from the path specified in the CoreAPI options
	if o.RecommendedOptions != nil && o.RecommendedOptions.CoreAPI != nil && len(o.RecommendedOptions.CoreAPI.CoreAPIKubeconfigPath) > 0 {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: o.RecommendedOptions.CoreAPI.CoreAPIKubeconfigPath}
		loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
		return loader.ClientConfig()
	}

	// if kubeconfig file was not specified, attempt to load in-cluster config. ignore error if not in cluster.
	clientConfig, err := rest.InClusterConfig()
	if err != nil && err != rest.ErrNotInCluster {
		return nil, err
	}

	return clientConfig, nil
}
