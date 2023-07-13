/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package establish

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	kindshelpers "github.com/grafana/grafana-apiserver/pkg/apihelpers"
	kindsv1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	client "github.com/grafana/grafana-apiserver/pkg/client/clientset/clientset/typed/kinds/v1"
	informers "github.com/grafana/grafana-apiserver/pkg/client/informers/externalversions/kinds/v1"
	listers "github.com/grafana/grafana-apiserver/pkg/client/listers/kinds/v1"
)

// EstablishingController controls how and when GRD is established.
type EstablishingController struct {
	grdClient client.GrafanaResourceDefinitionsGetter
	grdLister listers.GrafanaResourceDefinitionLister
	grdSynced cache.InformerSynced

	// To allow injection for testing.
	syncFn func(key string) error

	queue workqueue.RateLimitingInterface
}

// NewEstablishingController creates new EstablishingController.
func NewEstablishingController(grdInformer informers.GrafanaResourceDefinitionInformer,
	grdClient client.GrafanaResourceDefinitionsGetter) *EstablishingController {
	ec := &EstablishingController{
		grdClient: grdClient,
		grdLister: grdInformer.Lister(),
		grdSynced: grdInformer.Informer().HasSynced,
		queue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "grdEstablishing"),
	}

	ec.syncFn = ec.sync

	return ec
}

// QueueGRD adds GRD into the establishing queue.
func (ec *EstablishingController) QueueGRD(key string, timeout time.Duration) {
	ec.queue.AddAfter(key, timeout)
}

// Run starts the EstablishingController.
func (ec *EstablishingController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	defer ec.queue.ShutDown()

	klog.Info("Starting EstablishingController")
	defer klog.Info("Shutting down EstablishingController")

	if !cache.WaitForCacheSync(stopCh, ec.grdSynced) {
		return
	}

	// only start one worker thread since its a slow moving API
	go wait.Until(ec.runWorker, time.Second, stopCh)

	<-stopCh
}

func (ec *EstablishingController) runWorker() {
	for ec.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.
// It returns false when it's time to quit.
func (ec *EstablishingController) processNextWorkItem() bool {
	key, quit := ec.queue.Get()
	if quit {
		return false
	}
	defer ec.queue.Done(key)

	err := ec.syncFn(key.(string))
	if err == nil {
		ec.queue.Forget(key)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("%v failed with: %v", key, err))
	ec.queue.AddRateLimited(key)

	return true
}

// sync is used to turn GRDs into the Established state.
func (ec *EstablishingController) sync(key string) error {
	cachedGRD, err := ec.grdLister.Get(key)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	if !kindshelpers.IsGRDConditionTrue(cachedGRD, kindsv1.NamesAccepted) ||
		kindshelpers.IsGRDConditionTrue(cachedGRD, kindsv1.Established) {
		return nil
	}

	grd := cachedGRD.DeepCopy()
	establishedCondition := kindsv1.GrafanaResourceDefinitionCondition{
		Type:    kindsv1.Established,
		Status:  kindsv1.ConditionTrue,
		Reason:  "InitialNamesAccepted",
		Message: "the initial names have been accepted",
	}
	kindshelpers.SetGRDCondition(grd, establishedCondition)

	// Update server with new GRD condition.
	_, err = ec.grdClient.GrafanaResourceDefinitions().UpdateStatus(context.TODO(), grd, metav1.UpdateOptions{})
	if apierrors.IsNotFound(err) || apierrors.IsConflict(err) {
		// deleted or changed in the meantime, we'll get called again
		return nil
	}
	if err != nil {
		return err
	}

	return nil
}
