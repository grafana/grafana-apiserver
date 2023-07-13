// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/registry/customresource/strategy.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package grafanaresource

import (
	"context"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	apiserverstorage "k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"

	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

// grafanaResourceStrategy implements behavior for GrafanaResources.
type grafanaResourceStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator

	namespaceScoped bool
	status          bool
	scale           bool
	ref             bool
	history         bool

	kind schema.GroupVersionKind
}

func NewStrategy(typer runtime.ObjectTyper, namespaceScoped bool, kind schema.GroupVersionKind, status, scale, ref, history bool) grafanaResourceStrategy {
	return grafanaResourceStrategy{
		ObjectTyper:     typer,
		NameGenerator:   names.SimpleNameGenerator,
		namespaceScoped: namespaceScoped,

		status:  status,
		scale:   scale,
		ref:     ref,
		history: history,

		kind: kind,
	}
}

func (a grafanaResourceStrategy) NamespaceScoped() bool {
	return a.namespaceScoped
}

// GetResetFields returns the set of fields that get reset by the strategy
// and should not be modified by the user.
func (a grafanaResourceStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	fields := map[fieldpath.APIVersion]*fieldpath.Set{}

	if a.status {
		fields[fieldpath.APIVersion(a.kind.GroupVersion().String())] = fieldpath.NewSet(
			fieldpath.MakePathOrDie("status"),
		)
	}

	return fields
}

// PrepareForCreate clears the status of a GrafanaResource before creation.
func (a grafanaResourceStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	if a.status {
		grafanaResourceObject := obj.(*unstructured.Unstructured)
		grafanaResource := grafanaResourceObject.UnstructuredContent()

		// create cannot set status
		delete(grafanaResource, "status")
	}

	accessor, _ := meta.Accessor(obj)
	accessor.SetGeneration(1)
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (a grafanaResourceStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	newGrafanaResourceObject := obj.(*unstructured.Unstructured)
	oldGrafanaResourceObject := old.(*unstructured.Unstructured)

	newGrafanaResource := newGrafanaResourceObject.UnstructuredContent()
	oldGrafanaResource := oldGrafanaResourceObject.UnstructuredContent()

	// If the /status subresource endpoint is installed, update is not allowed to set status.
	if a.status {
		_, ok1 := newGrafanaResource["status"]
		_, ok2 := oldGrafanaResource["status"]
		switch {
		case ok2:
			newGrafanaResource["status"] = oldGrafanaResource["status"]
		case ok1:
			delete(newGrafanaResource, "status")
		}
	}

	// except for the changes to `metadata`, any other changes
	// cause the generation to increment.
	newCopyContent := copyNonMetadata(newGrafanaResource)
	oldCopyContent := copyNonMetadata(oldGrafanaResource)
	if !apiequality.Semantic.DeepEqual(newCopyContent, oldCopyContent) {
		oldAccessor, _ := meta.Accessor(oldGrafanaResourceObject)
		newAccessor, _ := meta.Accessor(newGrafanaResourceObject)
		newAccessor.SetGeneration(oldAccessor.GetGeneration() + 1)
	}
}

func copyNonMetadata(original map[string]interface{}) map[string]interface{} {
	ret := make(map[string]interface{})
	for key, val := range original {
		if key == "metadata" {
			continue
		}
		ret[key] = val
	}
	return ret
}

// Validate validates a new GrafanaResource.
func (a grafanaResourceStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return make(field.ErrorList, 0)
}

// WarningsOnCreate returns warnings for the creation of the given object.
func (grafanaResourceStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

// Canonicalize normalizes the object after validation.
func (grafanaResourceStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for GrafanaResources; this means a POST is
// needed to create one.
func (grafanaResourceStrategy) AllowCreateOnUpdate() bool {
	return false
}

// AllowUnconditionalUpdate is the default update policy for GrafanaResource objects.
func (grafanaResourceStrategy) AllowUnconditionalUpdate() bool {
	return false
}

// ValidateUpdate is the default update validation for an end user updating status.
func (a grafanaResourceStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return make(field.ErrorList, 0)
}

// WarningsOnUpdate returns warnings for the given update.
func (grafanaResourceStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

// GetAttrs returns labels and fields of a given object for filtering purposes.
func (a grafanaResourceStrategy) GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return nil, nil, err
	}
	return labels.Set(accessor.GetLabels()), objectMetaFieldsSet(accessor, a.namespaceScoped), nil
}

// objectMetaFieldsSet returns a fields that represent the ObjectMeta.
func objectMetaFieldsSet(objectMeta metav1.Object, namespaceScoped bool) fields.Set {
	if namespaceScoped {
		return fields.Set{
			"metadata.name":      objectMeta.GetName(),
			"metadata.namespace": objectMeta.GetNamespace(),
		}
	}
	return fields.Set{
		"metadata.name": objectMeta.GetName(),
	}
}

// MatchGrafanaResourceDefinitionStorage is the filter used by the generic storage backend to route
// watch events from etcd to clients of the apiserver only interested in specific
// labels/fields.
func (a grafanaResourceStrategy) MatchGrafanaResourceDefinitionStorage(label labels.Selector, field fields.Selector) apiserverstorage.SelectionPredicate {
	return apiserverstorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: a.GetAttrs,
	}
}
