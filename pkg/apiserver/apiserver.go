// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/apiserver/apiserver.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package apiserver

import (
	"fmt"
	"net/http"
	"reflect"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/endpoints/discovery"
	"k8s.io/apiserver/pkg/endpoints/discovery/aggregated"
	genericregistry "k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/options"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	clientRest "k8s.io/client-go/rest"
	apiregistrationClientset "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	apiregistrationInformers "k8s.io/kube-aggregator/pkg/client/informers/externalversions"

	"github.com/grafana/grafana-apiserver/pkg/apis/kinds"
	"github.com/grafana/grafana-apiserver/pkg/apis/kinds/install"
	v1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	"github.com/grafana/grafana-apiserver/pkg/client/clientset/clientset"
	externalinformers "github.com/grafana/grafana-apiserver/pkg/client/informers/externalversions"
	"github.com/grafana/grafana-apiserver/pkg/controller/apiapproval"
	"github.com/grafana/grafana-apiserver/pkg/controller/apiregistration"
	"github.com/grafana/grafana-apiserver/pkg/controller/establish"
	"github.com/grafana/grafana-apiserver/pkg/controller/finalizer"
	"github.com/grafana/grafana-apiserver/pkg/controller/status"
	"github.com/grafana/grafana-apiserver/pkg/registry/grafanaresourcedefinition"
	"github.com/grafana/grafana-apiserver/pkg/storage/filepath"
)

var (
	Scheme = runtime.NewScheme()
	Codecs = serializer.NewCodecFactory(Scheme)

	// if you modify this, make sure you update the crEncoder
	unversionedVersion = schema.GroupVersion{Group: "", Version: "v1"}
	unversionedTypes   = []runtime.Object{
		&metav1.Status{},
		&metav1.WatchEvent{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	}
)

func init() {
	install.Install(Scheme)

	// we need to add the options to empty v1
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Group: "", Version: "v1"})
	Scheme.AddUnversionedTypes(unversionedVersion, unversionedTypes...)
}

// ExtraConfig holds custom apiserver config
type ExtraConfig struct {
	RESTOptionsGetter genericregistry.RESTOptionsGetter

	// MasterCount is used to detect whether cluster is HA, and if it is
	// the GRD Establishing will be hold by 5 seconds.
	MasterCount int

	RemoteKubeConfig *clientRest.Config
}

// Config defines the config for the apiserver
type Config struct {
	GenericConfig *genericapiserver.RecommendedConfig
	ExtraConfig   ExtraConfig
}

// GrafanaAPIServer contains state for a Kubernetes cluster master/api server.
type GrafanaAPIServer struct {
	GenericAPIServer *genericapiserver.GenericAPIServer

	// provided for easier embedding
	Informers externalinformers.SharedInformerFactory
}

type completedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package.
type CompletedConfig struct {
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (cfg *Config) Complete() CompletedConfig {
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		&cfg.ExtraConfig,
	}

	c.GenericConfig.EnableDiscovery = true
	if c.GenericConfig.Version == nil {
		c.GenericConfig.Version = &version.Info{
			Major: "0",
			Minor: "1",
		}
	}

	if c.ExtraConfig.RESTOptionsGetter == nil {
		c.ExtraConfig.RESTOptionsGetter = filepath.NewRESTOptionsGetter("/tmp/grafana-apiserver", unstructured.UnstructuredJSONScheme)
		c.GenericConfig.RESTOptionsGetter = filepath.NewRESTOptionsGetter("/tmp/grafana-apiserver", Codecs.LegacyCodec(v1.SchemeGroupVersion))
		c.GenericConfig.Config.RESTOptionsGetter = filepath.NewRESTOptionsGetter("/tmp/grafana-apiserver", Codecs.LegacyCodec(v1.SchemeGroupVersion))
	}

	return CompletedConfig{&c}
}

// New returns a new instance of GrafanaAPIServer from the given config.
func (c completedConfig) New(delegationTarget genericapiserver.DelegationTarget) (*GrafanaAPIServer, error) {
	genericServer, err := c.GenericConfig.New("grafana-apiserver", delegationTarget)
	if err != nil {
		return nil, err
	}

	discoveryReady := make(chan struct{})
	if err := genericServer.RegisterMuxAndDiscoveryCompleteSignal("GrafanaResourceDefinitionDiscoveryReady", discoveryReady); err != nil {
		return nil, err
	}

	s := &GrafanaAPIServer{
		GenericAPIServer: genericServer,
	}

	apiResourceConfig := c.GenericConfig.MergedResourceConfig
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(kinds.GroupName, Scheme, metav1.ParameterCodec, Codecs)
	storage := map[string]rest.Storage{}
	// grafanaresourcedefinitions
	if resource := "grafanaresourcedefinitions"; apiResourceConfig.ResourceEnabled(v1.SchemeGroupVersion.WithResource(resource)) {
		grafanaResourceDefinitionStorage, err := grafanaresourcedefinition.NewREST(Scheme, c.GenericConfig.RESTOptionsGetter)
		if err != nil {
			return nil, err
		}
		storage[resource] = grafanaResourceDefinitionStorage
		storage[resource+"/status"] = grafanaresourcedefinition.NewStatusREST(Scheme, grafanaResourceDefinitionStorage)
	}
	if len(storage) > 0 {
		apiGroupInfo.VersionedResourcesStorageMap[v1.SchemeGroupVersion.Version] = storage
	}

	if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	grdClient, err := clientset.NewForConfig(s.GenericAPIServer.LoopbackClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}
	s.Informers = externalinformers.NewSharedInformerFactory(grdClient, 5*time.Minute)
	delegateHandler := delegationTarget.UnprotectedHandler()
	if delegateHandler == nil {
		delegateHandler = http.NotFoundHandler()
	}

	versionDiscovery := &versionDiscoveryHandler{
		discovery: map[schema.GroupVersion]*discovery.APIVersionHandler{},
		delegate:  delegateHandler,
	}

	groupDiscovery := &groupDiscoveryHandler{
		discovery: map[string]*discovery.APIGroupHandler{},
		delegate:  delegateHandler,
	}

	establishingController := establish.NewEstablishingController(s.Informers.Kinds().V1().GrafanaResourceDefinitions(), grdClient.KindsV1())
	grdHandler, err := NewGrafanaResourceDefinitionHandler(
		versionDiscovery,
		groupDiscovery,
		s.Informers.Kinds().V1().GrafanaResourceDefinitions(),
		delegateHandler,
		c.ExtraConfig.RESTOptionsGetter,
		c.GenericConfig.AdmissionControl,
		establishingController,
		c.ExtraConfig.MasterCount,
		s.GenericAPIServer.Authorizer,
		c.GenericConfig.RequestTimeout,
		time.Duration(c.GenericConfig.MinRequestTimeout)*time.Second,
		apiGroupInfo.StaticOpenAPISpec,
		c.GenericConfig.MaxRequestBodyBytes,
	)
	if err != nil {
		return nil, err
	}

	s.GenericAPIServer.Handler.NonGoRestfulMux.Handle("/apis", grdHandler)
	s.GenericAPIServer.Handler.NonGoRestfulMux.HandlePrefix("/apis/", grdHandler)
	s.GenericAPIServer.RegisterDestroyFunc(grdHandler.destroy)

	aggregatedDiscoveryManager := genericServer.AggregatedDiscoveryGroupManager
	if aggregatedDiscoveryManager != nil {
		aggregatedDiscoveryManager = aggregatedDiscoveryManager.WithSource(aggregated.CRDSource + 1)
	}

	var apiInformers apiregistrationInformers.SharedInformerFactory
	var autoRegistrationController *apiregistration.AutoRegisterController
	var grdRegistrationController *apiregistration.GRDRegistrationController
	_, hostPort, err := c.GenericConfig.SecureServing.HostPort()
	if err != nil {
		return nil, err
	}

	if c.ExtraConfig.RemoteKubeConfig != nil {
		apiClient, err := apiregistrationClientset.NewForConfig(c.ExtraConfig.RemoteKubeConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create clientset: %v", err)
		}
		apiInformers = apiregistrationInformers.NewSharedInformerFactory(apiClient, 5*time.Minute)
		autoRegistrationController = apiregistration.NewAutoRegisterController(apiInformers.Apiregistration().V1().APIServices(), apiClient.ApiregistrationV1())
		grdRegistrationController = apiregistration.NewGRDRegistrationController(
			s.Informers.Kinds().V1().GrafanaResourceDefinitions(),
			autoRegistrationController,
			int32(hostPort),
		)
	}

	discoveryController := NewDiscoveryController(s.Informers.Kinds().V1().GrafanaResourceDefinitions(), versionDiscovery, groupDiscovery, aggregatedDiscoveryManager)
	namingController := status.NewNamingConditionController(s.Informers.Kinds().V1().GrafanaResourceDefinitions(), grdClient.KindsV1())
	apiApprovalController := apiapproval.NewKubernetesAPIApprovalPolicyConformantConditionController(s.Informers.Kinds().V1().GrafanaResourceDefinitions(), grdClient.KindsV1())
	finalizingController := finalizer.NewGRDFinalizer(
		s.Informers.Kinds().V1().GrafanaResourceDefinitions(),
		grdClient.KindsV1(),
		grdHandler,
	)

	s.GenericAPIServer.AddPostStartHookOrDie("start-grafana-apiserver-informers", func(context genericapiserver.PostStartHookContext) error {
		s.Informers.Start(context.StopCh)
		if apiInformers != nil {
			apiInformers.Start(context.StopCh)
		}
		return nil
	})

	s.GenericAPIServer.AddPostStartHookOrDie("start-grafana-apiserver-controllers", func(context genericapiserver.PostStartHookContext) error {
		if apiInformers != nil {
			go autoRegistrationController.Run(5, context.StopCh)
			go grdRegistrationController.Run(5, context.StopCh)
		}
		go namingController.Run(context.StopCh)
		go establishingController.Run(context.StopCh)
		go apiApprovalController.Run(5, context.StopCh)
		go finalizingController.Run(5, context.StopCh)

		discoverySyncedCh := make(chan struct{})
		go discoveryController.Run(context.StopCh, discoverySyncedCh)
		select {
		case <-context.StopCh:
		case <-discoverySyncedCh:
		}

		return nil
	})

	// we don't want to report healthy until we can handle all GRDs that have already been registered.  Waiting for the informer
	// to sync makes sure that the lister will be valid before we begin.  There may still be races for GRDs added after startup,
	// but we won't go healthy until we can handle the ones already present.
	s.GenericAPIServer.AddPostStartHookOrDie("grd-informer-synced", func(context genericapiserver.PostStartHookContext) error {
		return wait.PollImmediateUntil(100*time.Millisecond, func() (bool, error) {
			if s.Informers.Kinds().V1().GrafanaResourceDefinitions().Informer().HasSynced() {
				close(discoveryReady)
				return true, nil
			}
			return false, nil
		}, context.StopCh)
	})

	return s, nil
}

func DefaultAPIResourceConfigSource() *serverstorage.ResourceConfig {
	ret := serverstorage.NewResourceConfig()
	// NOTE: GroupVersions listed here will be enabled by default. Don't put alpha versions in the list.
	ret.EnableVersions(
		v1.SchemeGroupVersion,
	)

	return ret
}

func NewGRDRESTOptionsGetter(etcdOptions options.EtcdOptions) (genericregistry.RESTOptionsGetter, error) {
	etcdOptions.StorageConfig.Codec = unstructured.UnstructuredJSONScheme
	etcdOptions.WatchCacheSizes = nil      // this control is not provided for custom resources
	etcdOptions.SkipHealthEndpoints = true // avoid double wiring of health checks

	// creates a generic apiserver config for etcdOptions to mutate
	c := genericapiserver.Config{}
	if err := etcdOptions.ApplyTo(&c); err != nil {
		return nil, err
	}
	restOptionsGetter := c.RESTOptionsGetter
	if restOptionsGetter == nil {
		return nil, fmt.Errorf("server.Config RESTOptionsGetter should not be nil")
	}
	// sanity check that no other fields are set
	c.RESTOptionsGetter = nil
	if !reflect.DeepEqual(c, genericapiserver.Config{}) {
		return nil, fmt.Errorf("only RESTOptionsGetter should have been mutated in server.Config")
	}
	return restOptionsGetter, nil
}
