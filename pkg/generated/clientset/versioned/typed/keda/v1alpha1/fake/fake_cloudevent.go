/*
Copyright 2023 The KEDA Authors

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/kedacore/keda/v2/apis/keda/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeCloudEvents implements CloudEventInterface
type FakeCloudEvents struct {
	Fake *FakeKedaV1alpha1
	ns   string
}

var cloudeventsResource = v1alpha1.SchemeGroupVersion.WithResource("cloudevents")

var cloudeventsKind = v1alpha1.SchemeGroupVersion.WithKind("CloudEvent")

// Get takes name of the cloudEvent, and returns the corresponding cloudEvent object, and an error if there is any.
func (c *FakeCloudEvents) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.CloudEvent, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(cloudeventsResource, c.ns, name), &v1alpha1.CloudEvent{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.CloudEvent), err
}

// List takes label and field selectors, and returns the list of CloudEvents that match those selectors.
func (c *FakeCloudEvents) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.CloudEventList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(cloudeventsResource, cloudeventsKind, c.ns, opts), &v1alpha1.CloudEventList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.CloudEventList{ListMeta: obj.(*v1alpha1.CloudEventList).ListMeta}
	for _, item := range obj.(*v1alpha1.CloudEventList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested cloudEvents.
func (c *FakeCloudEvents) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(cloudeventsResource, c.ns, opts))

}

// Create takes the representation of a cloudEvent and creates it.  Returns the server's representation of the cloudEvent, and an error, if there is any.
func (c *FakeCloudEvents) Create(ctx context.Context, cloudEvent *v1alpha1.CloudEvent, opts v1.CreateOptions) (result *v1alpha1.CloudEvent, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(cloudeventsResource, c.ns, cloudEvent), &v1alpha1.CloudEvent{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.CloudEvent), err
}

// Update takes the representation of a cloudEvent and updates it. Returns the server's representation of the cloudEvent, and an error, if there is any.
func (c *FakeCloudEvents) Update(ctx context.Context, cloudEvent *v1alpha1.CloudEvent, opts v1.UpdateOptions) (result *v1alpha1.CloudEvent, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(cloudeventsResource, c.ns, cloudEvent), &v1alpha1.CloudEvent{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.CloudEvent), err
}

// Delete takes name of the cloudEvent and deletes it. Returns an error if one occurs.
func (c *FakeCloudEvents) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteActionWithOptions(cloudeventsResource, c.ns, name, opts), &v1alpha1.CloudEvent{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeCloudEvents) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(cloudeventsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.CloudEventList{})
	return err
}

// Patch applies the patch and returns the patched cloudEvent.
func (c *FakeCloudEvents) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.CloudEvent, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(cloudeventsResource, c.ns, name, pt, data, subresources...), &v1alpha1.CloudEvent{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.CloudEvent), err
}
