// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1alpha1 "github.com/ctron/hono-qdrouter-proxy/pkg/client/clientset/versioned/typed/iotproject/v1alpha1"
	rest "k8s.io/client-go/rest"
	testing "k8s.io/client-go/testing"
)

type FakeHonoV1alpha1 struct {
	*testing.Fake
}

func (c *FakeHonoV1alpha1) IoTProjects(namespace string) v1alpha1.IoTProjectInterface {
	return &FakeIoTProjects{c, namespace}
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *FakeHonoV1alpha1) RESTClient() rest.Interface {
	var ret *rest.RESTClient
	return ret
}
