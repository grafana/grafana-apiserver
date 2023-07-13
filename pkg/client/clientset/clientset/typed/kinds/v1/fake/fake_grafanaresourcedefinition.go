// SPDX-License-Identifier: AGPL-3.0-only

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"
	json "encoding/json"
	"fmt"

	v1 "github.com/grafana/grafana-apiserver/pkg/apis/kinds/v1"
	kindsv1 "github.com/grafana/grafana-apiserver/pkg/client/applyconfiguration/kinds/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeGrafanaResourceDefinitions implements GrafanaResourceDefinitionInterface
type FakeGrafanaResourceDefinitions struct {
	Fake *FakeKindsV1
}

var grafanaresourcedefinitionsResource = v1.SchemeGroupVersion.WithResource("grafanaresourcedefinitions")

var grafanaresourcedefinitionsKind = v1.SchemeGroupVersion.WithKind("GrafanaResourceDefinition")

// Get takes name of the grafanaResourceDefinition, and returns the corresponding grafanaResourceDefinition object, and an error if there is any.
func (c *FakeGrafanaResourceDefinitions) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.GrafanaResourceDefinition, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(grafanaresourcedefinitionsResource, name), &v1.GrafanaResourceDefinition{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.GrafanaResourceDefinition), err
}

// List takes label and field selectors, and returns the list of GrafanaResourceDefinitions that match those selectors.
func (c *FakeGrafanaResourceDefinitions) List(ctx context.Context, opts metav1.ListOptions) (result *v1.GrafanaResourceDefinitionList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(grafanaresourcedefinitionsResource, grafanaresourcedefinitionsKind, opts), &v1.GrafanaResourceDefinitionList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1.GrafanaResourceDefinitionList{ListMeta: obj.(*v1.GrafanaResourceDefinitionList).ListMeta}
	for _, item := range obj.(*v1.GrafanaResourceDefinitionList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested grafanaResourceDefinitions.
func (c *FakeGrafanaResourceDefinitions) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(grafanaresourcedefinitionsResource, opts))
}

// Create takes the representation of a grafanaResourceDefinition and creates it.  Returns the server's representation of the grafanaResourceDefinition, and an error, if there is any.
func (c *FakeGrafanaResourceDefinitions) Create(ctx context.Context, grafanaResourceDefinition *v1.GrafanaResourceDefinition, opts metav1.CreateOptions) (result *v1.GrafanaResourceDefinition, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(grafanaresourcedefinitionsResource, grafanaResourceDefinition), &v1.GrafanaResourceDefinition{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.GrafanaResourceDefinition), err
}

// Update takes the representation of a grafanaResourceDefinition and updates it. Returns the server's representation of the grafanaResourceDefinition, and an error, if there is any.
func (c *FakeGrafanaResourceDefinitions) Update(ctx context.Context, grafanaResourceDefinition *v1.GrafanaResourceDefinition, opts metav1.UpdateOptions) (result *v1.GrafanaResourceDefinition, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(grafanaresourcedefinitionsResource, grafanaResourceDefinition), &v1.GrafanaResourceDefinition{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.GrafanaResourceDefinition), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeGrafanaResourceDefinitions) UpdateStatus(ctx context.Context, grafanaResourceDefinition *v1.GrafanaResourceDefinition, opts metav1.UpdateOptions) (*v1.GrafanaResourceDefinition, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(grafanaresourcedefinitionsResource, "status", grafanaResourceDefinition), &v1.GrafanaResourceDefinition{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.GrafanaResourceDefinition), err
}

// Delete takes name of the grafanaResourceDefinition and deletes it. Returns an error if one occurs.
func (c *FakeGrafanaResourceDefinitions) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(grafanaresourcedefinitionsResource, name, opts), &v1.GrafanaResourceDefinition{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGrafanaResourceDefinitions) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(grafanaresourcedefinitionsResource, listOpts)

	_, err := c.Fake.Invokes(action, &v1.GrafanaResourceDefinitionList{})
	return err
}

// Patch applies the patch and returns the patched grafanaResourceDefinition.
func (c *FakeGrafanaResourceDefinitions) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.GrafanaResourceDefinition, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(grafanaresourcedefinitionsResource, name, pt, data, subresources...), &v1.GrafanaResourceDefinition{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.GrafanaResourceDefinition), err
}

// Apply takes the given apply declarative configuration, applies it and returns the applied grafanaResourceDefinition.
func (c *FakeGrafanaResourceDefinitions) Apply(ctx context.Context, grafanaResourceDefinition *kindsv1.GrafanaResourceDefinitionApplyConfiguration, opts metav1.ApplyOptions) (result *v1.GrafanaResourceDefinition, err error) {
	if grafanaResourceDefinition == nil {
		return nil, fmt.Errorf("grafanaResourceDefinition provided to Apply must not be nil")
	}
	data, err := json.Marshal(grafanaResourceDefinition)
	if err != nil {
		return nil, err
	}
	name := grafanaResourceDefinition.Name
	if name == nil {
		return nil, fmt.Errorf("grafanaResourceDefinition.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(grafanaresourcedefinitionsResource, *name, types.ApplyPatchType, data), &v1.GrafanaResourceDefinition{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.GrafanaResourceDefinition), err
}

// ApplyStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating ApplyStatus().
func (c *FakeGrafanaResourceDefinitions) ApplyStatus(ctx context.Context, grafanaResourceDefinition *kindsv1.GrafanaResourceDefinitionApplyConfiguration, opts metav1.ApplyOptions) (result *v1.GrafanaResourceDefinition, err error) {
	if grafanaResourceDefinition == nil {
		return nil, fmt.Errorf("grafanaResourceDefinition provided to Apply must not be nil")
	}
	data, err := json.Marshal(grafanaResourceDefinition)
	if err != nil {
		return nil, err
	}
	name := grafanaResourceDefinition.Name
	if name == nil {
		return nil, fmt.Errorf("grafanaResourceDefinition.Name must be provided to Apply")
	}
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(grafanaresourcedefinitionsResource, *name, types.ApplyPatchType, data, "status"), &v1.GrafanaResourceDefinition{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1.GrafanaResourceDefinition), err
}