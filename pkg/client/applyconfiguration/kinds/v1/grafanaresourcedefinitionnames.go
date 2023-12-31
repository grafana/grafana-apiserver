// SPDX-License-Identifier: AGPL-3.0-only

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

// GrafanaResourceDefinitionNamesApplyConfiguration represents an declarative configuration of the GrafanaResourceDefinitionNames type for use
// with apply.
type GrafanaResourceDefinitionNamesApplyConfiguration struct {
	Plural     *string  `json:"plural,omitempty"`
	Singular   *string  `json:"singular,omitempty"`
	ShortNames []string `json:"shortNames,omitempty"`
	Kind       *string  `json:"kind,omitempty"`
	ListKind   *string  `json:"listKind,omitempty"`
	Categories []string `json:"categories,omitempty"`
}

// GrafanaResourceDefinitionNamesApplyConfiguration constructs an declarative configuration of the GrafanaResourceDefinitionNames type for use with
// apply.
func GrafanaResourceDefinitionNames() *GrafanaResourceDefinitionNamesApplyConfiguration {
	return &GrafanaResourceDefinitionNamesApplyConfiguration{}
}

// WithPlural sets the Plural field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Plural field is set to the value of the last call.
func (b *GrafanaResourceDefinitionNamesApplyConfiguration) WithPlural(value string) *GrafanaResourceDefinitionNamesApplyConfiguration {
	b.Plural = &value
	return b
}

// WithSingular sets the Singular field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Singular field is set to the value of the last call.
func (b *GrafanaResourceDefinitionNamesApplyConfiguration) WithSingular(value string) *GrafanaResourceDefinitionNamesApplyConfiguration {
	b.Singular = &value
	return b
}

// WithShortNames adds the given value to the ShortNames field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the ShortNames field.
func (b *GrafanaResourceDefinitionNamesApplyConfiguration) WithShortNames(values ...string) *GrafanaResourceDefinitionNamesApplyConfiguration {
	for i := range values {
		b.ShortNames = append(b.ShortNames, values[i])
	}
	return b
}

// WithKind sets the Kind field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Kind field is set to the value of the last call.
func (b *GrafanaResourceDefinitionNamesApplyConfiguration) WithKind(value string) *GrafanaResourceDefinitionNamesApplyConfiguration {
	b.Kind = &value
	return b
}

// WithListKind sets the ListKind field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ListKind field is set to the value of the last call.
func (b *GrafanaResourceDefinitionNamesApplyConfiguration) WithListKind(value string) *GrafanaResourceDefinitionNamesApplyConfiguration {
	b.ListKind = &value
	return b
}

// WithCategories adds the given value to the Categories field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the Categories field.
func (b *GrafanaResourceDefinitionNamesApplyConfiguration) WithCategories(values ...string) *GrafanaResourceDefinitionNamesApplyConfiguration {
	for i := range values {
		b.Categories = append(b.Categories, values[i])
	}
	return b
}
