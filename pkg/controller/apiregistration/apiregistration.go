// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/kube-aggregator/blob/master/pkg/controllers/autoregister/autoregister_controller.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package apiregistration

import (
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	kindsv1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	grdinformers "github.com/grafana/grafana-apiserver/pkg/client/informers/externalversions/kinds/v1"
	grdlisters "github.com/grafana/grafana-apiserver/pkg/client/listers/kinds/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	v1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

// AutoAPIServiceRegistration is an interface which callers can re-declare locally and properly cast to for
// adding and removing APIServices
type AutoAPIServiceRegistration interface {
	// AddAPIServiceToSync adds an API service to auto-register.
	AddAPIServiceToSync(in *v1.APIService)
	// RemoveAPIServiceToSync removes an API service to auto-register.
	RemoveAPIServiceToSync(name string)
}

type GRDRegistrationController struct {
	grdLister grdlisters.GrafanaResourceDefinitionLister
	grdSynced cache.InformerSynced

	apiServiceRegistration AutoAPIServiceRegistration

	hostPort int32

	syncHandler func(groupVersion schema.GroupVersion) error

	syncedInitialSet chan struct{}

	// queue is where incoming work is placed to de-dup and to allow "easy" rate limited requeues on errors
	// this is actually keyed by a groupVersion
	queue workqueue.RateLimitingInterface
}

// NewGRDRegistrationController returns a controller which will register GRD GroupVersions with the auto APIService registration
// controller so they automatically stay in sync.
func NewGRDRegistrationController(grdinformer grdinformers.GrafanaResourceDefinitionInformer, apiServiceRegistration AutoAPIServiceRegistration, hostPort int32) *GRDRegistrationController {
	c := &GRDRegistrationController{
		grdLister:              grdinformer.Lister(),
		grdSynced:              grdinformer.Informer().HasSynced,
		apiServiceRegistration: apiServiceRegistration,
		syncedInitialSet:       make(chan struct{}),
		queue:                  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "grd_autoregistration_controller"),
	}
	c.syncHandler = c.handleVersionUpdate
	c.hostPort = hostPort

	grdinformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			cast := obj.(*kindsv1.GrafanaResourceDefinition)
			c.enqueueGRD(cast)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			// Enqueue both old and new object to make sure we remove and add appropriate API services.
			// The working queue will resolve any duplicates and only changes will stay in the queue.
			c.enqueueGRD(oldObj.(*kindsv1.GrafanaResourceDefinition))
			c.enqueueGRD(newObj.(*kindsv1.GrafanaResourceDefinition))
		},
		DeleteFunc: func(obj interface{}) {
			cast, ok := obj.(*kindsv1.GrafanaResourceDefinition)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					klog.V(2).Infof("Couldn't get object from tombstone %#v", obj)
					return
				}
				cast, ok = tombstone.Obj.(*kindsv1.GrafanaResourceDefinition)
				if !ok {
					klog.V(2).Infof("Tombstone contained unexpected object: %#v", obj)
					return
				}
			}
			c.enqueueGRD(cast)
		},
	})

	return c
}

func (c *GRDRegistrationController) Run(workers int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()
	// make sure the work queue is shutdown which will trigger workers to end
	defer c.queue.ShutDown()

	klog.Infof("Starting grd-autoregister controller")
	defer klog.Infof("Shutting down grd-autoregister controller")

	// wait for your secondary caches to fill before starting your work
	if !cache.WaitForNamedCacheSync("grd-autoregister", stopCh, c.grdSynced) {
		return
	}

	// process each item in the list once
	if grds, err := c.grdLister.List(labels.Everything()); err != nil {
		utilruntime.HandleError(err)
	} else {
		for _, grd := range grds {
			for _, version := range grd.Spec.Versions {
				if err := c.syncHandler(schema.GroupVersion{Group: grd.Spec.Group, Version: version.Name}); err != nil {
					utilruntime.HandleError(err)
				}
			}
		}
	}
	close(c.syncedInitialSet)

	// start up your worker threads based on workers.  Some controllers have multiple kinds of workers
	for i := 0; i < workers; i++ {
		// runWorker will loop until "something bad" happens.  The .Until will then rekick the worker
		// after one second
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	// wait until we're told to stop
	<-stopCh
}

// WaitForInitialSync blocks until the initial set of GRD resources has been processed
func (c *GRDRegistrationController) WaitForInitialSync() {
	<-c.syncedInitialSet
}

func (c *GRDRegistrationController) runWorker() {
	// hot loop until we're told to stop.  processNextWorkItem will automatically wait until there's work
	// available, so we don't worry about secondary waits
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false when it's time to quit.
func (c *GRDRegistrationController) processNextWorkItem() bool {
	// pull the next work item from queue.  It should be a key we use to lookup something in a cache
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	// you always have to indicate to the queue that you've completed a piece of work
	defer c.queue.Done(key)

	// do your work on the key.  This method will contains your "do stuff" logic
	err := c.syncHandler(key.(schema.GroupVersion))
	if err == nil {
		// if you had no error, tell the queue to stop tracking history for your key.  This will
		// reset things like failure counts for per-item rate limiting
		c.queue.Forget(key)
		return true
	}

	// there was a failure so be sure to report it.  This method allows for pluggable error handling
	// which can be used for things like cluster-monitoring
	utilruntime.HandleError(fmt.Errorf("%v failed with : %v", key, err))
	// since we failed, we should requeue the item to work on later.  This method will add a backoff
	// to avoid hotlooping on particular items (they're probably still not going to work right away)
	// and overall controller protection (everything I've done is broken, this controller needs to
	// calm down or it can starve other useful work) cases.
	c.queue.AddRateLimited(key)

	return true
}

func (c *GRDRegistrationController) enqueueGRD(grd *kindsv1.GrafanaResourceDefinition) {
	for _, version := range grd.Spec.Versions {
		c.queue.Add(schema.GroupVersion{Group: grd.Spec.Group, Version: version.Name})
	}
}

func (c *GRDRegistrationController) handleVersionUpdate(groupVersion schema.GroupVersion) error {
	apiServiceName := groupVersion.Version + "." + groupVersion.Group

	// check all GRDs.  There shouldn't that many, but if we have problems later we can index them
	grds, err := c.grdLister.List(labels.Everything())
	if err != nil {
		return err
	}
	for _, grd := range grds {
		if grd.Spec.Group != groupVersion.Group {
			continue
		}
		for _, version := range grd.Spec.Versions {
			if version.Name != groupVersion.Version || !version.Served {
				continue
			}

			c.apiServiceRegistration.AddAPIServiceToSync(&v1.APIService{
				ObjectMeta: metav1.ObjectMeta{Name: apiServiceName},
				Spec: v1.APIServiceSpec{
					InsecureSkipTLSVerify: true,
					Service: &v1.ServiceReference{
						Namespace: "grafana",
						Name:      "ext-api",
						Port:      &c.hostPort,
					},
					Group:                groupVersion.Group,
					Version:              groupVersion.Version,
					GroupPriorityMinimum: 1000, // GRDs should have relatively low priority
					VersionPriority:      100,  // GRDs will be sorted by kube-like versions like any other APIService with the same VersionPriority
				},
			})
			return nil
		}
	}

	c.apiServiceRegistration.RemoveAPIServiceToSync(apiServiceName)
	return nil
}
