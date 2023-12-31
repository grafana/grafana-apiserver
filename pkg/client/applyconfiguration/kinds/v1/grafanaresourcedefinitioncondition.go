// SPDX-License-Identifier: AGPL-3.0-only

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GrafanaResourceDefinitionConditionApplyConfiguration represents an declarative configuration of the GrafanaResourceDefinitionCondition type for use
// with apply.
type GrafanaResourceDefinitionConditionApplyConfiguration struct {
	Type               *v1.GrafanaResourceDefinitionConditionType `json:"type,omitempty"`
	Status             *v1.ConditionStatus                        `json:"status,omitempty"`
	LastTransitionTime *metav1.Time                               `json:"lastTransitionTime,omitempty"`
	Reason             *string                                    `json:"reason,omitempty"`
	Message            *string                                    `json:"message,omitempty"`
}

// GrafanaResourceDefinitionConditionApplyConfiguration constructs an declarative configuration of the GrafanaResourceDefinitionCondition type for use with
// apply.
func GrafanaResourceDefinitionCondition() *GrafanaResourceDefinitionConditionApplyConfiguration {
	return &GrafanaResourceDefinitionConditionApplyConfiguration{}
}

// WithType sets the Type field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Type field is set to the value of the last call.
func (b *GrafanaResourceDefinitionConditionApplyConfiguration) WithType(value v1.GrafanaResourceDefinitionConditionType) *GrafanaResourceDefinitionConditionApplyConfiguration {
	b.Type = &value
	return b
}

// WithStatus sets the Status field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Status field is set to the value of the last call.
func (b *GrafanaResourceDefinitionConditionApplyConfiguration) WithStatus(value v1.ConditionStatus) *GrafanaResourceDefinitionConditionApplyConfiguration {
	b.Status = &value
	return b
}

// WithLastTransitionTime sets the LastTransitionTime field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the LastTransitionTime field is set to the value of the last call.
func (b *GrafanaResourceDefinitionConditionApplyConfiguration) WithLastTransitionTime(value metav1.Time) *GrafanaResourceDefinitionConditionApplyConfiguration {
	b.LastTransitionTime = &value
	return b
}

// WithReason sets the Reason field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Reason field is set to the value of the last call.
func (b *GrafanaResourceDefinitionConditionApplyConfiguration) WithReason(value string) *GrafanaResourceDefinitionConditionApplyConfiguration {
	b.Reason = &value
	return b
}

// WithMessage sets the Message field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the Message field is set to the value of the last call.
func (b *GrafanaResourceDefinitionConditionApplyConfiguration) WithMessage(value string) *GrafanaResourceDefinitionConditionApplyConfiguration {
	b.Message = &value
	return b
}
