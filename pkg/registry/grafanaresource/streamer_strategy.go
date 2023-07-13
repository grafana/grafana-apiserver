// SPDX-License-Identifier: AGPL-3.0-only

package grafanaresource

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage/names"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

type streamerStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

var _ rest.RESTCreateStrategy = (*streamerStrategy)(nil)
var _ rest.RESTUpdateStrategy = (*streamerStrategy)(nil)
var _ rest.RESTDeleteStrategy = (*streamerStrategy)(nil)
var _ rest.ResetFieldsStrategy = (*streamerStrategy)(nil)

func NewStreamerStrategy(typer runtime.ObjectTyper) streamerStrategy {
	return streamerStrategy{
		ObjectTyper:   typer,
		NameGenerator: names.SimpleNameGenerator,
	}
}

func (streamerStrategy) NamespaceScoped() bool {
	return true
}

func (streamerStrategy) Canonicalize(obj runtime.Object) {}

func (streamerStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {}

func (streamerStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (streamerStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (streamerStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {}

func (streamerStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}

func (streamerStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (streamerStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (streamerStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (streamerStrategy) PrepareForDelete(ctx context.Context, obj runtime.Object) {}

func (streamerStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return map[fieldpath.APIVersion]*fieldpath.Set{}
}
