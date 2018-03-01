package fake

import (
	v1alpha1 "github.com/barpilot/kinky/client/k8s/clientset/versioned/typed/kinky/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeKinkyV1alpha1 struct {
	*testing.Fake
}

func (c *FakeKinkyV1alpha1) Kinkies(namespace string) v1alpha1.KinkyInterface {
	return &FakeKinkies{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeKinkyV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
