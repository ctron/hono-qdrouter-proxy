/*
 * Copyright 2018, EnMasse authors.
 * License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).
 */

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	clientset "github.com/ctron/hono-qdrouter-proxy/pkg/client/clientset/versioned"
	honov1alpha1 "github.com/ctron/hono-qdrouter-proxy/pkg/client/clientset/versioned/typed/iotproject/v1alpha1"
	fakehonov1alpha1 "github.com/ctron/hono-qdrouter-proxy/pkg/client/clientset/versioned/typed/iotproject/v1alpha1/fake"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/testing"
)

// NewSimpleClientset returns a clientset that will respond with the provided objects.
// It's backed by a very simple object tracker that processes creates, updates and deletions as-is,
// without applying any validations and/or defaults. It shouldn't be considered a replacement
// for a real clientset and is mostly useful in simple unit tests.
func NewSimpleClientset(objects ...runtime.Object) *Clientset {
	o := testing.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objects {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	cs := &Clientset{}
	cs.discovery = &fakediscovery.FakeDiscovery{Fake: &cs.Fake}
	cs.AddReactor("*", "*", testing.ObjectReaction(o))
	cs.AddWatchReactor("*", func(action testing.Action) (handled bool, ret watch.Interface, err error) {
		gvr := action.GetResource()
		ns := action.GetNamespace()
		watch, err := o.Watch(gvr, ns)
		if err != nil {
			return false, nil, err
		}
		return true, watch, nil
	})

	return cs
}

// Clientset implements clientset.Interface. Meant to be embedded into a
// struct to get a default implementation. This makes faking out just the method
// you want to test easier.
type Clientset struct {
	testing.Fake
	discovery *fakediscovery.FakeDiscovery
}

func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	return c.discovery
}

var _ clientset.Interface = &Clientset{}

// HonoV1alpha1 retrieves the HonoV1alpha1Client
func (c *Clientset) HonoV1alpha1() honov1alpha1.HonoV1alpha1Interface {
	return &fakehonov1alpha1.FakeHonoV1alpha1{Fake: &c.Fake}
}

// Hono retrieves the HonoV1alpha1Client
func (c *Clientset) Hono() honov1alpha1.HonoV1alpha1Interface {
	return &fakehonov1alpha1.FakeHonoV1alpha1{Fake: &c.Fake}
}
