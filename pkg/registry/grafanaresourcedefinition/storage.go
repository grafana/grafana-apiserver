// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/registry/customresourcedefinition/storage.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package grafanaresourcedefinition

import (
	"context"
	"fmt"

	"github.com/grafana/grafana-apiserver/pkg/apis/kinds"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	storageerr "k8s.io/apiserver/pkg/storage/errors"
	"k8s.io/apiserver/pkg/util/dryrun"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"
)

// rest implements a RESTStorage for API services against etcd
type REST struct {
	*genericregistry.Store
}

// NewREST returns a RESTStorage object that will work against API services.
func NewREST(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (*REST, error) {
	strategy := NewStrategy(scheme)

	store := &genericregistry.Store{
		NewFunc:     func() runtime.Object { return &kinds.GrafanaResourceDefinition{} },
		NewListFunc: func() runtime.Object { return &kinds.GrafanaResourceDefinitionList{} },

		PredicateFunc:             MatchGrafanaResourceDefinition,
		DefaultQualifiedResource:  kinds.Resource("grafanaresourcedefinitions"),
		SingularQualifiedResource: kinds.Resource("grafanaresourcedefinition"),

		CreateStrategy:      strategy,
		UpdateStrategy:      strategy,
		DeleteStrategy:      strategy,
		ResetFieldsStrategy: strategy,

		// TODO: define table converter that exposes more than name/creation timestamp
		TableConvertor: rest.NewDefaultTableConvertor(kinds.Resource("grafanaresourcedefinitions")),
	}
	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}
	return &REST{store}, nil
}

// Implement ShortNamesProvider
var _ rest.ShortNamesProvider = &REST{}

// ShortNames implements the ShortNamesProvider interface. Returns a list of short names for a resource.
func (r *REST) ShortNames() []string {
	return []string{"grd", "grds"}
}

// Implement CategoriesProvider
var _ rest.CategoriesProvider = &REST{}

// Categories implements the CategoriesProvider interface. Returns a list of categories a resource is part of.
func (r *REST) Categories() []string {
	return []string{"api-extensions"}
}

// Delete adds the GRD finalizer to the list
func (r *REST) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	obj, err := r.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		return nil, false, err
	}

	grd := obj.(*kinds.GrafanaResourceDefinition)

	// Ensure we have a UID precondition
	if options == nil {
		options = metav1.NewDeleteOptions(0)
	}
	if options.Preconditions == nil {
		options.Preconditions = &metav1.Preconditions{}
	}
	if options.Preconditions.UID == nil {
		options.Preconditions.UID = &grd.UID
	} else if *options.Preconditions.UID != grd.UID {
		err = apierrors.NewConflict(
			kinds.Resource("grafanaresourcedefinitions"),
			name,
			fmt.Errorf("precondition failed: UID in precondition: %v, UID in object meta: %v", *options.Preconditions.UID, grd.UID),
		)
		return nil, false, err
	}
	if options.Preconditions.ResourceVersion != nil && *options.Preconditions.ResourceVersion != grd.ResourceVersion {
		err = apierrors.NewConflict(
			kinds.Resource("grafanaresourcedefinitions"),
			name,
			fmt.Errorf("precondition failed: ResourceVersion in precondition: %v, ResourceVersion in object meta: %v", *options.Preconditions.ResourceVersion, grd.ResourceVersion),
		)
		return nil, false, err
	}

	// upon first request to delete, add our finalizer and then delegate
	if grd.DeletionTimestamp.IsZero() {
		key, err := r.Store.KeyFunc(ctx, name)
		if err != nil {
			return nil, false, err
		}

		preconditions := storage.Preconditions{UID: options.Preconditions.UID, ResourceVersion: options.Preconditions.ResourceVersion}

		out := r.Store.NewFunc()
		err = r.Store.Storage.GuaranteedUpdate(
			ctx, key, out, false, &preconditions,
			storage.SimpleUpdate(func(existing runtime.Object) (runtime.Object, error) {
				existingGRD, ok := existing.(*kinds.GrafanaResourceDefinition)
				if !ok {
					// wrong type
					return nil, fmt.Errorf("expected *kinds.GrafanaResourceDefinition, got %v", existing)
				}
				if err := deleteValidation(ctx, existingGRD); err != nil {
					return nil, err
				}

				// Set the deletion timestamp if needed
				if existingGRD.DeletionTimestamp.IsZero() {
					now := metav1.Now()
					existingGRD.DeletionTimestamp = &now
				}

				if !kinds.GRDHasFinalizer(existingGRD, kinds.GrafanaResourceCleanupFinalizer) {
					existingGRD.Finalizers = append(existingGRD.Finalizers, kinds.GrafanaResourceCleanupFinalizer)
				}
				// update the status condition too
				kinds.SetGRDCondition(existingGRD, kinds.GrafanaResourceDefinitionCondition{
					Type:    kinds.Terminating,
					Status:  kinds.ConditionTrue,
					Reason:  "InstanceDeletionPending",
					Message: "GrafanaResourceDefinition marked for deletion; GrafanaResource deletion will begin soon",
				})
				return existingGRD, nil
			}),
			dryrun.IsDryRun(options.DryRun),
			nil,
		)

		if err != nil {
			err = storageerr.InterpretGetError(err, kinds.Resource("grafanaresourcedefinitions"), name)
			err = storageerr.InterpretUpdateError(err, kinds.Resource("grafanaresourcedefinitions"), name)
			if _, ok := err.(*apierrors.StatusError); !ok {
				err = apierrors.NewInternalError(err)
			}
			return nil, false, err
		}

		return out, false, nil
	}

	return r.Store.Delete(ctx, name, deleteValidation, options)
}

// NewStatusREST makes a RESTStorage for status that has more limited options.
// It is based on the original REST so that we can share the same underlying store
func NewStatusREST(scheme *runtime.Scheme, rest *REST) *StatusREST {
	statusStore := *rest.Store
	statusStore.CreateStrategy = nil
	statusStore.DeleteStrategy = nil
	statusStrategy := NewStatusStrategy(scheme)
	statusStore.UpdateStrategy = statusStrategy
	statusStore.ResetFieldsStrategy = statusStrategy
	return &StatusREST{store: &statusStore}
}

type StatusREST struct {
	store *genericregistry.Store
}

var _ = rest.Patcher(&StatusREST{})

func (r *StatusREST) New() runtime.Object {
	return &kinds.GrafanaResourceDefinition{}
}

// Destroy cleans up resources on shutdown.
func (r *StatusREST) Destroy() {
	// Given that underlying store is shared with REST,
	// we don't destroy it here explicitly.
}

// Get retrieves the object from the storage. It is required to support Patch.
func (r *StatusREST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return r.store.Get(ctx, name, options)
}

// Update alters the status subset of an object.
func (r *StatusREST) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	// We are explicitly setting forceAllowCreate to false in the call to the underlying storage because
	// subresources should never allow create on update.
	return r.store.Update(ctx, name, objInfo, createValidation, updateValidation, false, options)
}

// GetResetFields implements rest.ResetFieldsStrategy
func (r *StatusREST) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
	return r.store.GetResetFields()
}
