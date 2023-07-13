// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/apiserver/customresource_handler.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package apiserver

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/managedfields"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	utilwaitgroup "k8s.io/apimachinery/pkg/util/waitgroup"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/endpoints/handlers"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	"k8s.io/apiserver/pkg/endpoints/metrics"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	genericfilters "k8s.io/apiserver/pkg/server/filters"
	"k8s.io/apiserver/pkg/warning"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"k8s.io/kube-openapi/pkg/validation/spec"
	"sigs.k8s.io/structured-merge-diff/v4/fieldpath"

	kindshelpers "github.com/grafana/grafana-apiserver/pkg/apihelpers"
	kindsv1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	informers "github.com/grafana/grafana-apiserver/pkg/client/informers/externalversions/kinds/v1"
	listers "github.com/grafana/grafana-apiserver/pkg/client/listers/kinds/v1"
	"github.com/grafana/grafana-apiserver/pkg/controller/establish"
	"github.com/grafana/grafana-apiserver/pkg/controller/finalizer"
	"github.com/grafana/grafana-apiserver/pkg/endpoints/request"
	"github.com/grafana/grafana-apiserver/pkg/registry/grafanaresource"
	grafanaruntime "github.com/grafana/grafana-apiserver/pkg/runtime"
)

// grdHandler serves the `/apis` endpoint.
// This is registered as a filter so that it never collides with any explicitly registered endpoints
type grdHandler struct {
	versionDiscoveryHandler *versionDiscoveryHandler
	groupDiscoveryHandler   *groupDiscoveryHandler

	customStorageLock sync.Mutex
	// customStorage contains a grdStorageMap
	// atomic.Value has a very good read performance compared to sync.RWMutex
	// see https://gist.github.com/dim/152e6bf80e1384ea72e17ac717a5000a
	// which is suited for most read and rarely write cases
	customStorage atomic.Value

	grdLister listers.GrafanaResourceDefinitionLister

	delegate          http.Handler
	restOptionsGetter generic.RESTOptionsGetter
	admission         admission.Interface

	establishingController *establish.EstablishingController

	// MasterCount is used to implement sleep to improve
	// GRD establishing process for HA clusters.
	masterCount int

	// so that we can do create on update.
	authorizer authorizer.Authorizer

	// request timeout we should delay storage teardown for
	requestTimeout time.Duration

	// minRequestTimeout applies to CR's list/watch calls
	minRequestTimeout time.Duration

	// staticOpenAPISpec is used as a base for the schema of CR's for the
	// purpose of managing fields, it is how CR handlers get the structure
	// of TypeMeta and ObjectMeta
	staticOpenAPISpec map[string]*spec.Schema

	// The limit on the request size that would be accepted and decoded in a write request
	// 0 means no limit.
	maxRequestBodyBytes int64
}

// grdInfo stores enough information to serve the storage for the custom resource
type grdInfo struct {
	// spec and acceptedNames are used to compare against if a change is made on a GRD. We only update
	// the storage if one of these changes.
	spec          *kindsv1.GrafanaResourceDefinitionSpec
	acceptedNames *kindsv1.GrafanaResourceDefinitionNames

	// Deprecated per version
	deprecated map[string]bool

	// Warnings per version
	warnings map[string][]string

	// Storage per version
	storages map[string]grafanaresource.GrafanaResourceStorage

	// Request scope per version
	requestScopes map[string]*handlers.RequestScope

	// Scale scope per version
	scaleRequestScopes map[string]*handlers.RequestScope

	// Status scope per version
	statusRequestScopes map[string]*handlers.RequestScope

	// storageVersion is the GRD version used when storing the object in etcd.
	storageVersion string

	waitGroup *utilwaitgroup.SafeWaitGroup
}

// grdStorageMap goes from grafanaresourcedefinition to its storage
type grdStorageMap map[types.UID]*grdInfo

func NewGrafanaResourceDefinitionHandler(
	versionDiscoveryHandler *versionDiscoveryHandler,
	groupDiscoveryHandler *groupDiscoveryHandler,
	grdInformer informers.GrafanaResourceDefinitionInformer,
	delegate http.Handler,
	restOptionsGetter generic.RESTOptionsGetter,
	admission admission.Interface,
	establishingController *establish.EstablishingController,
	masterCount int,
	authorizer authorizer.Authorizer,
	requestTimeout time.Duration,
	minRequestTimeout time.Duration,
	staticOpenAPISpec map[string]*spec.Schema,
	maxRequestBodyBytes int64) (*grdHandler, error) {
	ret := &grdHandler{
		versionDiscoveryHandler: versionDiscoveryHandler,
		groupDiscoveryHandler:   groupDiscoveryHandler,
		customStorage:           atomic.Value{},
		grdLister:               grdInformer.Lister(),
		delegate:                delegate,
		restOptionsGetter:       restOptionsGetter,
		admission:               admission,
		establishingController:  establishingController,
		masterCount:             masterCount,
		authorizer:              authorizer,
		requestTimeout:          requestTimeout,
		minRequestTimeout:       minRequestTimeout,
		staticOpenAPISpec:       staticOpenAPISpec,
		maxRequestBodyBytes:     maxRequestBodyBytes,
	}
	grdInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    ret.createGrafanaResourceDefinition,
		UpdateFunc: ret.updateGrafanaResourceDefinition,
		DeleteFunc: func(obj interface{}) {
			ret.removeDeadStorage()
		},
	})

	ret.customStorage.Store(grdStorageMap{})

	return ret, nil
}

// watches are expected to handle storage disruption gracefully,
// both on the server-side (by terminating the watch connection)
// and on the client side (by restarting the watch)
var longRunningFilter = genericfilters.BasicLongRunningRequestCheck(sets.NewString("watch"), sets.NewString())

// possiblyAcrossAllNamespacesVerbs contains those verbs which can be per-namespace and across all
// namespaces for namespaces resources. I.e. for these an empty namespace in the requestInfo is fine.
var possiblyAcrossAllNamespacesVerbs = sets.NewString("list", "watch")

func (r *grdHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	requestInfo, ok := apirequest.RequestInfoFrom(ctx)
	if !ok {
		responsewriters.ErrorNegotiated(
			apierrors.NewInternalError(fmt.Errorf("no RequestInfo found in the context")),
			Codecs, schema.GroupVersion{}, w, req,
		)
		return
	}
	if !requestInfo.IsResourceRequest {
		pathParts := splitPath(requestInfo.Path)
		// only match /apis/<group>/<version>
		// only registered under /apis
		if len(pathParts) == 3 {
			r.versionDiscoveryHandler.ServeHTTP(w, req)
			return
		}
		// only match /apis/<group>
		if len(pathParts) == 2 {
			r.groupDiscoveryHandler.ServeHTTP(w, req)
			return
		}

		r.delegate.ServeHTTP(w, req)
		return
	}

	grdName := requestInfo.Resource + "." + requestInfo.APIGroup
	grd, err := r.grdLister.Get(grdName)
	if apierrors.IsNotFound(err) {
		r.delegate.ServeHTTP(w, req)
		return
	}
	if err != nil {
		utilruntime.HandleError(err)
		responsewriters.ErrorNegotiated(
			apierrors.NewInternalError(fmt.Errorf("error resolving resource")),
			Codecs, schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}, w, req,
		)
		return
	}

	// if the scope in the GRD and the scope in request differ (with exception of the verbs in possiblyAcrossAllNamespacesVerbs
	// for namespaced resources), pass request to the delegate, which is supposed to lead to a 404.
	namespacedGRD, namespacedReq := grd.Spec.Scope == kindsv1.NamespaceScoped, len(requestInfo.Namespace) > 0
	if !namespacedGRD && namespacedReq {
		r.delegate.ServeHTTP(w, req)
		return
	}
	if namespacedGRD && !namespacedReq && !possiblyAcrossAllNamespacesVerbs.Has(requestInfo.Verb) {
		r.delegate.ServeHTTP(w, req)
		return
	}

	if !kindshelpers.HasServedGRDVersion(grd, requestInfo.APIVersion) {
		r.delegate.ServeHTTP(w, req)
		return
	}

	// There is a small chance that a GRD is being served because NamesAccepted condition is true,
	// but it becomes "unserved" because another names update leads to a conflict
	// and EstablishingController wasn't fast enough to put the GRD into the Established condition.
	// We accept this as the problem is small and self-healing.
	if !kindshelpers.IsGRDConditionTrue(grd, kindsv1.NamesAccepted) &&
		!kindshelpers.IsGRDConditionTrue(grd, kindsv1.Established) {
		r.delegate.ServeHTTP(w, req)
		return
	}

	terminating := kindshelpers.IsGRDConditionTrue(grd, kindsv1.Terminating)

	grdInfo, err := r.getOrCreateServingInfoFor(grd.UID, grd.Name)
	if apierrors.IsNotFound(err) {
		r.delegate.ServeHTTP(w, req)
		return
	}
	if err != nil {
		utilruntime.HandleError(err)
		responsewriters.ErrorNegotiated(
			apierrors.NewInternalError(fmt.Errorf("error resolving resource")),
			Codecs, schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}, w, req,
		)
		return
	}
	if !hasServedGRDVersion(grdInfo.spec, requestInfo.APIVersion) {
		r.delegate.ServeHTTP(w, req)
		return
	}

	deprecated := grdInfo.deprecated[requestInfo.APIVersion]
	for _, w := range grdInfo.warnings[requestInfo.APIVersion] {
		warning.AddWarning(req.Context(), "", w)
	}

	verb := strings.ToUpper(requestInfo.Verb)
	resource := requestInfo.Resource
	subresource := requestInfo.Subresource
	scope := metrics.CleanScope(requestInfo)
	supportedTypes := []string{
		string(types.JSONPatchType),
		string(types.MergePatchType),
		string(types.ApplyPatchType),
	}

	var handlerFunc http.HandlerFunc
	subresources, err := kindshelpers.GetSubresourcesForVersion(grd, requestInfo.APIVersion)
	if err != nil {
		utilruntime.HandleError(err)
		responsewriters.ErrorNegotiated(
			apierrors.NewInternalError(fmt.Errorf("could not properly serve the subresource")),
			Codecs, schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}, w, req,
		)
		return
	}
	switch {
	case subresource == "status" && subresources != nil && subresources.Status != nil:
		handlerFunc = r.serveStatus(w, req, requestInfo, grdInfo, terminating, supportedTypes)
	case subresource == "ref" && subresources != nil && subresources.Ref != nil:
		handlerFunc = r.serveRef(w, req, requestInfo, grdInfo)
	case subresource == "history" && subresources != nil && subresources.History != nil:
		handlerFunc = r.serveHistory(w, req, requestInfo, grdInfo)
	case len(subresource) == 0:
		handlerFunc = r.serveResource(w, req, requestInfo, grdInfo, grd, terminating, supportedTypes)
	default:
		responsewriters.ErrorNegotiated(
			apierrors.NewNotFound(schema.GroupResource{Group: requestInfo.APIGroup, Resource: requestInfo.Resource}, requestInfo.Name),
			Codecs, schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}, w, req,
		)
	}

	if handlerFunc != nil {
		requestScope := grdInfo.requestScopes[requestInfo.APIVersion]
		ctx = request.WithOutputMediaType(req.Context(), req, requestScope)
		req = req.WithContext(ctx)

		handlerFunc = metrics.InstrumentHandlerFunc(verb, requestInfo.APIGroup, requestInfo.APIVersion, resource, subresource, scope, metrics.APIServerComponent, deprecated, "", handlerFunc)
		handler := genericfilters.WithWaitGroup(handlerFunc, longRunningFilter, grdInfo.waitGroup)
		handler.ServeHTTP(w, req)
		return
	}
}

func (r *grdHandler) serveResource(w http.ResponseWriter, req *http.Request, requestInfo *apirequest.RequestInfo, grdInfo *grdInfo, grd *kindsv1.GrafanaResourceDefinition, terminating bool, supportedTypes []string) http.HandlerFunc {
	requestScope := grdInfo.requestScopes[requestInfo.APIVersion]
	storage := grdInfo.storages[requestInfo.APIVersion].GrafanaResource

	switch requestInfo.Verb {
	case "get":
		return handlers.GetResource(storage, requestScope)
	case "list":
		forceWatch := false
		return handlers.ListResource(storage, storage, requestScope, forceWatch, r.minRequestTimeout)
	case "watch":
		forceWatch := true
		return handlers.ListResource(storage, storage, requestScope, forceWatch, r.minRequestTimeout)
	case "create":
		// we want to track recently created GRDs so that in HA environments we don't have server A allow a create and server B
		// not have observed the established, so a followup get,update,delete results in a 404. We've observed about 800ms
		// delay in some CI environments.  Two seconds looks long enough and reasonably short for hot retriers.
		justCreated := time.Since(kindshelpers.FindGRDCondition(grd, kindsv1.Established).LastTransitionTime.Time) < 2*time.Second
		if justCreated {
			time.Sleep(2 * time.Second)
		}
		if terminating {
			err := apierrors.NewMethodNotSupported(schema.GroupResource{Group: requestInfo.APIGroup, Resource: requestInfo.Resource}, requestInfo.Verb)
			err.ErrStatus.Message = fmt.Sprintf("%v not allowed while custom resource definition is terminating", requestInfo.Verb)
			responsewriters.ErrorNegotiated(err, Codecs, schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}, w, req)
			return nil
		}
		return handlers.CreateResource(storage, requestScope, r.admission)
	case "update":
		return handlers.UpdateResource(storage, requestScope, r.admission)
	case "patch":
		return handlers.PatchResource(storage, requestScope, r.admission, supportedTypes)
	case "delete":
		allowsOptions := true
		return handlers.DeleteResource(storage, allowsOptions, requestScope, r.admission)
	case "deletecollection":
		checkBody := true
		return handlers.DeleteCollection(storage, checkBody, requestScope, r.admission)
	default:
		responsewriters.ErrorNegotiated(
			apierrors.NewMethodNotSupported(schema.GroupResource{Group: requestInfo.APIGroup, Resource: requestInfo.Resource}, requestInfo.Verb),
			Codecs, schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}, w, req,
		)
		return nil
	}
}

func (r *grdHandler) serveStatus(w http.ResponseWriter, req *http.Request, requestInfo *apirequest.RequestInfo, grdInfo *grdInfo, terminating bool, supportedTypes []string) http.HandlerFunc {
	requestScope := grdInfo.statusRequestScopes[requestInfo.APIVersion]
	storage := grdInfo.storages[requestInfo.APIVersion].Status

	switch requestInfo.Verb {
	case "get":
		return handlers.GetResource(storage, requestScope)
	case "update":
		return handlers.UpdateResource(storage, requestScope, r.admission)
	case "patch":
		return handlers.PatchResource(storage, requestScope, r.admission, supportedTypes)
	default:
		responsewriters.ErrorNegotiated(
			apierrors.NewMethodNotSupported(schema.GroupResource{Group: requestInfo.APIGroup, Resource: requestInfo.Resource}, requestInfo.Verb),
			Codecs, schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}, w, req,
		)
		return nil
	}
}

func (r *grdHandler) serveRef(w http.ResponseWriter, req *http.Request, requestInfo *apirequest.RequestInfo, grdInfo *grdInfo) http.HandlerFunc {
	requestScope := grdInfo.statusRequestScopes[requestInfo.APIVersion]
	storage := grdInfo.storages[requestInfo.APIVersion].Ref

	switch requestInfo.Verb {
	case "get":
		return handlers.GetResource(storage, requestScope)
	default:
		responsewriters.ErrorNegotiated(
			apierrors.NewMethodNotSupported(schema.GroupResource{Group: requestInfo.APIGroup, Resource: requestInfo.Resource}, requestInfo.Verb),
			Codecs, schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}, w, req,
		)
		return nil
	}
}

func (r *grdHandler) serveHistory(w http.ResponseWriter, req *http.Request, requestInfo *apirequest.RequestInfo, grdInfo *grdInfo) http.HandlerFunc {
	requestScope := grdInfo.statusRequestScopes[requestInfo.APIVersion]
	storage := grdInfo.storages[requestInfo.APIVersion].History

	switch requestInfo.Verb {
	case "get":
		return handlers.GetResource(storage, requestScope)
	default:
		responsewriters.ErrorNegotiated(
			apierrors.NewMethodNotSupported(schema.GroupResource{Group: requestInfo.APIGroup, Resource: requestInfo.Resource}, requestInfo.Verb),
			Codecs, schema.GroupVersion{Group: requestInfo.APIGroup, Version: requestInfo.APIVersion}, w, req,
		)
		return nil
	}
}

// createGrafanaResourceDefinition removes potentially stale storage so it gets re-created
func (r *grdHandler) createGrafanaResourceDefinition(obj interface{}) {
	grd := obj.(*kindsv1.GrafanaResourceDefinition)
	r.customStorageLock.Lock()
	defer r.customStorageLock.Unlock()
	// this could happen if the create event is merged from create-update events
	storageMap := r.customStorage.Load().(grdStorageMap)
	oldInfo, found := storageMap[grd.UID]
	if !found {
		return
	}
	if apiequality.Semantic.DeepEqual(&grd.Spec, oldInfo.spec) && apiequality.Semantic.DeepEqual(&grd.Status.AcceptedNames, oldInfo.acceptedNames) {
		klog.V(6).Infof("Ignoring grafanaresourcedefinition %s create event because a storage with the same spec and accepted names exists",
			grd.Name)
		return
	}
	r.removeStorage_locked(grd.UID)
}

// updateGrafanaResourceDefinition removes potentially stale storage so it gets re-created
func (r *grdHandler) updateGrafanaResourceDefinition(oldObj, newObj interface{}) {
	oldGRD := oldObj.(*kindsv1.GrafanaResourceDefinition)
	newGRD := newObj.(*kindsv1.GrafanaResourceDefinition)

	r.customStorageLock.Lock()
	defer r.customStorageLock.Unlock()

	// Add GRD to the establishing controller queue.
	// For HA clusters, we want to prevent race conditions when changing status to Established,
	// so we want to be sure that GRD is Installing at least for 5 seconds before Establishing it.
	// TODO: find a real HA safe checkpointing mechanism instead of an arbitrary wait.
	if !kindshelpers.IsGRDConditionTrue(newGRD, kindsv1.Established) &&
		kindshelpers.IsGRDConditionTrue(newGRD, kindsv1.NamesAccepted) {
		if r.masterCount > 1 {
			r.establishingController.QueueGRD(newGRD.Name, 5*time.Second)
		} else {
			r.establishingController.QueueGRD(newGRD.Name, 0)
		}
	}

	if oldGRD.UID != newGRD.UID {
		r.removeStorage_locked(oldGRD.UID)
	}

	storageMap := r.customStorage.Load().(grdStorageMap)
	oldInfo, found := storageMap[newGRD.UID]
	if !found {
		return
	}
	if apiequality.Semantic.DeepEqual(&newGRD.Spec, oldInfo.spec) && apiequality.Semantic.DeepEqual(&newGRD.Status.AcceptedNames, oldInfo.acceptedNames) {
		klog.V(6).Infof("Ignoring grafanaresourcedefinition %s update because neither spec, nor accepted names changed", oldGRD.Name)
		return
	}

	klog.V(4).Infof("Updating grafanaresourcedefinition %s", newGRD.Name)
	r.removeStorage_locked(newGRD.UID)
}

// removeStorage_locked removes the cached storage with the given uid as key from the storage map. This function
// updates r.customStorage with the cleaned-up storageMap and tears down the old storage.
// NOTE: Caller MUST hold r.customStorageLock to write r.customStorage thread-safely.
func (r *grdHandler) removeStorage_locked(uid types.UID) {
	storageMap := r.customStorage.Load().(grdStorageMap)
	if oldInfo, ok := storageMap[uid]; ok {
		// Copy because we cannot write to storageMap without a race
		// as it is used without locking elsewhere.
		storageMap2 := storageMap.clone()

		// Remove from the GRD info map and store the map
		delete(storageMap2, uid)
		r.customStorage.Store(storageMap2)

		// Tear down the old storage
		go r.tearDown(oldInfo)
	}
}

// removeDeadStorage removes REST storage that isn't being used
func (r *grdHandler) removeDeadStorage() {
	allGrafanaResourceDefinitions, err := r.grdLister.List(labels.Everything())
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	r.customStorageLock.Lock()
	defer r.customStorageLock.Unlock()

	storageMap := r.customStorage.Load().(grdStorageMap)
	// Copy because we cannot write to storageMap without a race
	storageMap2 := make(grdStorageMap)
	for _, grd := range allGrafanaResourceDefinitions {
		if _, ok := storageMap[grd.UID]; ok {
			storageMap2[grd.UID] = storageMap[grd.UID]
		}
	}
	r.customStorage.Store(storageMap2)

	for uid, grdInfo := range storageMap {
		if _, ok := storageMap2[uid]; !ok {
			klog.V(4).Infof("Removing dead GRD storage for %s/%s", grdInfo.spec.Group, grdInfo.spec.Names.Kind)
			go r.tearDown(grdInfo)
		}
	}
}

// Wait up to a minute for requests to drain, then tear down storage
func (r *grdHandler) tearDown(oldInfo *grdInfo) {
	requestsDrained := make(chan struct{})
	go func() {
		defer close(requestsDrained)
		// Allow time for in-flight requests with a handle to the old info to register themselves
		time.Sleep(time.Second)
		// Wait for in-flight requests to drain
		oldInfo.waitGroup.Wait()
	}()

	select {
	case <-time.After(r.requestTimeout * 2):
		klog.Warningf("timeout waiting for requests to drain for %s/%s, tearing down storage", oldInfo.spec.Group, oldInfo.spec.Names.Kind)
	case <-requestsDrained:
	}

	for _, storage := range oldInfo.storages {
		// destroy only the main storage. Those for the subresources share cacher and etcd clients.
		storage.GrafanaResource.DestroyFunc()
	}
}

// Destroy shuts down storage layer for all registered GRDs.
// It should be called as a last step of the shutdown sequence.
func (r *grdHandler) destroy() {
	r.customStorageLock.Lock()
	defer r.customStorageLock.Unlock()

	storageMap := r.customStorage.Load().(grdStorageMap)
	for _, grdInfo := range storageMap {
		for _, storage := range grdInfo.storages {
			// DestroyFunc have to be implemented in idempotent way,
			// so the potential race with r.tearDown() (being called
			// from a goroutine) is safe.
			storage.GrafanaResource.DestroyFunc()
		}
	}
}

// GetGrafanaResourceListerCollectionDeleter returns the ListerCollectionDeleter of
// the given grd.
func (r *grdHandler) GetGrafanaResourceListerCollectionDeleter(grd *kindsv1.GrafanaResourceDefinition) (finalizer.ListerCollectionDeleter, error) {
	info, err := r.getOrCreateServingInfoFor(grd.UID, grd.Name)
	if err != nil {
		return nil, err
	}
	return info.storages[info.storageVersion].GrafanaResource, nil
}

// getOrCreateServingInfoFor gets the GRD serving info for the given GRD UID if the key exists in the storage map.
// Otherwise the function fetches the up-to-date GRD using the given GRD name and creates GRD serving info.
func (r *grdHandler) getOrCreateServingInfoFor(uid types.UID, name string) (*grdInfo, error) {
	storageMap := r.customStorage.Load().(grdStorageMap)
	if ret, ok := storageMap[uid]; ok {
		return ret, nil
	}

	r.customStorageLock.Lock()
	defer r.customStorageLock.Unlock()

	// Get the up-to-date GRD when we have the lock, to avoid racing with updateGrafanaResourceDefinition.
	// If updateGrafanaResourceDefinition sees an update and happens later, the storage will be deleted and
	// we will re-create the updated storage on demand. If updateGrafanaResourceDefinition happens before,
	// we make sure that we observe the same up-to-date GRD.
	grd, err := r.grdLister.Get(name)
	if err != nil {
		return nil, err
	}
	storageMap = r.customStorage.Load().(grdStorageMap)
	if ret, ok := storageMap[grd.UID]; ok {
		return ret, nil
	}

	storageVersion, err := kindshelpers.GetGRDStorageVersion(grd)
	if err != nil {
		return nil, err
	}

	// Scope/Storages per version.
	requestScopes := map[string]*handlers.RequestScope{}
	storages := map[string]grafanaresource.GrafanaResourceStorage{}
	statusScopes := map[string]*handlers.RequestScope{}
	scaleScopes := map[string]*handlers.RequestScope{}
	deprecated := map[string]bool{}
	warnings := map[string][]string{}

	equivalentResourceRegistry := runtime.NewEquivalentResourceRegistry()

	// Create replicasPathInGrafanaResource
	replicasPathInGrafanaResource := managedfields.ResourcePathMappings{}
	for _, v := range grd.Spec.Versions {
		subresources, err := kindshelpers.GetSubresourcesForVersion(grd, v.Name)
		if err != nil {
			utilruntime.HandleError(err)
			return nil, fmt.Errorf("the server could not properly serve the CR subresources")
		}
		if subresources == nil || subresources.Scale == nil {
			replicasPathInGrafanaResource[schema.GroupVersion{Group: grd.Spec.Group, Version: v.Name}.String()] = nil
			continue
		}
		path := fieldpath.Path{}
		splitReplicasPath := strings.Split(strings.TrimPrefix(subresources.Scale.SpecReplicasPath, "."), ".")
		for _, element := range splitReplicasPath {
			s := element
			path = append(path, fieldpath.PathElement{FieldName: &s})
		}
		replicasPathInGrafanaResource[schema.GroupVersion{Group: grd.Spec.Group, Version: v.Name}.String()] = path
	}

	for _, v := range grd.Spec.Versions {
		// In addition to Unstructured objects (Custom Resources), we also may sometimes need to
		// decode unversioned Options objects, so we delegate to parameterScheme for such types.
		parameterScheme := runtime.NewScheme()
		parameterScheme.AddUnversionedTypes(schema.GroupVersion{Group: grd.Spec.Group, Version: v.Name},
			&metav1.ListOptions{},
			&metav1.GetOptions{},
			&metav1.DeleteOptions{},
		)
		parameterCodec := runtime.NewParameterCodec(parameterScheme)

		resource := schema.GroupVersionResource{Group: grd.Spec.Group, Version: v.Name, Resource: grd.Status.AcceptedNames.Plural}
		singularResource := schema.GroupVersionResource{Group: grd.Spec.Group, Version: v.Name, Resource: grd.Status.AcceptedNames.Singular}
		kind := schema.GroupVersionKind{Group: grd.Spec.Group, Version: v.Name, Kind: grd.Status.AcceptedNames.Kind}
		equivalentResourceRegistry.RegisterKindFor(resource, "", kind)

		subresources, err := kindshelpers.GetSubresourcesForVersion(grd, v.Name)
		if err != nil {
			utilruntime.HandleError(err)
			return nil, fmt.Errorf("the server could not properly serve the CR subresources")
		}

		table := rest.NewDefaultTableConvertor(grd.GroupVersionKind().GroupVersion().WithResource(grd.Status.AcceptedNames.Plural).GroupResource())

		creator := grafanaruntime.NewObjectCreator()
		typer := grafanaruntime.NewObjectTyper()
		convertor := grafanaruntime.NewObjectConvertor()

		storages[v.Name] = grafanaresource.NewStorage(
			resource.GroupResource(),
			singularResource.GroupResource(),
			kind,
			schema.GroupVersionKind{Group: grd.Spec.Group, Version: v.Name, Kind: grd.Status.AcceptedNames.ListKind},
			grafanaresource.NewStrategy(
				typer,
				grd.Spec.Scope == kindsv1.NamespaceScoped,
				kind,
				true,
				false,
				true,
				true,
			),
			r.restOptionsGetter,
			grd.Status.AcceptedNames.Categories,
			table,
		)

		clusterScoped := grd.Spec.Scope == kindsv1.ClusterScoped

		// GRDs explicitly do not support protobuf, but some objects returned by the API server do
		negotiatedSerializer := grafanaruntime.NewNegotiatedSerializer(Scheme)
		var standardSerializers []runtime.SerializerInfo
		for _, s := range negotiatedSerializer.SupportedMediaTypes() {
			if s.MediaType == runtime.ContentTypeProtobuf {
				continue
			}
			standardSerializers = append(standardSerializers, s)
		}

		reqScope := handlers.RequestScope{
			Namer: handlers.ContextBasedNaming{
				Namer:         meta.NewAccessor(),
				ClusterScoped: clusterScoped,
			},
			Serializer:          negotiatedSerializer,
			ParameterCodec:      parameterCodec,
			StandardSerializers: standardSerializers,

			Creater:         creator,
			Convertor:       convertor,
			Defaulter:       Scheme,
			Typer:           typer,
			UnsafeConvertor: convertor,

			EquivalentResourceMapper: equivalentResourceRegistry,

			Resource: schema.GroupVersionResource{Group: grd.Spec.Group, Version: v.Name, Resource: grd.Status.AcceptedNames.Plural},
			Kind:     kind,

			// a handler for a specific group-version of a custom resource uses that version as the in-memory representation
			HubGroupVersion: kind.GroupVersion(),

			MetaGroupVersion: metav1.SchemeGroupVersion,

			TableConvertor: storages[v.Name].GrafanaResource,

			Authorizer: r.authorizer,

			MaxRequestBodyBytes: r.maxRequestBodyBytes,
		}

		resetFields := storages[v.Name].GrafanaResource.GetResetFields()
		reqScope, err = scopeWithFieldManager(
			managedfields.NewDeducedTypeConverter(),
			reqScope,
			resetFields,
			"",
		)
		if err != nil {
			return nil, err
		}
		requestScopes[v.Name] = &reqScope

		// override status subresource values
		// shallow copy
		statusScope := *requestScopes[v.Name]
		statusScope.Subresource = "status"
		statusScope.Namer = handlers.ContextBasedNaming{
			Namer:         meta.NewAccessor(),
			ClusterScoped: clusterScoped,
		}

		if subresources != nil && subresources.Status != nil {
			resetFields := storages[v.Name].Status.GetResetFields()
			statusScope, err = scopeWithFieldManager(
				managedfields.NewDeducedTypeConverter(),
				statusScope,
				resetFields,
				"status",
			)
			if err != nil {
				return nil, err
			}
		}

		statusScopes[v.Name] = &statusScope

		if v.Deprecated {
			deprecated[v.Name] = true
			if v.DeprecationWarning != nil {
				warnings[v.Name] = append(warnings[v.Name], *v.DeprecationWarning)
			} else {
				warnings[v.Name] = append(warnings[v.Name], defaultDeprecationWarning(v.Name, grd.Spec))
			}
		}
	}

	ret := &grdInfo{
		spec:                &grd.Spec,
		acceptedNames:       &grd.Status.AcceptedNames,
		storages:            storages,
		requestScopes:       requestScopes,
		scaleRequestScopes:  scaleScopes,
		statusRequestScopes: statusScopes,
		deprecated:          deprecated,
		warnings:            warnings,
		storageVersion:      storageVersion,
		waitGroup:           &utilwaitgroup.SafeWaitGroup{},
	}

	// Copy because we cannot write to storageMap without a race
	// as it is used without locking elsewhere.
	storageMap2 := storageMap.clone()

	storageMap2[grd.UID] = ret
	r.customStorage.Store(storageMap2)

	return ret, nil
}

func scopeWithFieldManager(typeConverter managedfields.TypeConverter, reqScope handlers.RequestScope, resetFields map[fieldpath.APIVersion]*fieldpath.Set, subresource string) (handlers.RequestScope, error) {
	fieldManager, err := managedfields.NewDefaultCRDFieldManager(
		typeConverter,
		reqScope.Convertor,
		reqScope.Defaulter,
		reqScope.Creater,
		reqScope.Kind,
		reqScope.HubGroupVersion,
		subresource,
		resetFields,
	)
	if err != nil {
		return handlers.RequestScope{}, err
	}
	reqScope.FieldManager = fieldManager
	return reqScope, nil
}

func defaultDeprecationWarning(deprecatedVersion string, grd kindsv1.GrafanaResourceDefinitionSpec) string {
	msg := fmt.Sprintf("%s/%s %s is deprecated", grd.Group, deprecatedVersion, grd.Names.Kind)

	var servedNonDeprecatedVersions []string
	for _, v := range grd.Versions {
		if v.Served && !v.Deprecated && version.CompareKubeAwareVersionStrings(deprecatedVersion, v.Name) < 0 {
			servedNonDeprecatedVersions = append(servedNonDeprecatedVersions, v.Name)
		}
	}
	if len(servedNonDeprecatedVersions) == 0 {
		return msg
	}
	sort.Slice(servedNonDeprecatedVersions, func(i, j int) bool {
		return version.CompareKubeAwareVersionStrings(servedNonDeprecatedVersions[i], servedNonDeprecatedVersions[j]) > 0
	})
	msg += fmt.Sprintf("; use %s/%s %s", grd.Group, servedNonDeprecatedVersions[0], grd.Names.Kind)
	return msg
}

type UnstructuredObjectTyper struct {
	Delegate          runtime.ObjectTyper
	UnstructuredTyper runtime.ObjectTyper
}

// clone returns a clone of the provided grdStorageMap.
// The clone is a shallow copy of the map.
func (in grdStorageMap) clone() grdStorageMap {
	if in == nil {
		return nil
	}
	out := make(grdStorageMap, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

// hasServedGRDVersion returns true if the given version is in the list of GRD's versions and the Served flag is set.
func hasServedGRDVersion(spec *kindsv1.GrafanaResourceDefinitionSpec, version string) bool {
	for _, v := range spec.Versions {
		if v.Name == version {
			return v.Served
		}
	}
	return false
}
