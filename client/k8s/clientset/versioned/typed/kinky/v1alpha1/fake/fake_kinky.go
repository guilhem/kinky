package fake

import (
	v1alpha1 "github.com/barpilot/kinky/apis/kinky/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeKinkies implements KinkyInterface
type FakeKinkies struct {
	Fake *FakeKinkyV1alpha1
	ns   string
}

var kinkiesResource = schema.GroupVersionResource{Group: "kinky", Version: "v1alpha1", Resource: "kinkies"}

var kinkiesKind = schema.GroupVersionKind{Group: "kinky", Version: "v1alpha1", Kind: "Kinky"}

// Get takes name of the kinky, and returns the corresponding kinky object, and an error if there is any.
func (c *FakeKinkies) Get(name string, options v1.GetOptions) (result *v1alpha1.Kinky, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(kinkiesResource, c.ns, name), &v1alpha1.Kinky{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Kinky), err
}

// List takes label and field selectors, and returns the list of Kinkies that match those selectors.
func (c *FakeKinkies) List(opts v1.ListOptions) (result *v1alpha1.KinkyList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(kinkiesResource, kinkiesKind, c.ns, opts), &v1alpha1.KinkyList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.KinkyList{}
	for _, item := range obj.(*v1alpha1.KinkyList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested kinkies.
func (c *FakeKinkies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(kinkiesResource, c.ns, opts))

}

// Create takes the representation of a kinky and creates it.  Returns the server's representation of the kinky, and an error, if there is any.
func (c *FakeKinkies) Create(kinky *v1alpha1.Kinky) (result *v1alpha1.Kinky, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(kinkiesResource, c.ns, kinky), &v1alpha1.Kinky{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Kinky), err
}

// Update takes the representation of a kinky and updates it. Returns the server's representation of the kinky, and an error, if there is any.
func (c *FakeKinkies) Update(kinky *v1alpha1.Kinky) (result *v1alpha1.Kinky, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(kinkiesResource, c.ns, kinky), &v1alpha1.Kinky{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Kinky), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeKinkies) UpdateStatus(kinky *v1alpha1.Kinky) (*v1alpha1.Kinky, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(kinkiesResource, "status", c.ns, kinky), &v1alpha1.Kinky{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Kinky), err
}

// Delete takes name of the kinky and deletes it. Returns an error if one occurs.
func (c *FakeKinkies) Delete(name string, options *v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(kinkiesResource, c.ns, name), &v1alpha1.Kinky{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeKinkies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(kinkiesResource, c.ns, listOptions)

	_, err := c.Fake.Invokes(action, &v1alpha1.KinkyList{})
	return err
}

// Patch applies the patch and returns the patched kinky.
func (c *FakeKinkies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Kinky, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(kinkiesResource, c.ns, name, data, subresources...), &v1alpha1.Kinky{})

	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.Kinky), err
}
