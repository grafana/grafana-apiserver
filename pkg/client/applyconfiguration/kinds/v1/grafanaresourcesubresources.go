// SPDX-License-Identifier: AGPL-3.0-only

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
)

// GrafanaResourceSubresourcesApplyConfiguration represents an declarative configuration of the GrafanaResourceSubresources type for use
// with apply.
type GrafanaResourceSubresourcesApplyConfiguration struct {
	Status  *v1.GrafanaResourceSubresourceStatus               `json:"status,omitempty"`
	Scale   *GrafanaResourceSubresourceScaleApplyConfiguration `json:"scale,omitempty"`
	History *v1.GrafanaResourceSubresourceHistory              `json:"history,omitempty"`
	Ref     *v1.GrafanaResourceSubresourceRef                  `json:"ref,omitempty"`
}

// GrafanaResourceSubresourcesApplyConfiguration constructs an declarative configuration of the GrafanaResourceSubresources type for use with
// apply.
func GrafanaResourceSubresources() *GrafanaResourceSubresourcesApplyConfiguration {
	return &GrafanaResourceSubresourcesApplyConfiguration{}
}

// WithStatus sets the Status field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Status field is set to the value of the last call.
func (b *GrafanaResourceSubresourcesApplyConfiguration) WithStatus(value v1.GrafanaResourceSubresourceStatus) *GrafanaResourceSubresourcesApplyConfiguration {
	b.Status = &value
	return b
}

// WithScale sets the Scale field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Scale field is set to the value of the last call.
func (b *GrafanaResourceSubresourcesApplyConfiguration) WithScale(value *GrafanaResourceSubresourceScaleApplyConfiguration) *GrafanaResourceSubresourcesApplyConfiguration {
	b.Scale = value
	return b
}

// WithHistory sets the History field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the History field is set to the value of the last call.
func (b *GrafanaResourceSubresourcesApplyConfiguration) WithHistory(value v1.GrafanaResourceSubresourceHistory) *GrafanaResourceSubresourcesApplyConfiguration {
	b.History = &value
	return b
}

// WithRef sets the Ref field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Ref field is set to the value of the last call.
func (b *GrafanaResourceSubresourcesApplyConfiguration) WithRef(value v1.GrafanaResourceSubresourceRef) *GrafanaResourceSubresourcesApplyConfiguration {
	b.Ref = &value
	return b
}
