// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/apihelpers/helpers.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package apihelpers

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	kindsv1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsProtectedCommunityGroup returns whether or not a group specified for a GRD is protected for the community and needs
// to have the v1beta1.KubeAPIApprovalAnnotation set.
func IsProtectedCommunityGroup(group string) bool {
	switch {
	case group == "k8s.io" || strings.HasSuffix(group, ".k8s.io"):
		return true
	case group == "kubernetes.io" || strings.HasSuffix(group, ".kubernetes.io"):
		return true
	default:
		return false
	}

}

// APIApprovalState covers the various options for API approval annotation states
type APIApprovalState int

const (
	// APIApprovalInvalid means the annotation doesn't have an expected value
	APIApprovalInvalid APIApprovalState = iota
	// APIApproved if the annotation has a URL (this means the API is approved)
	APIApproved
	// APIApprovalBypassed if the annotation starts with "unapproved" indicating that for whatever reason the API isn't approved, but we should allow its creation
	APIApprovalBypassed
	// APIApprovalMissing means the annotation is empty
	APIApprovalMissing
)

// GetAPIApprovalState returns the state of the API approval and reason for that state
func GetAPIApprovalState(annotations map[string]string) (state APIApprovalState, reason string) {
	annotation := annotations[kindsv1.KubeAPIApprovedAnnotation]

	// we use the result of this parsing in the switch/case below
	url, annotationURLParseErr := url.ParseRequestURI(annotation)
	switch {
	case len(annotation) == 0:
		return APIApprovalMissing, fmt.Sprintf("protected groups must have approval annotation %q, see https://github.com/kubernetes/enhancements/pull/1111", kindsv1.KubeAPIApprovedAnnotation)
	case strings.HasPrefix(annotation, "unapproved"):
		return APIApprovalBypassed, fmt.Sprintf("not approved: %q", annotation)
	case annotationURLParseErr == nil && url != nil && len(url.Host) > 0 && len(url.Scheme) > 0:
		return APIApproved, fmt.Sprintf("approved in %v", annotation)
	default:
		return APIApprovalInvalid, fmt.Sprintf("protected groups must have approval annotation %q with either a URL or a reason starting with \"unapproved\", see https://github.com/kubernetes/enhancements/pull/1111", kindsv1.KubeAPIApprovedAnnotation)
	}
}

// SetGRDCondition sets the status condition. It either overwrites the existing one or creates a new one.
func SetGRDCondition(grd *kindsv1.GrafanaResourceDefinition, newCondition kindsv1.GrafanaResourceDefinitionCondition) {
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
func RemoveGRDCondition(grd *kindsv1.GrafanaResourceDefinition, conditionType kindsv1.GrafanaResourceDefinitionConditionType) {
	newConditions := []kindsv1.GrafanaResourceDefinitionCondition{}
	for _, condition := range grd.Status.Conditions {
		if condition.Type != conditionType {
			newConditions = append(newConditions, condition)
		}
	}
	grd.Status.Conditions = newConditions
}

// FindGRDCondition returns the condition you're looking for or nil.
func FindGRDCondition(grd *kindsv1.GrafanaResourceDefinition, conditionType kindsv1.GrafanaResourceDefinitionConditionType) *kindsv1.GrafanaResourceDefinitionCondition {
	for i := range grd.Status.Conditions {
		if grd.Status.Conditions[i].Type == conditionType {
			return &grd.Status.Conditions[i]
		}
	}

	return nil
}

// IsGRDConditionTrue indicates if the condition is present and strictly true.
func IsGRDConditionTrue(grd *kindsv1.GrafanaResourceDefinition, conditionType kindsv1.GrafanaResourceDefinitionConditionType) bool {
	return IsGRDConditionPresentAndEqual(grd, conditionType, kindsv1.ConditionTrue)
}

// IsGRDConditionFalse indicates if the condition is present and false.
func IsGRDConditionFalse(grd *kindsv1.GrafanaResourceDefinition, conditionType kindsv1.GrafanaResourceDefinitionConditionType) bool {
	return IsGRDConditionPresentAndEqual(grd, conditionType, kindsv1.ConditionFalse)
}

// IsGRDConditionPresentAndEqual indicates if the condition is present and equal to the given status.
func IsGRDConditionPresentAndEqual(grd *kindsv1.GrafanaResourceDefinition, conditionType kindsv1.GrafanaResourceDefinitionConditionType, status kindsv1.ConditionStatus) bool {
	for _, condition := range grd.Status.Conditions {
		if condition.Type == conditionType {
			return condition.Status == status
		}
	}
	return false
}

// IsGRDConditionEquivalent returns true if the lhs and rhs are equivalent except for times.
func IsGRDConditionEquivalent(lhs, rhs *kindsv1.GrafanaResourceDefinitionCondition) bool {
	if lhs == nil && rhs == nil {
		return true
	}
	if lhs == nil || rhs == nil {
		return false
	}

	return lhs.Message == rhs.Message && lhs.Reason == rhs.Reason && lhs.Status == rhs.Status && lhs.Type == rhs.Type
}

// GRDHasFinalizer returns true if the finalizer is in the list.
func GRDHasFinalizer(grd *kindsv1.GrafanaResourceDefinition, needle string) bool {
	for _, finalizer := range grd.Finalizers {
		if finalizer == needle {
			return true
		}
	}

	return false
}

// GRDRemoveFinalizer removes the finalizer if present.
func GRDRemoveFinalizer(grd *kindsv1.GrafanaResourceDefinition, needle string) {
	newFinalizers := []string{}
	for _, finalizer := range grd.Finalizers {
		if finalizer != needle {
			newFinalizers = append(newFinalizers, finalizer)
		}
	}
	grd.Finalizers = newFinalizers
}

// HasServedGRDVersion returns true if the given version is in the list of GRD's versions and the Served flag is set.
func HasServedGRDVersion(grd *kindsv1.GrafanaResourceDefinition, version string) bool {
	for _, v := range grd.Spec.Versions {
		if v.Name == version {
			return v.Served
		}
	}
	return false
}

// GetGRDStorageVersion returns the storage version for given GRD.
func GetGRDStorageVersion(grd *kindsv1.GrafanaResourceDefinition) (string, error) {
	for _, v := range grd.Spec.Versions {
		if v.Storage {
			return v.Name, nil
		}
	}
	// This should not happened if grd is valid
	return "", fmt.Errorf("invalid kindsv1.GrafanaResourceDefinition, no storage version")
}

// IsStoredVersion returns whether the given version is the storage version of the GRD.
func IsStoredVersion(grd *kindsv1.GrafanaResourceDefinition, version string) bool {
	for _, v := range grd.Status.StoredVersions {
		if version == v {
			return true
		}
	}
	return false
}

func GetSubresourcesForVersion(grd *kindsv1.GrafanaResourceDefinition, version string) (*kindsv1.GrafanaResourceSubresources, error) {
	for _, v := range grd.Spec.Versions {
		if version == v.Name {
			return v.Subresources, nil
		}
	}
	return nil, fmt.Errorf("version %s not found in kindsv1.GrafanaResourceDefinition: %v", version, grd.Name)
}

// HasPerVersionSchema returns true if a GRD uses per-version schema.
func HasPerVersionSchema(versions []kindsv1.GrafanaResourceDefinitionVersion) bool {
	return false
}

// HasPerVersionSubresources returns true if a GRD uses per-version subresources.
func HasPerVersionSubresources(versions []kindsv1.GrafanaResourceDefinitionVersion) bool {
	for _, v := range versions {
		if v.Subresources != nil {
			return true
		}
	}
	return false
}

// HasPerVersionColumns returns true if a GRD uses per-version columns.
func HasPerVersionColumns(versions []kindsv1.GrafanaResourceDefinitionVersion) bool {
	return false
}

// HasVersionServed returns true if given GRD has given version served.
func HasVersionServed(grd *kindsv1.GrafanaResourceDefinition, version string) bool {
	for _, v := range grd.Spec.Versions {
		if !v.Served || v.Name != version {
			continue
		}
		return true
	}
	return false
}
