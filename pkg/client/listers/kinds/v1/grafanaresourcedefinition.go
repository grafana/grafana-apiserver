// SPDX-License-Identifier: AGPL-3.0-only

// Code generated by lister-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// GrafanaResourceDefinitionLister helps list GrafanaResourceDefinitions.
// All objects returned here must be treated as read-only.
type GrafanaResourceDefinitionLister interface {
	// List lists all GrafanaResourceDefinitions in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1.GrafanaResourceDefinition, err error)
	// Get retrieves the GrafanaResourceDefinition from the index for a given name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1.GrafanaResourceDefinition, error)
	GrafanaResourceDefinitionListerExpansion
}

// grafanaResourceDefinitionLister implements the GrafanaResourceDefinitionLister interface.
type grafanaResourceDefinitionLister struct {
	indexer cache.Indexer
}

// NewGrafanaResourceDefinitionLister returns a new GrafanaResourceDefinitionLister.
func NewGrafanaResourceDefinitionLister(indexer cache.Indexer) GrafanaResourceDefinitionLister {
	return &grafanaResourceDefinitionLister{indexer: indexer}
}

// List lists all GrafanaResourceDefinitions in the indexer.
func (s *grafanaResourceDefinitionLister) List(selector labels.Selector) (ret []*v1.GrafanaResourceDefinition, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1.GrafanaResourceDefinition))
	})
	return ret, err
}

// Get retrieves the GrafanaResourceDefinition from the index for a given name.
func (s *grafanaResourceDefinitionLister) Get(name string) (*v1.GrafanaResourceDefinition, error) {
	obj, exists, err := s.indexer.GetByKey(name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1.Resource("grafanaresourcedefinition"), name)
	}
	return obj.(*v1.GrafanaResourceDefinition), nil
}
