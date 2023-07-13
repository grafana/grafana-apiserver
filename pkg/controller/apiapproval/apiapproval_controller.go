// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/controller/apiapproval/apiapproval_controller.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package apiapproval

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/grafana/grafana-apiserver/pkg/apihelpers"
	kindsv1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	client "github.com/grafana/grafana-apiserver/pkg/client/clientset/clientset/typed/kinds/v1"
	informers "github.com/grafana/grafana-apiserver/pkg/client/informers/externalversions/kinds/v1"
	listers "github.com/grafana/grafana-apiserver/pkg/client/listers/kinds/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// KubernetesAPIApprovalPolicyConformantConditionController is maintaining the KubernetesAPIApprovalPolicyConformant condition.
type KubernetesAPIApprovalPolicyConformantConditionController struct {
	grdClient client.GrafanaResourceDefinitionsGetter

	grdLister listers.GrafanaResourceDefinitionLister
	grdSynced cache.InformerSynced

	// To allow injection for testing.
	syncFn func(key string) error

	queue workqueue.RateLimitingInterface

	// last protectedAnnotation value this controller updated the condition per GRD name (to avoid two
	// different version of the kinds-apiservers in HA to fight for the right message)
	lastSeenProtectedAnnotationLock sync.Mutex
	lastSeenProtectedAnnotation     map[string]string
}

// NewKubernetesAPIApprovalPolicyConformantConditionController constructs a KubernetesAPIApprovalPolicyConformant schema condition controller.
func NewKubernetesAPIApprovalPolicyConformantConditionController(
	grdInformer informers.GrafanaResourceDefinitionInformer,
	grdClient client.GrafanaResourceDefinitionsGetter,
) *KubernetesAPIApprovalPolicyConformantConditionController {
	c := &KubernetesAPIApprovalPolicyConformantConditionController{
		grdClient:                   grdClient,
		grdLister:                   grdInformer.Lister(),
		grdSynced:                   grdInformer.Informer().HasSynced,
		queue:                       workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "kubernetes_api_approval_conformant_condition_controller"),
		lastSeenProtectedAnnotation: map[string]string{},
	}

	grdInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addGrafanaResourceDefinition,
		UpdateFunc: c.updateGrafanaResourceDefinition,
		DeleteFunc: c.deleteGrafanaResourceDefinition,
	})

	c.syncFn = c.sync

	return c
}

// calculateCondition determines the new KubernetesAPIApprovalPolicyConformant condition
func calculateCondition(grd *kindsv1.GrafanaResourceDefinition) *kindsv1.GrafanaResourceDefinitionCondition {
	if !apihelpers.IsProtectedCommunityGroup(grd.Spec.Group) {
		return nil
	}

	approvalState, reason := apihelpers.GetAPIApprovalState(grd.Annotations)
	switch approvalState {
	case apihelpers.APIApprovalInvalid:
		return &kindsv1.GrafanaResourceDefinitionCondition{
			Type:    kindsv1.KubernetesAPIApprovalPolicyConformant,
			Status:  kindsv1.ConditionFalse,
			Reason:  "InvalidAnnotation",
			Message: reason,
		}
	case apihelpers.APIApprovalMissing:
		return &kindsv1.GrafanaResourceDefinitionCondition{
			Type:    kindsv1.KubernetesAPIApprovalPolicyConformant,
			Status:  kindsv1.ConditionFalse,
			Reason:  "MissingAnnotation",
			Message: reason,
		}
	case apihelpers.APIApproved:
		return &kindsv1.GrafanaResourceDefinitionCondition{
			Type:    kindsv1.KubernetesAPIApprovalPolicyConformant,
			Status:  kindsv1.ConditionTrue,
			Reason:  "ApprovedAnnotation",
			Message: reason,
		}
	case apihelpers.APIApprovalBypassed:
		return &kindsv1.GrafanaResourceDefinitionCondition{
			Type:    kindsv1.KubernetesAPIApprovalPolicyConformant,
			Status:  kindsv1.ConditionFalse,
			Reason:  "UnapprovedAnnotation",
			Message: reason,
		}
	default:
		return &kindsv1.GrafanaResourceDefinitionCondition{
			Type:    kindsv1.KubernetesAPIApprovalPolicyConformant,
			Status:  kindsv1.ConditionUnknown,
			Reason:  "UnknownAnnotation",
			Message: reason,
		}
	}
}

func (c *KubernetesAPIApprovalPolicyConformantConditionController) sync(key string) error {
	inGrafanaResourceDefinition, err := c.grdLister.Get(key)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	// avoid repeated calculation for the same annotation
	protectionAnnotationValue := inGrafanaResourceDefinition.Annotations[kindsv1.KubeAPIApprovedAnnotation]
	c.lastSeenProtectedAnnotationLock.Lock()
	lastSeen, seenBefore := c.lastSeenProtectedAnnotation[inGrafanaResourceDefinition.Name]
	c.lastSeenProtectedAnnotationLock.Unlock()
	if seenBefore && protectionAnnotationValue == lastSeen {
		return nil
	}

	// check old condition
	cond := calculateCondition(inGrafanaResourceDefinition)
	if cond == nil {
		// because group is immutable, if we have no condition now, we have no need to remove a condition.
		return nil
	}
	old := apihelpers.FindGRDCondition(inGrafanaResourceDefinition, kindsv1.KubernetesAPIApprovalPolicyConformant)

	// don't attempt a write if all the condition details are the same
	if old != nil && old.Status == cond.Status && old.Reason == cond.Reason && old.Message == cond.Message {
		// no need to update annotation because we took no action.
		return nil
	}

	// update condition
	grd := inGrafanaResourceDefinition.DeepCopy()
	apihelpers.SetGRDCondition(grd, *cond)

	_, err = c.grdClient.GrafanaResourceDefinitions().UpdateStatus(context.TODO(), grd, metav1.UpdateOptions{})
	if apierrors.IsNotFound(err) || apierrors.IsConflict(err) {
		// deleted or changed in the meantime, we'll get called again
		return nil
	}
	if err != nil {
		return err
	}

	// store annotation in order to avoid repeated updates for the same annotation (and potential
	// fights of API server in HA environments).
	c.lastSeenProtectedAnnotationLock.Lock()
	defer c.lastSeenProtectedAnnotationLock.Unlock()
	c.lastSeenProtectedAnnotation[grd.Name] = protectionAnnotationValue

	return nil
}

// Run starts the controller.
func (c *KubernetesAPIApprovalPolicyConformantConditionController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()

	klog.Infof("Starting KubernetesAPIApprovalPolicyConformantConditionController")
	defer klog.Infof("Shutting down KubernetesAPIApprovalPolicyConformantConditionController")

	if !cache.WaitForCacheSync(stopCh, c.grdSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
}

func (c *KubernetesAPIApprovalPolicyConformantConditionController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false when it's time to quit.
func (c *KubernetesAPIApprovalPolicyConformantConditionController) processNextWorkItem() bool {
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

func (c *KubernetesAPIApprovalPolicyConformantConditionController) enqueue(obj *kindsv1.GrafanaResourceDefinition) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for object %#v: %v", obj, err))
		return
	}

	c.queue.Add(key)
}

func (c *KubernetesAPIApprovalPolicyConformantConditionController) addGrafanaResourceDefinition(obj interface{}) {
	castObj := obj.(*kindsv1.GrafanaResourceDefinition)
	klog.V(4).Infof("Adding %s", castObj.Name)
	c.enqueue(castObj)
}

func (c *KubernetesAPIApprovalPolicyConformantConditionController) updateGrafanaResourceDefinition(obj, _ interface{}) {
	castObj := obj.(*kindsv1.GrafanaResourceDefinition)
	klog.V(4).Infof("Updating %s", castObj.Name)
	c.enqueue(castObj)
}

func (c *KubernetesAPIApprovalPolicyConformantConditionController) deleteGrafanaResourceDefinition(obj interface{}) {
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

	c.lastSeenProtectedAnnotationLock.Lock()
	defer c.lastSeenProtectedAnnotationLock.Unlock()
	delete(c.lastSeenProtectedAnnotation, castObj.Name)
}
