// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/apis/apiextensions/helpers.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package kinds

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var swaggerMetadataDescriptions = metav1.ObjectMeta{}.SwaggerDoc()

// SetGRDCondition sets the status condition. It either overwrites the existing one or creates a new one.
func SetGRDCondition(grd *GrafanaResourceDefinition, newCondition GrafanaResourceDefinitionCondition) {
	newCondition.LastTransitionTime = metav1.NewTime(time.Now())

	existingCondition := FindGRDCondition(grd, newCondition.Type)
	if existingCondition == nil {
		grd.Status.Conditions = append(grd.Status.Conditions, newCondition)
		return
	}

	if existingCondition.Status != newCondition.Status || existingCondition.LastTransitionTime.IsZero() {
		existingCondition.LastTransitionTime = newCondition.LastTransitionTime
	}

	existingCondition.Status = newCondition.Status
	existingCondition.Reason = newCondition.Reason
	existingCondition.Message = newCondition.Message
}

// RemoveGRDCondition removes the status condition.
func RemoveGRDCondition(grd *GrafanaResourceDefinition, conditionType GrafanaResourceDefinitionConditionType) {
	newConditions := []GrafanaResourceDefinitionCondition{}
	for _, condition := range grd.Status.Conditions {
		if condition.Type != conditionType {
			newConditions = append(newConditions, condition)
		}
	}
	grd.Status.Conditions = newConditions
}

// FindGRDCondition returns the condition you're looking for or nil.
func FindGRDCondition(grd *GrafanaResourceDefinition, conditionType GrafanaResourceDefinitionConditionType) *GrafanaResourceDefinitionCondition {
	for i := range grd.Status.Conditions {
		if grd.Status.Conditions[i].Type == conditionType {
			return &grd.Status.Conditions[i]
		}
	}

	return nil
}

// IsGRDConditionTrue indicates if the condition is present and strictly true.
func IsGRDConditionTrue(grd *GrafanaResourceDefinition, conditionType GrafanaResourceDefinitionConditionType) bool {
	return IsGRDConditionPresentAndEqual(grd, conditionType, ConditionTrue)
}

// IsGRDConditionFalse indicates if the condition is present and false.
func IsGRDConditionFalse(grd *GrafanaResourceDefinition, conditionType GrafanaResourceDefinitionConditionType) bool {
	return IsGRDConditionPresentAndEqual(grd, conditionType, ConditionFalse)
}

// IsGRDConditionPresentAndEqual indicates if the condition is present and equal to the given status.
func IsGRDConditionPresentAndEqual(grd *GrafanaResourceDefinition, conditionType GrafanaResourceDefinitionConditionType, status ConditionStatus) bool {
	for _, condition := range grd.Status.Conditions {
		if condition.Type == conditionType {
			return condition.Status == status
		}
	}
	return false
}

// IsGRDConditionEquivalent returns true if the lhs and rhs are equivalent except for times.
func IsGRDConditionEquivalent(lhs, rhs *GrafanaResourceDefinitionCondition) bool {
	if lhs == nil && rhs == nil {
		return true
	}
	if lhs == nil || rhs == nil {
		return false
	}

	return lhs.Message == rhs.Message && lhs.Reason == rhs.Reason && lhs.Status == rhs.Status && lhs.Type == rhs.Type
}

// GRDHasFinalizer returns true if the finalizer is in the list.
func GRDHasFinalizer(grd *GrafanaResourceDefinition, needle string) bool {
	for _, finalizer := range grd.Finalizers {
		if finalizer == needle {
			return true
		}
	}

	return false
}

// GRDRemoveFinalizer removes the finalizer if present.
func GRDRemoveFinalizer(grd *GrafanaResourceDefinition, needle string) {
	newFinalizers := []string{}
	for _, finalizer := range grd.Finalizers {
		if finalizer != needle {
			newFinalizers = append(newFinalizers, finalizer)
		}
	}
	grd.Finalizers = newFinalizers
}

// HasServedGRDVersion returns true if the given version is in the list of GRD's versions and the Served flag is set.
func HasServedGRDVersion(grd *GrafanaResourceDefinition, version string) bool {
	for _, v := range grd.Spec.Versions {
		if v.Name == version {
			return v.Served
		}
	}
	return false
}

// GetGRDStorageVersion returns the storage version for given GRD.
func GetGRDStorageVersion(grd *GrafanaResourceDefinition) (string, error) {
	for _, v := range grd.Spec.Versions {
		if v.Storage {
			return v.Name, nil
		}
	}
	// This should not happened if grd is valid
	return "", fmt.Errorf("invalid GrafanaResourceDefinition, no storage version")
}

// IsStoredVersion returns whether the given version is the storage version of the GRD.
func IsStoredVersion(grd *GrafanaResourceDefinition, version string) bool {
	for _, v := range grd.Status.StoredVersions {
		if version == v {
			return true
		}
	}
	return false
}

// GetSubresourcesForVersion returns the subresources for given version or nil.
func GetSubresourcesForVersion(grd *GrafanaResourceDefinition, version string) (*GrafanaResourceSubresources, error) {
	if !HasPerVersionSubresources(grd.Spec.Versions) {
		return grd.Spec.Subresources, nil
	}
	if grd.Spec.Subresources != nil {
		return nil, fmt.Errorf("malformed GrafanaResourceDefinition %s version %s: top-level and per-version subresources must be mutual exclusive", grd.Name, version)
	}
	for _, v := range grd.Spec.Versions {
		if version == v.Name {
			return v.Subresources, nil
		}
	}
	return nil, fmt.Errorf("version %s not found in GrafanaResourceDefinition: %v", version, grd.Name)
}

// HasPerVersionSchema returns true if a GRD uses per-version schema.
func HasPerVersionSchema(versions []GrafanaResourceDefinitionVersion) bool {
	return false
}

// HasPerVersionSubresources returns true if a GRD uses per-version subresources.
func HasPerVersionSubresources(versions []GrafanaResourceDefinitionVersion) bool {
	for _, v := range versions {
		if v.Subresources != nil {
			return true
		}
	}
	return false
}

// HasPerVersionColumns returns true if a GRD uses per-version columns.
func HasPerVersionColumns(versions []GrafanaResourceDefinitionVersion) bool {
	return false
}

// HasVersionServed returns true if given GRD has given version served.
func HasVersionServed(grd *GrafanaResourceDefinition, version string) bool {
	for _, v := range grd.Spec.Versions {
		if !v.Served || v.Name != version {
			continue
		}
		return true
	}
	return false
}
