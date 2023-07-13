// SPDX-License-Identifier: AGPL-3.0-only
// Provenance-includes-location: https://github.com/kubernetes/apiextensions-apiserver/blob/master/pkg/apiserver/customresource_discovery_controller.go
// Provenance-includes-license: Apache-2.0
// Provenance-includes-copyright: The Kubernetes Authors.

package apiserver

import (
	"fmt"
	"sort"
	"time"

	"k8s.io/klog/v2"

	apidiscoveryv2beta1 "k8s.io/api/apidiscovery/v2beta1"
	autoscaling "k8s.io/api/autoscaling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/endpoints/discovery"
	discoveryendpoint "k8s.io/apiserver/pkg/endpoints/discovery/aggregated"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	kindshelpers "github.com/grafana/grafana-apiserver/pkg/apihelpers"
	kindsv1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	informers "github.com/grafana/grafana-apiserver/pkg/client/informers/externalversions/kinds/v1"
	listers "github.com/grafana/grafana-apiserver/pkg/client/listers/kinds/v1"
)

type DiscoveryController struct {
	versionHandler  *versionDiscoveryHandler
	groupHandler    *groupDiscoveryHandler
	resourceManager discoveryendpoint.ResourceManager

	grdLister  listers.GrafanaResourceDefinitionLister
	grdsSynced cache.InformerSynced

	// To allow injection for testing.
	syncFn func(version schema.GroupVersion) error

	queue workqueue.RateLimitingInterface
}

func NewDiscoveryController(
	grdInformer informers.GrafanaResourceDefinitionInformer,
	versionHandler *versionDiscoveryHandler,
	groupHandler *groupDiscoveryHandler,
	resourceManager discoveryendpoint.ResourceManager,
) *DiscoveryController {
	c := &DiscoveryController{
		versionHandler:  versionHandler,
		groupHandler:    groupHandler,
		resourceManager: resourceManager,
		grdLister:       grdInformer.Lister(),
		grdsSynced:      grdInformer.Informer().HasSynced,

		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "DiscoveryController"),
	}

	grdInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.addGrafanaResourceDefinition,
		UpdateFunc: c.updateGrafanaResourceDefinition,
		DeleteFunc: c.deleteGrafanaResourceDefinition,
	})

	c.syncFn = c.sync

	return c
}

func (c *DiscoveryController) sync(version schema.GroupVersion) error {

	apiVersionsForDiscovery := []metav1.GroupVersionForDiscovery{}
	apiResourcesForDiscovery := []metav1.APIResource{}
	aggregatedApiResourcesForDiscovery := []apidiscoveryv2beta1.APIResourceDiscovery{}
	versionsForDiscoveryMap := map[metav1.GroupVersion]bool{}

	grds, err := c.grdLister.List(labels.Everything())
	if err != nil {
		return err
	}
	foundVersion := false
	foundGroup := false
	for _, grd := range grds {
		if !kindshelpers.IsGRDConditionTrue(grd, kindsv1.Established) {
			continue
		}

		if grd.Spec.Group != version.Group {
			continue
		}

		foundThisVersion := false
		var storageVersionHash string
		for _, v := range grd.Spec.Versions {
			if !v.Served {
				continue
			}
			// If there is any Served version, that means the group should show up in discovery
			foundGroup = true

			gv := metav1.GroupVersion{Group: grd.Spec.Group, Version: v.Name}
			if !versionsForDiscoveryMap[gv] {
				versionsForDiscoveryMap[gv] = true
				apiVersionsForDiscovery = append(apiVersionsForDiscovery, metav1.GroupVersionForDiscovery{
					GroupVersion: grd.Spec.Group + "/" + v.Name,
					Version:      v.Name,
				})
			}
			if v.Name == version.Version {
				foundThisVersion = true
			}
			if v.Storage {
				storageVersionHash = discovery.StorageVersionHash(gv.Group, gv.Version, grd.Spec.Names.Kind)
			}
		}

		if !foundThisVersion {
			continue
		}
		foundVersion = true

		verbs := metav1.Verbs([]string{"delete", "deletecollection", "get", "list", "patch", "create", "update", "watch"})
		// if we're terminating we don't allow some verbs
		if kindshelpers.IsGRDConditionTrue(grd, kindsv1.Terminating) {
			verbs = metav1.Verbs([]string{"delete", "deletecollection", "get", "list", "watch"})
		}

		apiResourcesForDiscovery = append(apiResourcesForDiscovery, metav1.APIResource{
			Name:               grd.Status.AcceptedNames.Plural,
			SingularName:       grd.Status.AcceptedNames.Singular,
			Namespaced:         grd.Spec.Scope == kindsv1.NamespaceScoped,
			Kind:               grd.Status.AcceptedNames.Kind,
			Verbs:              verbs,
			ShortNames:         grd.Status.AcceptedNames.ShortNames,
			Categories:         grd.Status.AcceptedNames.Categories,
			StorageVersionHash: storageVersionHash,
		})

		subresources, err := kindshelpers.GetSubresourcesForVersion(grd, version.Version)
		if err != nil {
			return err
		}

		if c.resourceManager != nil {
			var scope apidiscoveryv2beta1.ResourceScope
			if grd.Spec.Scope == kindsv1.NamespaceScoped {
				scope = apidiscoveryv2beta1.ScopeNamespace
			} else {
				scope = apidiscoveryv2beta1.ScopeCluster
			}
			apiResourceDiscovery := apidiscoveryv2beta1.APIResourceDiscovery{
				Resource:         grd.Status.AcceptedNames.Plural,
				SingularResource: grd.Status.AcceptedNames.Singular,
				Scope:            scope,
				ResponseKind: &metav1.GroupVersionKind{
					Group:   version.Group,
					Version: version.Version,
					Kind:    grd.Status.AcceptedNames.Kind,
				},
				Verbs:      verbs,
				ShortNames: grd.Status.AcceptedNames.ShortNames,
				Categories: grd.Status.AcceptedNames.Categories,
			}
			if subresources != nil && subresources.Ref != nil {
				apiResourceDiscovery.Subresources = append(apiResourceDiscovery.Subresources, apidiscoveryv2beta1.APISubresourceDiscovery{
					Subresource: "ref",
					ResponseKind: &metav1.GroupVersionKind{
						Group:   version.Group,
						Version: version.Version,
						Kind:    grd.Status.AcceptedNames.Kind,
					},
					Verbs: metav1.Verbs([]string{"get"}),
				})
			}
			if subresources != nil && subresources.History != nil {
				apiResourceDiscovery.Subresources = append(apiResourceDiscovery.Subresources, apidiscoveryv2beta1.APISubresourceDiscovery{
					Subresource: "history",
					ResponseKind: &metav1.GroupVersionKind{
						Group:   version.Group,
						Version: version.Version,
						Kind:    grd.Status.AcceptedNames.Kind,
					},
					Verbs: metav1.Verbs([]string{"get"}),
				})
			}
			if subresources != nil && subresources.Status != nil {
				apiResourceDiscovery.Subresources = append(apiResourceDiscovery.Subresources, apidiscoveryv2beta1.APISubresourceDiscovery{
					Subresource: "status",
					ResponseKind: &metav1.GroupVersionKind{
						Group:   version.Group,
						Version: version.Version,
						Kind:    grd.Status.AcceptedNames.Kind,
					},
					Verbs: metav1.Verbs([]string{"get", "patch", "update"}),
				})
			}
			if subresources != nil && subresources.Scale != nil {
				apiResourceDiscovery.Subresources = append(apiResourceDiscovery.Subresources, apidiscoveryv2beta1.APISubresourceDiscovery{
					Subresource: "scale",
					ResponseKind: &metav1.GroupVersionKind{
						Group:   autoscaling.GroupName,
						Version: "v1",
						Kind:    "Scale",
					},
					Verbs: metav1.Verbs([]string{"get", "patch", "update"}),
				})

			}
			aggregatedApiResourcesForDiscovery = append(aggregatedApiResourcesForDiscovery, apiResourceDiscovery)
		}

		if subresources != nil && subresources.Ref != nil {
			apiResourcesForDiscovery = append(apiResourcesForDiscovery, metav1.APIResource{
				Name:       grd.Status.AcceptedNames.Plural + "/ref",
				Namespaced: grd.Spec.Scope == kindsv1.NamespaceScoped,
				Kind:       grd.Status.AcceptedNames.Kind,
				Verbs:      metav1.Verbs([]string{"get"}),
			})
		}

		if subresources != nil && subresources.History != nil {
			apiResourcesForDiscovery = append(apiResourcesForDiscovery, metav1.APIResource{
				Name:       grd.Status.AcceptedNames.Plural + "/history",
				Namespaced: grd.Spec.Scope == kindsv1.NamespaceScoped,
				Kind:       grd.Status.AcceptedNames.Kind,
				Verbs:      metav1.Verbs([]string{"get"}),
			})
		}

		if subresources != nil && subresources.Status != nil {
			apiResourcesForDiscovery = append(apiResourcesForDiscovery, metav1.APIResource{
				Name:       grd.Status.AcceptedNames.Plural + "/status",
				Namespaced: grd.Spec.Scope == kindsv1.NamespaceScoped,
				Kind:       grd.Status.AcceptedNames.Kind,
				Verbs:      metav1.Verbs([]string{"get", "patch", "update"}),
			})
		}

		if subresources != nil && subresources.Scale != nil {
			apiResourcesForDiscovery = append(apiResourcesForDiscovery, metav1.APIResource{
				Group:      autoscaling.GroupName,
				Version:    "v1",
				Kind:       "Scale",
				Name:       grd.Status.AcceptedNames.Plural + "/scale",
				Namespaced: grd.Spec.Scope == kindsv1.NamespaceScoped,
				Verbs:      metav1.Verbs([]string{"get", "patch", "update"}),
			})
		}
	}

	if !foundGroup {
		c.groupHandler.unsetDiscovery(version.Group)
		c.versionHandler.unsetDiscovery(version)

		if c.resourceManager != nil {
			c.resourceManager.RemoveGroup(version.Group)
		}
		return nil
	}

	sortGroupDiscoveryByKubeAwareVersion(apiVersionsForDiscovery)

	apiGroup := metav1.APIGroup{
		Name:     version.Group,
		Versions: apiVersionsForDiscovery,
		// the preferred versions for a group is the first item in
		// apiVersionsForDiscovery after it put in the right ordered
		PreferredVersion: apiVersionsForDiscovery[0],
	}
	c.groupHandler.setDiscovery(version.Group, discovery.NewAPIGroupHandler(Codecs, apiGroup))

	if !foundVersion {
		c.versionHandler.unsetDiscovery(version)

		if c.resourceManager != nil {
			c.resourceManager.RemoveGroupVersion(metav1.GroupVersion{
				Group:   version.Group,
				Version: version.Version,
			})
		}
		return nil
	}
	c.versionHandler.setDiscovery(version, discovery.NewAPIVersionHandler(Codecs, version, discovery.APIResourceListerFunc(func() []metav1.APIResource {
		return apiResourcesForDiscovery
	})))

	sort.Slice(aggregatedApiResourcesForDiscovery[:], func(i, j int) bool {
		return aggregatedApiResourcesForDiscovery[i].Resource < aggregatedApiResourcesForDiscovery[j].Resource
	})
	if c.resourceManager != nil {
		c.resourceManager.AddGroupVersion(version.Group, apidiscoveryv2beta1.APIVersionDiscovery{
			Freshness: apidiscoveryv2beta1.DiscoveryFreshnessCurrent,
			Version:   version.Version,
			Resources: aggregatedApiResourcesForDiscovery,
		})
		// Default priority for GRDs
		c.resourceManager.SetGroupVersionPriority(metav1.GroupVersion(version), 1000, 100)
	}
	return nil
}

func sortGroupDiscoveryByKubeAwareVersion(gd []metav1.GroupVersionForDiscovery) {
	sort.Slice(gd, func(i, j int) bool {
		return version.CompareKubeAwareVersionStrings(gd[i].Version, gd[j].Version) > 0
	})
}

func (c *DiscoveryController) Run(stopCh <-chan struct{}, synchedCh chan<- struct{}) {
	defer utilruntime.HandleCrash()
	defer c.queue.ShutDown()
	defer klog.Info("Shutting down DiscoveryController")

	klog.Info("Starting DiscoveryController")

	if !cache.WaitForCacheSync(stopCh, c.grdsSynced) {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for caches to sync"))
		return
	}

	// initially sync all group versions to make sure we serve complete discovery
	if err := wait.PollImmediateUntil(time.Second, func() (bool, error) {
		grds, err := c.grdLister.List(labels.Everything())
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to initially list GRDs: %v", err))
			return false, nil
		}
		for _, grd := range grds {
			for _, v := range grd.Spec.Versions {
				gv := schema.GroupVersion{Group: grd.Spec.Group, Version: v.Name}
				if err := c.sync(gv); err != nil {
					utilruntime.HandleError(fmt.Errorf("failed to initially sync GRD version %v: %v", gv, err))
					return false, nil
				}
			}
		}
		return true, nil
	}, stopCh); err == wait.ErrWaitTimeout {
		utilruntime.HandleError(fmt.Errorf("timed out waiting for discovery endpoint to initialize"))
		return
	} else if err != nil {
		panic(fmt.Errorf("unexpected error: %v", err))
	}
	close(synchedCh)

	// only start one worker thread since its a slow moving API
	go wait.Until(c.runWorker, time.Second, stopCh)

	<-stopCh
}

func (c *DiscoveryController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem deals with one key off the queue.  It returns false when it's time to quit.
func (c *DiscoveryController) processNextWorkItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.syncFn(key.(schema.GroupVersion))
	if err == nil {
		c.queue.Forget(key)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("%v failed with: %v", key, err))
	c.queue.AddRateLimited(key)

	return true
}

func (c *DiscoveryController) enqueue(obj *kindsv1.GrafanaResourceDefinition) {
	for _, v := range obj.Spec.Versions {
		c.queue.Add(schema.GroupVersion{Group: obj.Spec.Group, Version: v.Name})
	}
}

func (c *DiscoveryController) addGrafanaResourceDefinition(obj interface{}) {
	castObj := obj.(*kindsv1.GrafanaResourceDefinition)
	klog.V(4).Infof("Adding grafanaresourcedefinition %s", castObj.Name)
	c.enqueue(castObj)
}

func (c *DiscoveryController) updateGrafanaResourceDefinition(oldObj, newObj interface{}) {
	castNewObj := newObj.(*kindsv1.GrafanaResourceDefinition)
	castOldObj := oldObj.(*kindsv1.GrafanaResourceDefinition)
	klog.V(4).Infof("Updating grafanaresourcedefinition %s", castOldObj.Name)
	// Enqueue both old and new object to make sure we remove and add appropriate Versions.
	// The working queue will resolve any duplicates and only changes will stay in the queue.
	c.enqueue(castNewObj)
	c.enqueue(castOldObj)
}

func (c *DiscoveryController) deleteGrafanaResourceDefinition(obj interface{}) {
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
	klog.V(4).Infof("Deleting grafanaresourcedefinition %q", castObj.Name)
	c.enqueue(castObj)
}
