// SPDX-License-Identifier: AGPL-3.0-only

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package applyconfiguration

import (
	v1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	kindsv1 "github.com/grafana/grafana-apiserver/pkg/client/applyconfiguration/kinds/v1"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
)

// ForKind returns an apply configuration type for the given GroupVersionKind, or nil if no
// apply configuration type exists for the given GroupVersionKind.
func ForKind(kind schema.GroupVersionKind) interface{} {
	switch kind {
	// Group=kinds.grafana.com, Version=v1
	case v1.SchemeGroupVersion.WithKind("GrafanaResourceDefinition"):
		return &kindsv1.GrafanaResourceDefinitionApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("GrafanaResourceDefinitionCondition"):
		return &kindsv1.GrafanaResourceDefinitionConditionApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("GrafanaResourceDefinitionNames"):
		return &kindsv1.GrafanaResourceDefinitionNamesApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("GrafanaResourceDefinitionSpec"):
		return &kindsv1.GrafanaResourceDefinitionSpecApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("GrafanaResourceDefinitionStatus"):
		return &kindsv1.GrafanaResourceDefinitionStatusApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("GrafanaResourceDefinitionVersion"):
		return &kindsv1.GrafanaResourceDefinitionVersionApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("GrafanaResourceSubresources"):
		return &kindsv1.GrafanaResourceSubresourcesApplyConfiguration{}
	case v1.SchemeGroupVersion.WithKind("GrafanaResourceSubresourceScale"):
		return &kindsv1.GrafanaResourceSubresourceScaleApplyConfiguration{}

	}
	return nil
}
