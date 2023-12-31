// SPDX-License-Identifier: AGPL-3.0-only

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// GrafanaResourceSubresourceScaleApplyConfiguration represents an declarative configuration of the GrafanaResourceSubresourceScale type for use
// with apply.
type GrafanaResourceSubresourceScaleApplyConfiguration struct {
	SpecReplicasPath   *string `json:"specReplicasPath,omitempty"`
	StatusReplicasPath *string `json:"statusReplicasPath,omitempty"`
	LabelSelectorPath  *string `json:"labelSelectorPath,omitempty"`
}

// GrafanaResourceSubresourceScaleApplyConfiguration constructs an declarative configuration of the GrafanaResourceSubresourceScale type for use with
// apply.
func GrafanaResourceSubresourceScale() *GrafanaResourceSubresourceScaleApplyConfiguration {
	return &GrafanaResourceSubresourceScaleApplyConfiguration{}
}

// WithSpecReplicasPath sets the SpecReplicasPath field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the SpecReplicasPath field is set to the value of the last call.
func (b *GrafanaResourceSubresourceScaleApplyConfiguration) WithSpecReplicasPath(value string) *GrafanaResourceSubresourceScaleApplyConfiguration {
	b.SpecReplicasPath = &value
	return b
}

// WithStatusReplicasPath sets the StatusReplicasPath field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the StatusReplicasPath field is set to the value of the last call.
func (b *GrafanaResourceSubresourceScaleApplyConfiguration) WithStatusReplicasPath(value string) *GrafanaResourceSubresourceScaleApplyConfiguration {
	b.StatusReplicasPath = &value
	return b
}

// WithLabelSelectorPath sets the LabelSelectorPath field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the LabelSelectorPath field is set to the value of the last call.
func (b *GrafanaResourceSubresourceScaleApplyConfiguration) WithLabelSelectorPath(value string) *GrafanaResourceSubresourceScaleApplyConfiguration {
	b.LabelSelectorPath = &value
	return b
}
