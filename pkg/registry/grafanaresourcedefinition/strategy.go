// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/registry/customresourcedefinition/strategy.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package grafanaresourcedefinition

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-apiserver/pkg/apis/kinds"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

// strategy implements behavior for GrafanaResources.
type strategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStrategy(typer runtime.ObjectTyper) strategy {
	return strategy{typer, names.SimpleNameGenerator}
}

func (strategy) NamespaceScoped() bool {
	return false
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (strategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	fields := map[fieldpath.APIVersion]*fieldpath.Set{
		"kinds.grafana.com/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		),
	}

	return fields
}

// PrepareForCreate clears the status of a GrafanaResourceDefinition before creation.
func (strategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	grd := obj.(*kinds.GrafanaResourceDefinition)
	grd.Status = kinds.GrafanaResourceDefinitionStatus{}
	grd.Generation = 1

	for _, v := range grd.Spec.Versions {
		if v.Storage {
			if !kinds.IsStoredVersion(grd, v.Name) {
				grd.Status.StoredVersions = append(grd.Status.StoredVersions, v.Name)
			}
			break
		}
	}

}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (strategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newGRD := obj.(*kinds.GrafanaResourceDefinition)
	oldGRD := old.(*kinds.GrafanaResourceDefinition)
	newGRD.Status = oldGRD.Status

	// Any changes to the spec increment the generation number, any changes to the
	// status should reflect the generation number of the corresponding object. We push
	// the burden of managing the status onto the clients because we can't (in general)
	// know here what version of spec the writer of the status has seen. It may seem like
	// we can at first -- since obj contains spec -- but in the future we will probably make
	// status its own object, and even if we don't, writes may be the result of a
	// read-update-write loop, so the contents of spec may not actually be the spec that
	// the controller has *seen*.
	if !apiequality.Semantic.DeepEqual(oldGRD.Spec, newGRD.Spec) {
		newGRD.Generation = oldGRD.Generation + 1
	}

	for _, v := range newGRD.Spec.Versions {
		if v.Storage {
			if !kinds.IsStoredVersion(newGRD, v.Name) {
				newGRD.Status.StoredVersions = append(newGRD.Status.StoredVersions, v.Name)
			}
			break
		}
	}

}

// Validate validates a new GrafanaResourceDefinition.
func (strategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return make(field.ErrorList, 0)
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (strategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string { return nil }

// AllowCreateOnUpdate is false for GrafanaResourceDefinition; this means a POST is
// needed to create one.
func (strategy) AllowCreateOnUpdate() bool {
	return false
}

// AllowUnconditionalUpdate is the default update policy for GrafanaResourceDefinition objects.
func (strategy) AllowUnconditionalUpdate() bool {
	return false
}

// Canonicalize normalizes the object after validation.
func (strategy) Canonicalize(obj runtime.Object) {
}

// ValidateUpdate is the default update validation for an end user updating status.
func (strategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return make(field.ErrorList, 0)
}

// WarningsOnUpdate returns warnings for the given update.
func (strategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

type statusStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func NewStatusStrategy(typer runtime.ObjectTyper) statusStrategy {
	return statusStrategy{typer, names.SimpleNameGenerator}
}

func (statusStrategy) NamespaceScoped() bool {
	return false
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (statusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	fields := map[fieldpath.APIVersion]*fieldpath.Set{
		"kinds.grafana.com/v1": fieldpath.NewSet(
			fieldpath.MakePathOrDie("metadata"),
			fieldpath.MakePathOrDie("spec"),
		),
	}

	return fields
}

func (statusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newObj := obj.(*kinds.GrafanaResourceDefinition)
	oldObj := old.(*kinds.GrafanaResourceDefinition)
	newObj.Spec = oldObj.Spec

	// Status updates are for only for updating status, not objectmeta.
	metav1.ResetObjectMetaForStatus(&newObj.ObjectMeta, &newObj.ObjectMeta)
}

func (statusStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (statusStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (statusStrategy) Canonicalize(obj runtime.Object) {
}

func (statusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return make(field.ErrorList, 0)
}

// WarningsOnUpdate returns warnings for the given update.
func (statusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	apiserver, ok := obj.(*kinds.GrafanaResourceDefinition)
	if !ok {
		return nil, nil, fmt.Errorf("given object is not a GrafanaResourceDefinition")
	}
	return labels.Set(apiserver.ObjectMeta.Labels), GrafanaResourceDefinitionToSelectableFields(apiserver), nil
}

// MatchGrafanaResourceDefinition is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func MatchGrafanaResourceDefinition(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// GrafanaResourceDefinitionToSelectableFields returns a field set that represents the object.
func GrafanaResourceDefinitionToSelectableFields(obj *kinds.GrafanaResourceDefinition) fields.Set {
	return generic.ObjectMetaFieldsSet(&obj.ObjectMeta, true)
}
