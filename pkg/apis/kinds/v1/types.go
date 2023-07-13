// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/apis/apiextensions/v1/types.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package v1

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
	// group is the API group of the defined custom resource.
	// The custom resources are served under `/apis/<group>/...`.
	// Must match the name of the GrafanaResourceDefinition (in the form `<names.plural>.<group>`).
	Group string `json:"group" protobuf:"bytes,1,opt,name=group"`
	// names specify the resource and kind names for the custom resource.
	Names GrafanaResourceDefinitionNames `json:"names" protobuf:"bytes,3,opt,name=names"`
	// scope indicates whether the defined custom resource is cluster- or namespace-scoped.
	// Allowed values are `Cluster` and `Namespaced`.
	Scope ResourceScope `json:"scope" protobuf:"bytes,4,opt,name=scope,casttype=ResourceScope"`
	// versions is the list of all API versions of the defined custom resource.
	// Version names are used to compute the order in which served versions are listed in API discovery.
	// If the version string is "kube-like", it will sort above non "kube-like" version strings, which are ordered
	// lexicographically. "Kube-like" versions start with a "v", then are followed by a number (the major version),
	// then optionally the string "alpha" or "beta" and another number (the minor version). These are sorted first
	// by GA > beta > alpha (where GA is a version with no suffix such as beta or alpha), and then by comparing
	// major version, then minor version. An example sorted list of versions:
	// v10, v2, v1, v11beta2, v10beta3, v3beta1, v12alpha1, v11alpha2, foo1, foo10.
	Versions []GrafanaResourceDefinitionVersion `json:"versions" protobuf:"bytes,7,rep,name=versions"`

	// preserveUnknownFields indicates that object fields which are not specified
	// in the OpenAPI schema should be preserved when persisting to storage.
	// apiVersion, kind, metadata and known fields inside metadata are always preserved.
	// This field is deprecated in favor of setting `x-preserve-unknown-fields` to true in `spec.versions[*].schema.openAPIV3Schema`.
	// See https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#field-pruning for details.
	// +optional
	PreserveUnknownFields bool `json:"preserveUnknownFields,omitempty" protobuf:"varint,10,opt,name=preserveUnknownFields"`
}

// GrafanaResourceDefinitionVersion describes a version for GRD.
type GrafanaResourceDefinitionVersion struct {
	// name is the version name, e.g. “v1”, “v2beta1”, etc.
	// The custom resources are served under this version at `/apis/<group>/<version>/...` if `served` is true.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// served is a flag enabling/disabling this version from being served via REST APIs
	Served bool `json:"served" protobuf:"varint,2,opt,name=served"`
	// storage indicates this version should be used when persisting custom resources to storage.
	// There must be exactly one version with storage=true.
	Storage bool `json:"storage" protobuf:"varint,3,opt,name=storage"`
	// deprecated indicates this version of the custom resource API is deprecated.
	// When set to true, API requests to this version receive a warning header in the server response.
	// Defaults to false.
	// +optional
	Deprecated bool `json:"deprecated,omitempty" protobuf:"varint,7,opt,name=deprecated"`
	// deprecationWarning overrides the default warning returned to API clients.
	// May only be set when `deprecated` is true.
	// The default warning indicates this version is deprecated and recommends use
	// of the newest served version of equal or greater stability, if one exists.
	// +optional
	DeprecationWarning *string `json:"deprecationWarning,omitempty" protobuf:"bytes,8,opt,name=deprecationWarning"`
	// subresources specify what subresources this version of the defined custom resource have.
	// +optional
	Subresources *GrafanaResourceSubresources `json:"subresources,omitempty" protobuf:"bytes,5,opt,name=subresources"`
}

// GrafanaResourceDefinitionNames indicates the names to serve this GrafanaResourceDefinition
type GrafanaResourceDefinitionNames struct {
	// plural is the plural name of the resource to serve.
	// The custom resources are served under `/apis/<group>/<version>/.../<plural>`.
	// Must match the name of the GrafanaResourceDefinition (in the form `<names.plural>.<group>`).
	// Must be all lowercase.
	Plural string `json:"plural" protobuf:"bytes,1,opt,name=plural"`
	// singular is the singular name of the resource. It must be all lowercase. Defaults to lowercased `kind`.
	// +optional
	Singular string `json:"singular,omitempty" protobuf:"bytes,2,opt,name=singular"`
	// shortNames are short names for the resource, exposed in API discovery documents,
	// and used by clients to support invocations like `kubectl get <shortname>`.
	// It must be all lowercase.
	// +optional
	ShortNames []string `json:"shortNames,omitempty" protobuf:"bytes,3,opt,name=shortNames"`
	// kind is the serialized kind of the resource. It is normally CamelCase and singular.
	// Custom resource instances will use this value as the `kind` attribute in API calls.
	Kind string `json:"kind" protobuf:"bytes,4,opt,name=kind"`
	// listKind is the serialized kind of the list for this resource. Defaults to "`kind`List".
	// +optional
	ListKind string `json:"listKind,omitempty" protobuf:"bytes,5,opt,name=listKind"`
	// categories is a list of grouped resources this custom resource belongs to (e.g. 'all').
	// This is published in API discovery documents, and used by clients to support invocations like
	// `kubectl get all`.
	// +optional
	Categories []string `json:"categories,omitempty" protobuf:"bytes,6,rep,name=categories"`
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
	// type is the type of the condition. Types include Established, NamesAccepted and Terminating.
	Type GrafanaResourceDefinitionConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=GrafanaResourceDefinitionConditionType"`
	// status is the status of the condition.
	// Can be True, False, Unknown.
	Status ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=ConditionStatus"`
	// lastTransitionTime last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,3,opt,name=lastTransitionTime"`
	// reason is a unique, one-word, CamelCase reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`
	// message is a human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
}

// GrafanaResourceDefinitionStatus indicates the state of the GrafanaResourceDefinition
type GrafanaResourceDefinitionStatus struct {
	// conditions indicate state for particular aspects of a GrafanaResourceDefinition
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []GrafanaResourceDefinitionCondition `json:"conditions" protobuf:"bytes,1,opt,name=conditions"`

	// acceptedNames are the names that are actually being used to serve discovery.
	// They may be different than the names in spec.
	// +optional
	AcceptedNames GrafanaResourceDefinitionNames `json:"acceptedNames" protobuf:"bytes,2,opt,name=acceptedNames"`

	// storedVersions lists all versions of GrafanaResources that were ever persisted. Tracking these
	// versions allows a migration path for stored versions in etcd. The field is mutable
	// so a migration controller can finish a migration to another version (ensuring
	// no old objects are left in storage), and then remove the rest of the
	// versions from this list.
	// Versions may not be removed from `spec.versions` while they exist in this list.
	// +optional
	StoredVersions []string `json:"storedVersions" protobuf:"bytes,3,rep,name=storedVersions"`
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
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// spec describes how the user wants the resources to appear
	Spec GrafanaResourceDefinitionSpec `json:"spec" protobuf:"bytes,2,opt,name=spec"`
	// status indicates the actual state of the GrafanaResourceDefinition
	// +optional
	Status GrafanaResourceDefinitionStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GrafanaResourceDefinitionList is a list of GrafanaResourceDefinition objects.
type GrafanaResourceDefinitionList struct {
	metav1.TypeMeta `json:",inline"`

	// Standard object's metadata
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// items list individual GrafanaResourceDefinition objects
	Items []GrafanaResourceDefinition `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// GrafanaResourceSubresources defines the status and scale subresources for GrafanaResources.
type GrafanaResourceSubresources struct {
	// status indicates the custom resource should serve a `/status` subresource.
	// When enabled:
	// 1. requests to the custom resource primary endpoint ignore changes to the `status` stanza of the object.
	// 2. requests to the custom resource `/status` subresource ignore changes to anything other than the `status` stanza of the object.
	// +optional
	Status *GrafanaResourceSubresourceStatus `json:"status,omitempty" protobuf:"bytes,1,opt,name=status"`
	// scale indicates the custom resource should serve a `/scale` subresource that returns an `autoscaling/v1` Scale object.
	// +optional
	Scale *GrafanaResourceSubresourceScale `json:"scale,omitempty" protobuf:"bytes,2,opt,name=scale"`
	// +optional
	History *GrafanaResourceSubresourceHistory `json:"history,omitempty" protobuf:"bytes,3,opt,name=history"`
	// +optional
	Ref *GrafanaResourceSubresourceRef `json:"ref,omitempty" protobuf:"bytes,4,opt,name=ref"`
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
	// specReplicasPath defines the JSON path inside of a custom resource that corresponds to Scale `spec.replicas`.
	// Only JSON paths without the array notation are allowed.
	// Must be a JSON Path under `.spec`.
	// If there is no value under the given path in the custom resource, the `/scale` subresource will return an error on GET.
	SpecReplicasPath string `json:"specReplicasPath" protobuf:"bytes,1,name=specReplicasPath"`
	// statusReplicasPath defines the JSON path inside of a custom resource that corresponds to Scale `status.replicas`.
	// Only JSON paths without the array notation are allowed.
	// Must be a JSON Path under `.status`.
	// If there is no value under the given path in the custom resource, the `status.replicas` value in the `/scale` subresource
	// will default to 0.
	StatusReplicasPath string `json:"statusReplicasPath" protobuf:"bytes,2,opt,name=statusReplicasPath"`
	// labelSelectorPath defines the JSON path inside of a custom resource that corresponds to Scale `status.selector`.
	// Only JSON paths without the array notation are allowed.
	// Must be a JSON Path under `.status` or `.spec`.
	// Must be set to work with HorizontalPodAutoscaler.
	// The field pointed by this JSON path must be a string field (not a complex selector struct)
	// which contains a serialized label selector in string form.
	// More info: https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions#scale-subresource
	// If there is no value under the given path in the custom resource, the `status.selector` value in the `/scale`
	// subresource will default to the empty string.
	// +optional
	LabelSelectorPath *string `json:"labelSelectorPath,omitempty" protobuf:"bytes,3,opt,name=labelSelectorPath"`
}
