// SPDX-License-Identifier: AGPL-3.0-only

package filepath

import (
	"path"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	flowcontrolrequest "k8s.io/apiserver/pkg/util/flowcontrol/request"
)

var _ generic.RESTOptionsGetter = (*RESTOptionsGetter)(nil)

type RESTOptionsGetter struct {
	path  string
	Codec runtime.Codec
}

func NewRESTOptionsGetter(path string, codec runtime.Codec) *RESTOptionsGetter {
	if path == "" {
		path = "/tmp/grafana-apiserver"
	}

	return &RESTOptionsGetter{path: path, Codec: codec}
}

func (f *RESTOptionsGetter) GetRESTOptions(resource schema.GroupResource) (generic.RESTOptions, error) {
	storageConfig := &storagebackend.ConfigForResource{
		Config: storagebackend.Config{
			Type:                      "custom",
			Prefix:                    f.path,
			Transport:                 storagebackend.TransportConfig{},
			Paging:                    false,
			Codec:                     f.Codec,
			EncodeVersioner:           nil,
			Transformer:               nil,
			CompactionInterval:        0,
			CountMetricPollPeriod:     0,
			DBMetricPollInterval:      0,
			HealthcheckTimeout:        0,
			ReadycheckTimeout:         0,
			StorageObjectCountTracker: nil,
		},
		GroupResource: resource,
	}

	ret := generic.RESTOptions{
		StorageConfig:             storageConfig,
		Decorator:                 NewStorage,
		DeleteCollectionWorkers:   0,
		EnableGarbageCollection:   false,
		ResourcePrefix:            path.Join(storageConfig.Prefix, resource.Group, resource.Resource),
		CountMetricPollPeriod:     1 * time.Second,
		StorageObjectCountTracker: flowcontrolrequest.NewStorageObjectCountTracker(),
	}

	return ret, nil
}
