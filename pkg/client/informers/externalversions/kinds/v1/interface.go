// SPDX-License-Identifier: AGPL-3.0-only

// Code generated by informer-gen. DO NOT EDIT.

package v1

import (
	internalinterfaces "github.com/grafana/grafana-apiserver/pkg/client/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// GrafanaResourceDefinitions returns a GrafanaResourceDefinitionInformer.
	GrafanaResourceDefinitions() GrafanaResourceDefinitionInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// GrafanaResourceDefinitions returns a GrafanaResourceDefinitionInformer.
func (v *version) GrafanaResourceDefinitions() GrafanaResourceDefinitionInformer {
	return &grafanaResourceDefinitionInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}