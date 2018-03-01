package v1alpha1

import (
	v1alpha1 "github.com/barpilot/kinky/apis/kinky/v1alpha1"
	scheme "github.com/barpilot/kinky/client/k8s/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// KinkiesGetter has a method to return a KinkyInterface.
// A group's client should implement this interface.
type KinkiesGetter interface {
	Kinkies(namespace string) KinkyInterface
}

// KinkyInterface has methods to work with Kinky resources.
type KinkyInterface interface {
	Create(*v1alpha1.Kinky) (*v1alpha1.Kinky, error)
	Update(*v1alpha1.Kinky) (*v1alpha1.Kinky, error)
	UpdateStatus(*v1alpha1.Kinky) (*v1alpha1.Kinky, error)
	Delete(name string, options *v1.DeleteOptions) error
	DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error
	Get(name string, options v1.GetOptions) (*v1alpha1.Kinky, error)
	List(opts v1.ListOptions) (*v1alpha1.KinkyList, error)
	Watch(opts v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Kinky, err error)
	KinkyExpansion
}

// kinkies implements KinkyInterface
type kinkies struct {
	client rest.Interface
	ns     string
}

// newKinkies returns a Kinkies
func newKinkies(c *KinkyV1alpha1Client, namespace string) *kinkies {
	return &kinkies{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the kinky, and returns the corresponding kinky object, and an error if there is any.
func (c *kinkies) Get(name string, options v1.GetOptions) (result *v1alpha1.Kinky, err error) {
	result = &v1alpha1.Kinky{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("kinkies").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of Kinkies that match those selectors.
func (c *kinkies) List(opts v1.ListOptions) (result *v1alpha1.KinkyList, err error) {
	result = &v1alpha1.KinkyList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("kinkies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested kinkies.
func (c *kinkies) Watch(opts v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("kinkies").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a kinky and creates it.  Returns the server's representation of the kinky, and an error, if there is any.
func (c *kinkies) Create(kinky *v1alpha1.Kinky) (result *v1alpha1.Kinky, err error) {
	result = &v1alpha1.Kinky{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("kinkies").
		Body(kinky).
		Do().
		Into(result)
	return
}

// Update takes the representation of a kinky and updates it. Returns the server's representation of the kinky, and an error, if there is any.
func (c *kinkies) Update(kinky *v1alpha1.Kinky) (result *v1alpha1.Kinky, err error) {
	result = &v1alpha1.Kinky{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("kinkies").
		Name(kinky.Name).
		Body(kinky).
		Do().
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *kinkies) UpdateStatus(kinky *v1alpha1.Kinky) (result *v1alpha1.Kinky, err error) {
	result = &v1alpha1.Kinky{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("kinkies").
		Name(kinky.Name).
		SubResource("status").
		Body(kinky).
		Do().
		Into(result)
	return
}

// Delete takes name of the kinky and deletes it. Returns an error if one occurs.
func (c *kinkies) Delete(name string, options *v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("kinkies").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *kinkies) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("kinkies").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched kinky.
func (c *kinkies) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Kinky, err error) {
	result = &v1alpha1.Kinky{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("kinkies").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
