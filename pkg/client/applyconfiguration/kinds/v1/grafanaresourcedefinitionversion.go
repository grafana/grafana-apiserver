// SPDX-License-Identifier: AGPL-3.0-only

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// GrafanaResourceDefinitionVersionApplyConfiguration represents an declarative configuration of the GrafanaResourceDefinitionVersion type for use
// with apply.
type GrafanaResourceDefinitionVersionApplyConfiguration struct {
	Name               *string                                        `json:"name,omitempty"`
	Served             *bool                                          `json:"served,omitempty"`
	Storage            *bool                                          `json:"storage,omitempty"`
	Deprecated         *bool                                          `json:"deprecated,omitempty"`
	DeprecationWarning *string                                        `json:"deprecationWarning,omitempty"`
	Subresources       *GrafanaResourceSubresourcesApplyConfiguration `json:"subresources,omitempty"`
}

// GrafanaResourceDefinitionVersionApplyConfiguration constructs an declarative configuration of the GrafanaResourceDefinitionVersion type for use with
// apply.
func GrafanaResourceDefinitionVersion() *GrafanaResourceDefinitionVersionApplyConfiguration {
	return &GrafanaResourceDefinitionVersionApplyConfiguration{}
}

// WithName sets the Name field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Name field is set to the value of the last call.
func (b *GrafanaResourceDefinitionVersionApplyConfiguration) WithName(value string) *GrafanaResourceDefinitionVersionApplyConfiguration {
	b.Name = &value
	return b
}

// WithServed sets the Served field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Served field is set to the value of the last call.
func (b *GrafanaResourceDefinitionVersionApplyConfiguration) WithServed(value bool) *GrafanaResourceDefinitionVersionApplyConfiguration {
	b.Served = &value
	return b
}

// WithStorage sets the Storage field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Storage field is set to the value of the last call.
func (b *GrafanaResourceDefinitionVersionApplyConfiguration) WithStorage(value bool) *GrafanaResourceDefinitionVersionApplyConfiguration {
	b.Storage = &value
	return b
}

// WithDeprecated sets the Deprecated field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Deprecated field is set to the value of the last call.
func (b *GrafanaResourceDefinitionVersionApplyConfiguration) WithDeprecated(value bool) *GrafanaResourceDefinitionVersionApplyConfiguration {
	b.Deprecated = &value
	return b
}

// WithDeprecationWarning sets the DeprecationWarning field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the DeprecationWarning field is set to the value of the last call.
func (b *GrafanaResourceDefinitionVersionApplyConfiguration) WithDeprecationWarning(value string) *GrafanaResourceDefinitionVersionApplyConfiguration {
	b.DeprecationWarning = &value
	return b
}

// WithSubresources sets the Subresources field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Subresources field is set to the value of the last call.
func (b *GrafanaResourceDefinitionVersionApplyConfiguration) WithSubresources(value *GrafanaResourceSubresourcesApplyConfiguration) *GrafanaResourceDefinitionVersionApplyConfiguration {
	b.Subresources = value
	return b
}