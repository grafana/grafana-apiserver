// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/apis/apiextensions/types.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package kinds

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// KubeAPIApprovedAnnotation is an annotation that must be set to create a GRD for the k8s.io, *.k8s.io, kubernetes.io, or *.kubernetes.io namespaces.
	// The value should be a link to a URL where the current spec was approved, so updates to the spec should also update the URL.
	// If the API is unapproved, you may set the annotation to a string starting with `"unapproved"`.  For instance, `"unapproved, temporarily squatting"` or `"unapproved, experimental-only"`.  This is discouraged.
	KubeAPIApprovedAnnotation = "api-approved.kubernetes.io"
)

// GrafanaResourceDefinitionSpec describes how a user wants their resource to appear
type GrafanaResourceDefinitionSpec struct {
	// Group is the group this resource belongs in
	Group string
	// Version is the version this resource belongs in
	// Should be always first item in Versions field if provided.
	// Optional, but at least one of Version or Versions must be set.
	// Deprecated: Please use `Versions`.
	Version string
	// Names are the names used to describe this custom resource
	Names GrafanaResourceDefinitionNames
	// Scope indicates whether this resource is cluster or namespace scoped.  Default is namespaced
	Scope ResourceScope
	// Subresources describes the subresources for GrafanaResource
	// Optional, the global subresources for all versions.
	// Top-level and per-version subresources are mutually exclusive.
	// +optional
	Subresources *GrafanaResourceSubresources
	// Versions is the list of all supported versions for this resource.
	// If Version field is provided, this field is optional.
	// Validation: All versions must use the same validation schema for now. i.e., top
	// level Validation field is applied to all of these versions.
	// Order: The version name will be used to compute the order.
	// If the version string is "kube-like", it will sort above non "kube-like" version strings, which are ordered
	// lexicographically. "Kube-like" versions start with a "v", then are followed by a number (the major version),
	// then optionally the string "alpha" or "beta" and another number (the minor version). These are sorted first
	// by GA > beta > alpha (where GA is a version with no suffix such as beta or alpha), and then by comparing
	// major version, then minor version. An example sorted list of versions:
	// v10, v2, v1, v11beta2, v10beta3, v3beta1, v12alpha1, v11alpha2, foo1, foo10.
	Versions []GrafanaResourceDefinitionVersion

	// preserveUnknownFields disables pruning of object fields which are not
	// specified in the OpenAPI schema. apiVersion, kind, metadata and known
	// fields inside metadata are always preserved.
	// Defaults to true in v1beta and will default to false in v1.
	PreserveUnknownFields *bool
}

// GrafanaResourceDefinitionVersion describes a version for GRD.
type GrafanaResourceDefinitionVersion struct {
	// Name is the version name, e.g. “v1”, “v2beta1”, etc.
	Name string
	// Served is a flag enabling/disabling this version from being served via REST APIs
	Served bool
	// Storage flags the version as storage version. There must be exactly one flagged
	// as storage version.
	Storage bool
	// deprecated indicates this version of the custom resource API is deprecated.
	// When set to true, API requests to this version receive a warning header in the server response.
	// Defaults to false.
	Deprecated bool
	// deprecationWarning overrides the default warning returned to API clients.
	// May only be set when `deprecated` is true.
	// The default warning indicates this version is deprecated and recommends use
	// of the newest served version of equal or greater stability, if one exists.
	DeprecationWarning *string
	// Subresources describes the subresources for GrafanaResource
	// Top-level and per-version subresources are mutually exclusive.
	// Per-version subresources must not all be set to identical values (top-level subresources should be used instead)
	// This field is alpha-level and is only honored by servers that enable the GrafanaResourceWebhookConversion feature.
	// +optional
	Subresources *GrafanaResourceSubresources
}

// GrafanaResourceDefinitionNames indicates the names to serve this GrafanaResourceDefinition
type GrafanaResourceDefinitionNames struct {
	// Plural is the plural name of the resource to serve.  It must match the name of the GrafanaResourceDefinition-registration
	// too: plural.group and it must be all lowercase.
	Plural string
	// Singular is the singular name of the resource.  It must be all lowercase  Defaults to lowercased <kind>
	Singular string
	// ShortNames are short names for the resource.  It must be all lowercase.
	ShortNames []string
	// Kind is the serialized kind of the resource.  It is normally CamelCase and singular.
	Kind string
	// ListKind is the serialized kind of the list for this resource.  Defaults to <kind>List.
	ListKind string
	// Categories is a list of grouped resources custom resources belong to (e.g. 'all')
	// +optional
	Categories []string
}

// ResourceScope is an enum defining the different scopes available to a custom resource
type ResourceScope string

const (
	ClusterScoped   ResourceScope = "Cluster"
	NamespaceScoped ResourceScope = "Namespaced"
)

type ConditionStatus string

// These are valid condition statuses. "ConditionTrue" means a resource is in the condition.
// "ConditionFalse" means a resource is not in the condition. "ConditionUnknown" means kubernetes
// can't decide if a resource is in the condition or not. In the future, we could add other
// intermediate conditions, e.g. ConditionDegraded.
const (
	ConditionTrue    ConditionStatus = "True"
	ConditionFalse   ConditionStatus = "False"
	ConditionUnknown ConditionStatus = "Unknown"
)

// GrafanaResourceDefinitionConditionType is a valid value for GrafanaResourceDefinitionCondition.Type
type GrafanaResourceDefinitionConditionType string

const (
	// Established means that the resource has become active. A resource is established when all names are
	// accepted without a conflict for the first time. A resource stays established until deleted, even during
	// a later NamesAccepted due to changed names. Note that not all names can be changed.
	Established GrafanaResourceDefinitionConditionType = "Established"
	// NamesAccepted means the names chosen for this GrafanaResourceDefinition do not conflict with others in
	// the group and are therefore accepted.
	NamesAccepted GrafanaResourceDefinitionConditionType = "NamesAccepted"
	// NonStructuralSchema means that one or more OpenAPI schema is not structural.
	//
	// A schema is structural if it specifies types for all values, with the only exceptions of those with
	// - x-kubernetes-int-or-string: true — for fields which can be integer or string
	// - x-kubernetes-preserve-unknown-fields: true — for raw, unspecified JSON values
	// and there is no type, additionalProperties, default, nullable or x-kubernetes-* vendor extenions
	// specified under allOf, anyOf, oneOf or not.
	//
	// Non-structural schemas will not be allowed anymore in v1 API groups. Moreover, new features will not be
	// available for non-structural GRDs:
	// - pruning
	// - defaulting
	// - read-only
	// - OpenAPI publishing
	// - webhook conversion
	NonStructuralSchema GrafanaResourceDefinitionConditionType = "NonStructuralSchema"
	// Terminating means that the GrafanaResourceDefinition has been deleted and is cleaning up.
	Terminating GrafanaResourceDefinitionConditionType = "Terminating"
	// KubernetesAPIApprovalPolicyConformant indicates that an API in *.k8s.io or *.kubernetes.io is or is not approved.  For GRDs
	// outside those groups, this condition will not be set.  For GRDs inside those groups, the condition will
	// be true if .metadata.annotations["api-approved.kubernetes.io"] is set to a URL, otherwise it will be false.
	// See https://github.com/kubernetes/enhancements/pull/1111 for more details.
	KubernetesAPIApprovalPolicyConformant GrafanaResourceDefinitionConditionType = "KubernetesAPIApprovalPolicyConformant"
)

// GrafanaResourceDefinitionCondition contains details for the current condition of this pod.
type GrafanaResourceDefinitionCondition struct {
	// Type is the type of the condition. Types include Established, NamesAccepted and Terminating.
	Type GrafanaResourceDefinitionConditionType
	// Status is the status of the condition.
	// Can be True, False, Unknown.
	Status ConditionStatus
	// Last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time
	// Unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string
	// Human-readable message indicating details about last transition.
	// +optional
	Message string
}

// GrafanaResourceDefinitionStatus indicates the state of the GrafanaResourceDefinition
type GrafanaResourceDefinitionStatus struct {
	// Conditions indicate state for particular aspects of a GrafanaResourceDefinition
	// +listType=map
	// +listMapKey=type
	Conditions []GrafanaResourceDefinitionCondition

	// AcceptedNames are the names that are actually being used to serve discovery
	// They may be different than the names in spec.
	AcceptedNames GrafanaResourceDefinitionNames

	// StoredVersions are all versions of GrafanaResources that were ever persisted. Tracking these
	// versions allows a migration path for stored versions in etcd. The field is mutable
	// so the migration controller can first finish a migration to another version (i.e.
	// that no old objects are left in the storage), and then remove the rest of the
	// versions from this list.
	// None of the versions in this list can be removed from the spec.Versions field.
	StoredVersions []string
}

// GrafanaResourceCleanupFinalizer is the name of the finalizer which will delete instances of
// a GrafanaResourceDefinition
const GrafanaResourceCleanupFinalizer = "cleanup.kinds.grafana.com"

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GrafanaResourceDefinition represents a resource that should be exposed on the API server.  Its name MUST be in the format
// <.spec.name>.<.spec.group>.
type GrafanaResourceDefinition struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	// Spec describes how the user wants the resources to appear
	Spec GrafanaResourceDefinitionSpec
	// Status indicates the actual state of the GrafanaResourceDefinition
	Status GrafanaResourceDefinitionStatus
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GrafanaResourceDefinitionList is a list of GrafanaResourceDefinition objects.
type GrafanaResourceDefinitionList struct {
	metav1.TypeMeta
	metav1.ListMeta

	// Items individual GrafanaResourceDefinitions
	Items []GrafanaResourceDefinition
}

// GrafanaResourceSubresources defines the status and scale subresources for GrafanaResources.
type GrafanaResourceSubresources struct {
	// Status denotes the status subresource for GrafanaResources
	Status *GrafanaResourceSubresourceStatus
	// Scale denotes the scale subresource for GrafanaResources
	Scale *GrafanaResourceSubresourceScale

	History *GrafanaResourceSubresourceHistory
	Ref     *GrafanaResourceSubresourceRef
}

type GrafanaResourceSubresourceHistory struct{}
type GrafanaResourceSubresourceRef struct{}

// GrafanaResourceSubresourceStatus defines how to serve the status subresource for GrafanaResources.
// Status is represented by the `.status` JSON path inside of a GrafanaResource. When set,
// * exposes a /status subresource for the custom resource
// * PUT requests to the /status subresource take a custom resource object, and ignore changes to anything except the status stanza
// * PUT/POST/PATCH requests to the custom resource ignore changes to the status stanza
type GrafanaResourceSubresourceStatus struct{}

// GrafanaResourceSubresourceScale defines how to serve the scale subresource for GrafanaResources.
type GrafanaResourceSubresourceScale struct {
	// SpecReplicasPath defines the JSON path inside of a GrafanaResource that corresponds to Scale.Spec.Replicas.
	// Only JSON paths without the array notation are allowed.
	// Must be a JSON Path under .spec.
	// If there is no value under the given path in the GrafanaResource, the /scale subresource will return an error on GET.
	SpecReplicasPath string
	// StatusReplicasPath defines the JSON path inside of a GrafanaResource that corresponds to Scale.Status.Replicas.
	// Only JSON paths without the array notation are allowed.
	// Must be a JSON Path under .status.
	// If there is no value under the given path in the GrafanaResource, the status replica value in the /scale subresource
	// will default to 0.
	StatusReplicasPath string
	// LabelSelectorPath defines the JSON path inside of a GrafanaResource that corresponds to Scale.Status.Selector.
	// Only JSON paths without the array notation are allowed.
	// Must be a JSON Path under .status or .spec.
	// Must be set to work with HPA.
	// The field pointed by this JSON path must be a string field (not a complex selector struct)
	// which contains a serialized label selector in string form.
	// More info: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions#scale-subresource
	// If there is no value under the given path in the GrafanaResource, the status label selector value in the /scale
	// subresource will default to the empty string.
	// +optional
	LabelSelectorPath *string
}
