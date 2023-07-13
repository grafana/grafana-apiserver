// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/controller/finalizer/crd_finalizer.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package finalizer

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"k8s.io/klog/v2"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	kindshelpers "github.com/grafana/grafana-apiserver/pkg/apihelpers"
	kindsv1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	client "github.com/grafana/grafana-apiserver/pkg/client/clientset/clientset/typed/kinds/v1"
	informers "github.com/grafana/grafana-apiserver/pkg/client/informers/externalversions/kinds/v1"
	listers "github.com/grafana/grafana-apiserver/pkg/client/listers/kinds/v1"
)

// OverlappingBuiltInResources returns the set of built-in group/resources that are persisted
// in storage paths that overlap with GRD storage paths, and should not be deleted
// by this controller if an associated GRD is deleted.
func OverlappingBuiltInResources() map[schema.GroupResource]bool {
	return map[schema.GroupResource]bool{
		{Group: "apiregistration.k8s.io", Resource: "apiservices"}:      true,
		{Group: "kinds.k8s.io", Resource: "grafanaresourcedefinitions"}: true,
	}
}

// GRDFinalizer is a controller that finalizes the GRD by deleting all the CRs associated with it.
type GRDFinalizer struct {
	grdClient      client.GrafanaResourceDefinitionsGetter
	crClientGetter GRClientGetter

	grdLister listers.GrafanaResourceDefinitionLister
	grdSynced cache.InformerSynced

	// To allow injection for testing.
	syncFn func(key string) error

	queue workqueue.RateLimitingInterface
}

// ListerCollectionDeleter combines rest.Lister and rest.CollectionDeleter.
type ListerCollectionDeleter interface {
	rest.Lister
	rest.CollectionDeleter
}

// GRClientGetter knows how to get a ListerCollectionDeleter for a given GRD UID.
type GRClientGetter interface {
	// GetGrafanaResourceListerCollectionDeleter gets the ListerCollectionDeleter for the given GRD
	// UID.
	GetGrafanaResourceListerCollectionDeleter(grd *kindsv1.GrafanaResourceDefinition) (ListerCollectionDeleter, error)
}

// NewGRDFinalizer creates a new GRDFinalizer.
func NewGRDFinalizer(
	grdInformer informers.GrafanaResourceDefinitionInformer,
	grdClient client.GrafanaResourceDefinitionsGetter,
	crClientGetter GRClientGetter,
) *GRDFinalizer {
	c := &GRDFinalizer{
		grdClient:      grdClient,
		grdLister:      grdInformer.Lister(),
		grdSynced:      grdInformer.Informer().HasSynced,
		crClientGetter: crClientGetter,
		queue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "grd_finalizer"),
	}

	grdInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addGrafanaResourceDefinition,
		UpdateFunc: c.updateGrafanaResourceDefinition,
	})

	c.syncFn = c.sync

	return c
}

func (c *GRDFinalizer) sync(key string) error {
	cachedGRD, err := c.grdLister.Get(key)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	// no work to do
	if cachedGRD.DeletionTimestamp.IsZero() || !kindshelpers.GRDHasFinalizer(cachedGRD, kindsv1.GrafanaResourceCleanupFinalizer) {
		return nil
	}

	grd := cachedGRD.DeepCopy()

	// update the status condition.  This cleanup could take a while.
	kindshelpers.SetGRDCondition(grd, kindsv1.GrafanaResourceDefinitionCondition{
		Type:    kindsv1.Terminating,
		Status:  kindsv1.ConditionTrue,
		Reason:  "InstanceDeletionInProgress",
		Message: "GrafanaResource deletion is in progress",
	})
	grd, err = c.grdClient.GrafanaResourceDefinitions().UpdateStatus(context.TODO(), grd, metav1.UpdateOptions{})
	if apierrors.IsNotFound(err) || apierrors.IsConflict(err) {
		// deleted or changed in the meantime, we'll get called again
		return nil
	}
	if err != nil {
		return err
	}

	// Now we can start deleting items.  We should use the REST API to ensure that all normal admission runs.
	// Since we control the endpoints, we know that delete collection works. No need to delete if not established.
	if OverlappingBuiltInResources()[schema.GroupResource{Group: grd.Spec.Group, Resource: grd.Spec.Names.Plural}] {
		// Skip deletion, explain why, and proceed to remove the finalizer and delete the GRD
		kindshelpers.SetGRDCondition(grd, kindsv1.GrafanaResourceDefinitionCondition{
			Type:    kindsv1.Terminating,
			Status:  kindsv1.ConditionFalse,
			Reason:  "OverlappingBuiltInResource",
			Message: "instances overlap with built-in resources in storage",
		})
	} else if kindshelpers.IsGRDConditionTrue(grd, kindsv1.Established) {
		cond, deleteErr := c.deleteInstances(grd)
		kindshelpers.SetGRDCondition(grd, cond)
		if deleteErr != nil {
			if _, err = c.grdClient.GrafanaResourceDefinitions().UpdateStatus(context.TODO(), grd, metav1.UpdateOptions{}); err != nil {
				utilruntime.HandleError(err)
			}
			return deleteErr
		}
	} else {
		kindshelpers.SetGRDCondition(grd, kindsv1.GrafanaResourceDefinitionCondition{
			Type:    kindsv1.Terminating,
			Status:  kindsv1.ConditionFalse,
			Reason:  "NeverEstablished",
			Message: "resource was never established",
		})
	}

	kindshelpers.GRDRemoveFinalizer(grd, kindsv1.GrafanaResourceCleanupFinalizer)
	_, err = c.grdClient.GrafanaResourceDefinitions().UpdateStatus(context.TODO(), grd, metav1.UpdateOptions{})
	if apierrors.IsNotFound(err) || apierrors.IsConflict(err) {
		// deleted or changed in the meantime, we'll get called again
		return nil
	}
	return err
}

func (c *GRDFinalizer) deleteInstances(grd *kindsv1.GrafanaResourceDefinition) (kindsv1.GrafanaResourceDefinitionCondition, error) {
	// Now we can start deleting items. While it would be ideal to use a REST API client, doing so
	// could incorrectly delete a ThirdPartyResource with the same URL as the GrafanaResource, so we go
	// directly to the storage instead. Since we control the storage, we know that delete collection works.
	crClient, err := c.crClientGetter.GetGrafanaResourceListerCollectionDeleter(grd)
	if err != nil {
		err = fmt.Errorf("unable to find a custom resource client for %s.%s: %v", grd.Status.AcceptedNames.Plural, grd.Spec.Group, err)
		return kindsv1.GrafanaResourceDefinitionCondition{
			Type:    kindsv1.Terminating,
			Status:  kindsv1.ConditionTrue,
			Reason:  "InstanceDeletionFailed",
			Message: fmt.Sprintf("could not list instances: %v", err),
		}, err
	}

	ctx := genericapirequest.NewContext()
	allResources, err := crClient.List(ctx, nil)
	if err != nil {
		return kindsv1.GrafanaResourceDefinitionCondition{
			Type:    kindsv1.Terminating,
			Status:  kindsv1.ConditionTrue,
			Reason:  "InstanceDeletionFailed",
			Message: fmt.Sprintf("could not list instances: %v", err),
		}, err
	}

	deletedNamespaces := sets.String{}
	deleteErrors := []error{}
	for _, item := range allResources.(*unstructured.UnstructuredList).Items {
		metadata, err := meta.Accessor(&item)
		if err != nil {
			utilruntime.HandleError(err)
			continue
		}
		if deletedNamespaces.Has(metadata.GetNamespace()) {
			continue
		}
		// don't retry deleting the same namespace
		deletedNamespaces.Insert(metadata.GetNamespace())
		nsCtx := genericapirequest.WithNamespace(ctx, metadata.GetNamespace())
		if _, err := crClient.DeleteCollection(nsCtx, rest.ValidateAllObjectFunc, nil, nil); err != nil {
			deleteErrors = append(deleteErrors, err)
			continue
		}
	}
	if deleteError := utilerrors.NewAggregate(deleteErrors); deleteError != nil {
		return kindsv1.GrafanaResourceDefinitionCondition{
			Type:    kindsv1.Terminating,
			Status:  kindsv1.ConditionTrue,
			Reason:  "InstanceDeletionFailed",
			Message: fmt.Sprintf("could not issue all deletes: %v", deleteError),
		}, deleteError
	}

	// now we need to wait until all the resources are deleted.  Start with a simple poll before we do anything fancy.
	// TODO not all servers are synchronized on caches.  It is possible for a stale one to still be creating things.
	// Once we have a mechanism for servers to indicate their states, we should check that for concurrence.
	err = wait.PollImmediate(5*time.Second, 1*time.Minute, func() (bool, error) {
		listObj, err := crClient.List(ctx, nil)
		if err != nil {
			return false, err
		}
		if len(listObj.(*unstructured.UnstructuredList).Items) == 0 {
			return true, nil
		}
		klog.V(2).Infof("%s.%s waiting for %d items to be removed", grd.Status.AcceptedNames.Plural, grd.Spec.Group, len(listObj.(*unstructured.UnstructuredList).Items))
		return false, nil
	})
	if err != nil {
		return kindsv1.GrafanaResourceDefinitionCondition{
			Type:    kindsv1.Terminating,
			Status:  kindsv1.ConditionTrue,
			Reason:  "InstanceDeletionCheck",
			Message: fmt.Sprintf("could not confirm zero GrafanaResources remaining: %v", err),
		}, err
	}
	return kindsv1.GrafanaResourceDefinitionCondition{
		Type:    kindsv1.Terminating,
		Status:  kindsv1.ConditionFalse,
		Reason:  "InstanceDeletionCompleted",
		Message: "removed all instances",
	}, nil
}

func (c *GRDFinalizer) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("Starting GRDFinalizer")
	defer klog.Info("Shutting down GRDFinalizer")

	if !cache.WaitForCacheSync(stopCh, c.grdSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
}

func (c *GRDFinalizer) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false when it's time to quit.
func (c *GRDFinalizer) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.syncFn(key.(string))
	if err == nil {
		c.queue.Forget(key)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("%v failed with: %v", key, err))
	c.queue.AddRateLimited(key)

	return true
}

func (c *GRDFinalizer) enqueue(obj *kindsv1.GrafanaResourceDefinition) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", obj, err))
		return
	}

	c.queue.Add(key)
}

func (c *GRDFinalizer) addGrafanaResourceDefinition(obj interface{}) {
	castObj := obj.(*kindsv1.GrafanaResourceDefinition)
	// only queue deleted things
	if !castObj.DeletionTimestamp.IsZero() && kindshelpers.GRDHasFinalizer(castObj, kindsv1.GrafanaResourceCleanupFinalizer) {
		c.enqueue(castObj)
	}
}

func (c *GRDFinalizer) updateGrafanaResourceDefinition(oldObj, newObj interface{}) {
	oldGRD := oldObj.(*kindsv1.GrafanaResourceDefinition)
	newGRD := newObj.(*kindsv1.GrafanaResourceDefinition)
	// only queue deleted things that haven't been finalized by us
	if newGRD.DeletionTimestamp.IsZero() || !kindshelpers.GRDHasFinalizer(newGRD, kindsv1.GrafanaResourceCleanupFinalizer) {
		return
	}

	// always requeue resyncs just in case
	if oldGRD.ResourceVersion == newGRD.ResourceVersion {
		c.enqueue(newGRD)
		return
	}

	// If the only difference is in the terminating condition, then there's no reason to requeue here.  This controller
	// is likely to be the originator, so requeuing would hot-loop us.  Failures are requeued by the workqueue directly.
	// This is a low traffic and scale resource, so the copy is terrible.  It's not good, so better ideas
	// are welcome.
	oldCopy := oldGRD.DeepCopy()
	newCopy := newGRD.DeepCopy()
	oldCopy.ResourceVersion = ""
	newCopy.ResourceVersion = ""
	kindshelpers.RemoveGRDCondition(oldCopy, kindsv1.Terminating)
	kindshelpers.RemoveGRDCondition(newCopy, kindsv1.Terminating)

	if !reflect.DeepEqual(oldCopy, newCopy) {
		c.enqueue(newGRD)
	}
}
