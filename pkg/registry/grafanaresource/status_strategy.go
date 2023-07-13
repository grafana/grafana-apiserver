// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/registry/customresource/status_strategy.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package grafanaresource

import (
	"context"

	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

type statusStrategy struct {
	grafanaResourceStrategy
}

func NewStatusStrategy(strategy grafanaResourceStrategy) statusStrategy {
	return statusStrategy{strategy}
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (a statusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	fields := map[fieldpath.APIVersion]*fieldpath.Set{
		fieldpath.APIVersion(a.grafanaResourceStrategy.kind.GroupVersion().String()): fieldpath.NewSet(
			// Note that if there are other top level fields unique to GRDs,
			// those will also get removed by the apiserver prior to persisting,
			// but won't be added to the resetFields set.

			// This isn't an issue now, but if it becomes an issue in the future
			// we might need a mechanism that is the inverse of resetFields where
			// you specify only the fields to be kept rather than the fields to be wiped
			// that way you could wipe everything but the status in this case.
			fieldpath.MakePathOrDie("spec"),
		),
	}

	return fields
}

func (a statusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	// update is only allowed to set status
	newGrafanaResourceObject := obj.(*unstructured.Unstructured)
	newGrafanaResource := newGrafanaResourceObject.UnstructuredContent()
	status, ok := newGrafanaResource["status"]

	// managedFields must be preserved since it's been modified to
	// track changed fields in the status update.
	managedFields := newGrafanaResourceObject.GetManagedFields()

	// copy old object into new object
	oldGrafanaResourceObject := old.(*unstructured.Unstructured)
	// overridding the resourceVersion in metadata is safe here, we have already checked that
	// new object and old object have the same resourceVersion.
	*newGrafanaResourceObject = *oldGrafanaResourceObject.DeepCopy()

	// set status
	newGrafanaResourceObject.SetManagedFields(managedFields)
	newGrafanaResource = newGrafanaResourceObject.UnstructuredContent()
	if ok {
		newGrafanaResource["status"] = status
	} else {
		delete(newGrafanaResource, "status")
	}
}

// ValidateUpdate is the default update validation for an end user updating status.
func (a statusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return make(field.ErrorList, 0)
}

// WarningsOnUpdate returns warnings for the given update.
func (statusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
