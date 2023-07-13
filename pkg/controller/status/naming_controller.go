// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/controller/status/naming_controller.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package status

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"k8s.io/klog/v2"

	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	kindshelpers "github.com/grafana/grafana-apiserver/pkg/apihelpers"
	kindsv1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	client "github.com/grafana/grafana-apiserver/pkg/client/clientset/clientset/typed/kinds/v1"
	informers "github.com/grafana/grafana-apiserver/pkg/client/informers/externalversions/kinds/v1"
	listers "github.com/grafana/grafana-apiserver/pkg/client/listers/kinds/v1"
)

// This controller is reserving names. To avoid conflicts, be sure to run only one instance of the worker at a time.
// This could eventually be lifted, but starting simple.
type NamingConditionController struct {
	grdClient client.GrafanaResourceDefinitionsGetter

	grdLister listers.GrafanaResourceDefinitionLister
	grdSynced cache.InformerSynced
	// grdMutationCache backs our lister and keeps track of committed updates to avoid racy
	// write/lookup cycles.  It's got 100 slots by default, so it unlikely to overrun
	// TODO to revisit this if naming conflicts are found to occur in the wild
	grdMutationCache cache.MutationCache

	// To allow injection for testing.
	syncFn func(key string) error

	queue workqueue.RateLimitingInterface
}

func NewNamingConditionController(
	grdInformer informers.GrafanaResourceDefinitionInformer,
	grdClient client.GrafanaResourceDefinitionsGetter,
) *NamingConditionController {
	c := &NamingConditionController{
		grdClient: grdClient,
		grdLister: grdInformer.Lister(),
		grdSynced: grdInformer.Informer().HasSynced,
		queue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "grd_naming_condition_controller"),
	}

	informerIndexer := grdInformer.Informer().GetIndexer()
	c.grdMutationCache = cache.NewIntegerResourceVersionMutationCache(informerIndexer, informerIndexer, 60*time.Second, false)

	grdInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addGrafanaResourceDefinition,
		UpdateFunc: c.updateGrafanaResourceDefinition,
		DeleteFunc: c.deleteGrafanaResourceDefinition,
	})

	c.syncFn = c.sync

	return c
}

func (c *NamingConditionController) getAcceptedNamesForGroup(group string) (allResources sets.String, allKinds sets.String) {
	allResources = sets.String{}
	allKinds = sets.String{}

	list, err := c.grdLister.List(labels.Everything())
	if err != nil {
		panic(err)
	}

	for _, curr := range list {
		if curr.Spec.Group != group {
			continue
		}

		// for each item here, see if we have a mutation cache entry that is more recent
		// this makes sure that if we tight loop on update and run, our mutation cache will show
		// us the version of the objects we just updated to.
		item := curr
		obj, exists, err := c.grdMutationCache.GetByKey(curr.Name)
		if exists && err == nil {
			item = obj.(*kindsv1.GrafanaResourceDefinition)
		}

		allResources.Insert(item.Status.AcceptedNames.Plural)
		allResources.Insert(item.Status.AcceptedNames.Singular)
		allResources.Insert(item.Status.AcceptedNames.ShortNames...)

		allKinds.Insert(item.Status.AcceptedNames.Kind)
		allKinds.Insert(item.Status.AcceptedNames.ListKind)
	}

	return allResources, allKinds
}

func (c *NamingConditionController) calculateNamesAndConditions(in *kindsv1.GrafanaResourceDefinition) (kindsv1.GrafanaResourceDefinitionNames, kindsv1.GrafanaResourceDefinitionCondition, kindsv1.GrafanaResourceDefinitionCondition) {
	// Get the names that have already been claimed
	allResources, allKinds := c.getAcceptedNamesForGroup(in.Spec.Group)

	namesAcceptedCondition := kindsv1.GrafanaResourceDefinitionCondition{
		Type:   kindsv1.NamesAccepted,
		Status: kindsv1.ConditionUnknown,
	}

	requestedNames := in.Spec.Names
	acceptedNames := in.Status.AcceptedNames
	newNames := in.Status.AcceptedNames

	// Check each name for mismatches.  If there's a mismatch between spec and status, then try to deconflict.
	// Continue on errors so that the status is the best match possible
	if err := equalToAcceptedOrFresh(requestedNames.Plural, acceptedNames.Plural, allResources); err != nil {
		namesAcceptedCondition.Status = kindsv1.ConditionFalse
		namesAcceptedCondition.Reason = "PluralConflict"
		namesAcceptedCondition.Message = err.Error()
	} else {
		newNames.Plural = requestedNames.Plural
	}
	if err := equalToAcceptedOrFresh(requestedNames.Singular, acceptedNames.Singular, allResources); err != nil {
		namesAcceptedCondition.Status = kindsv1.ConditionFalse
		namesAcceptedCondition.Reason = "SingularConflict"
		namesAcceptedCondition.Message = err.Error()
	} else {
		newNames.Singular = requestedNames.Singular
	}
	if !reflect.DeepEqual(requestedNames.ShortNames, acceptedNames.ShortNames) {
		errs := []error{}
		existingShortNames := sets.NewString(acceptedNames.ShortNames...)
		for _, shortName := range requestedNames.ShortNames {
			// if the shortname is already ours, then we're fine
			if existingShortNames.Has(shortName) {
				continue
			}
			if err := equalToAcceptedOrFresh(shortName, "", allResources); err != nil {
				errs = append(errs, err)
			}

		}
		if err := utilerrors.NewAggregate(errs); err != nil {
			namesAcceptedCondition.Status = kindsv1.ConditionFalse
			namesAcceptedCondition.Reason = "ShortNamesConflict"
			namesAcceptedCondition.Message = err.Error()
		} else {
			newNames.ShortNames = requestedNames.ShortNames
		}
	}

	if err := equalToAcceptedOrFresh(requestedNames.Kind, acceptedNames.Kind, allKinds); err != nil {
		namesAcceptedCondition.Status = kindsv1.ConditionFalse
		namesAcceptedCondition.Reason = "KindConflict"
		namesAcceptedCondition.Message = err.Error()
	} else {
		newNames.Kind = requestedNames.Kind
	}
	if err := equalToAcceptedOrFresh(requestedNames.ListKind, acceptedNames.ListKind, allKinds); err != nil {
		namesAcceptedCondition.Status = kindsv1.ConditionFalse
		namesAcceptedCondition.Reason = "ListKindConflict"
		namesAcceptedCondition.Message = err.Error()
	} else {
		newNames.ListKind = requestedNames.ListKind
	}

	newNames.Categories = requestedNames.Categories

	// if we haven't changed the condition, then our names must be good.
	if namesAcceptedCondition.Status == kindsv1.ConditionUnknown {
		namesAcceptedCondition.Status = kindsv1.ConditionTrue
		namesAcceptedCondition.Reason = "NoConflicts"
		namesAcceptedCondition.Message = "no conflicts found"
	}

	// set EstablishedCondition initially to false, then set it to true in establishing controller.
	// The Establishing Controller will see the NamesAccepted condition when it arrives through the shared informer.
	// At that time the API endpoint handler will serve the endpoint, avoiding a race
	// which we had if we set Established to true here.
	establishedCondition := kindsv1.GrafanaResourceDefinitionCondition{
		Type:    kindsv1.Established,
		Status:  kindsv1.ConditionFalse,
		Reason:  "NotAccepted",
		Message: "not all names are accepted",
	}
	if old := kindshelpers.FindGRDCondition(in, kindsv1.Established); old != nil {
		establishedCondition = *old
	}
	if establishedCondition.Status != kindsv1.ConditionTrue && namesAcceptedCondition.Status == kindsv1.ConditionTrue {
		establishedCondition = kindsv1.GrafanaResourceDefinitionCondition{
			Type:    kindsv1.Established,
			Status:  kindsv1.ConditionFalse,
			Reason:  "Installing",
			Message: "the initial names have been accepted",
		}
	}

	return newNames, namesAcceptedCondition, establishedCondition
}

func equalToAcceptedOrFresh(requestedName, acceptedName string, usedNames sets.String) error {
	if requestedName == acceptedName {
		return nil
	}
	if !usedNames.Has(requestedName) {
		return nil
	}

	return fmt.Errorf("%q is already in use", requestedName)
}

func (c *NamingConditionController) sync(key string) error {
	inGrafanaResourceDefinition, err := c.grdLister.Get(key)
	if apierrors.IsNotFound(err) {
		// GRD was deleted and has freed its names.
		// Reconsider all other GRDs in the same group.
		if err := c.requeueAllOtherGroupGRDs(key); err != nil {
			return err
		}
		return nil
	}
	if err != nil {
		return err
	}

	// Skip checking names if Spec and Status names are same.
	if equality.Semantic.DeepEqual(inGrafanaResourceDefinition.Spec.Names, inGrafanaResourceDefinition.Status.AcceptedNames) {
		return nil
	}

	acceptedNames, namingCondition, establishedCondition := c.calculateNamesAndConditions(inGrafanaResourceDefinition)

	// nothing to do if accepted names and NamesAccepted condition didn't change
	if reflect.DeepEqual(inGrafanaResourceDefinition.Status.AcceptedNames, acceptedNames) &&
		kindshelpers.IsGRDConditionEquivalent(&namingCondition, kindshelpers.FindGRDCondition(inGrafanaResourceDefinition, kindsv1.NamesAccepted)) {
		return nil
	}

	grd := inGrafanaResourceDefinition.DeepCopy()
	grd.Status.AcceptedNames = acceptedNames
	kindshelpers.SetGRDCondition(grd, namingCondition)
	kindshelpers.SetGRDCondition(grd, establishedCondition)

	updatedObj, err := c.grdClient.GrafanaResourceDefinitions().UpdateStatus(context.TODO(), grd, metav1.UpdateOptions{})
	if apierrors.IsNotFound(err) || apierrors.IsConflict(err) {
		// deleted or changed in the meantime, we'll get called again
		return nil
	}
	if err != nil {
		return err
	}

	// if the update was successful, go ahead and add the entry to the mutation cache
	c.grdMutationCache.Mutation(updatedObj)

	// we updated our status, so we may be releasing a name.  When this happens, we need to rekick everything in our group
	// if we fail to rekick, just return as normal.  We'll get everything on a resync
	if err := c.requeueAllOtherGroupGRDs(key); err != nil {
		return err
	}

	return nil
}

func (c *NamingConditionController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Info("Starting NamingConditionController")
	defer klog.Info("Shutting down NamingConditionController")

	if !cache.WaitForCacheSync(stopCh, c.grdSynced) {
		return
	}

	// only start one worker thread since its a slow moving API and the naming conflict resolution bits aren't thread-safe
	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
}

func (c *NamingConditionController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false when it's time to quit.
func (c *NamingConditionController) processNextWorkItem() bool {
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

func (c *NamingConditionController) enqueue(obj *kindsv1.GrafanaResourceDefinition) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", obj, err))
		return
	}

	c.queue.Add(key)
}

func (c *NamingConditionController) addGrafanaResourceDefinition(obj interface{}) {
	castObj := obj.(*kindsv1.GrafanaResourceDefinition)
	klog.V(4).Infof("Adding %s", castObj.Name)
	c.enqueue(castObj)
}

func (c *NamingConditionController) updateGrafanaResourceDefinition(obj, _ interface{}) {
	castObj := obj.(*kindsv1.GrafanaResourceDefinition)
	klog.V(4).Infof("Updating %s", castObj.Name)
	c.enqueue(castObj)
}

func (c *NamingConditionController) deleteGrafanaResourceDefinition(obj interface{}) {
	castObj, ok := obj.(*kindsv1.GrafanaResourceDefinition)
	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			klog.Errorf("Couldn't get object from tombstone %#v", obj)
			return
		}
		castObj, ok = tombstone.Obj.(*kindsv1.GrafanaResourceDefinition)
		if !ok {
			klog.Errorf("Tombstone contained object that is not expected %#v", obj)
			return
		}
	}
	klog.V(4).Infof("Deleting %q", castObj.Name)
	c.enqueue(castObj)
}

func (c *NamingConditionController) requeueAllOtherGroupGRDs(name string) error {
	pluralGroup := strings.SplitN(name, ".", 2)
	list, err := c.grdLister.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, curr := range list {
		if curr.Spec.Group == pluralGroup[1] && curr.Name != name {
			c.queue.Add(curr.Name)
		}
	}
	return nil
}
