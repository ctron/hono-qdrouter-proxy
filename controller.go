/*
Copyright 2017 The Kubernetes Authors.

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

package main

import (
    "fmt"
    "github.com/ctron/hono-qdrouter-proxy/pkg/apis/iotproject/v1alpha1"
    "reflect"
    "strconv"
    "time"

    "k8s.io/apimachinery/pkg/api/errors"
    "k8s.io/apimachinery/pkg/util/runtime"
    utilruntime "k8s.io/apimachinery/pkg/util/runtime"
    "k8s.io/apimachinery/pkg/util/wait"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/kubernetes/scheme"
    "k8s.io/client-go/tools/cache"
    "k8s.io/client-go/util/workqueue"
    "k8s.io/klog"

    clientset "github.com/ctron/hono-qdrouter-proxy/pkg/client/clientset/versioned"
    iotscheme "github.com/ctron/hono-qdrouter-proxy/pkg/client/clientset/versioned/scheme"
    informers "github.com/ctron/hono-qdrouter-proxy/pkg/client/informers/externalversions/iotproject/v1alpha1"
    listers "github.com/ctron/hono-qdrouter-proxy/pkg/client/listers/iotproject/v1alpha1"

    "github.com/ctron/hono-qdrouter-proxy/pkg/qdr"
)

// Controller is the controller implementation for Foo resources
type Controller struct {
    // kubeclientset is a standard kubernetes clientset
    kubeclientset kubernetes.Interface
    // sampleclientset is a clientset for our own API group
    iotclientset clientset.Interface

    projectLister  listers.IoTProjectLister
    projectsSynced cache.InformerSynced

    // workqueue is a rate limited work queue. This is used to queue work to be
    // processed instead of performing it as soon as a change happens. This
    // means we can ensure we only process a fixed amount of resources at a
    // time, and makes it easy to ensure we are never processing the same item
    // simultaneously in two different workers.
    workqueue workqueue.RateLimitingInterface

    manage *qdr.Manage
}

func NewController(
    kubeclientset kubernetes.Interface,
    iotclientset clientset.Interface,
    projectInformer informers.IoTProjectInformer) *Controller {

    utilruntime.Must(iotscheme.AddToScheme(scheme.Scheme))

    controller := &Controller{
        kubeclientset:  kubeclientset,
        iotclientset:   iotclientset,
        projectLister:  projectInformer.Lister(),
        projectsSynced: projectInformer.Informer().HasSynced,
        workqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "IoTProjects"),
        manage:         qdr.NewManage(),
    }

    klog.Info("Setting up event handlers")

    // listen for events on the project resource
    projectInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            klog.Infof("Add: %s", obj)
            controller.enqueueProject(obj)
        },
        UpdateFunc: func(old, new interface{}) {
            klog.Infof("Update - old: %s, new: %s", old, new)
            controller.enqueueProject(new)
        },
        DeleteFunc: func(obj interface{}) {
            klog.Infof("Deleted: %s", obj)
            controller.enqueueProject(obj)
        },
    })

    return controller
}

func (c *Controller) enqueueProject(obj interface{}) {
    var key string
    var err error
    if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
        runtime.HandleError(err)
        return
    }
    c.workqueue.AddRateLimited(key)
}

// Run main controller
//
// This will run until the `stopCh` is closed, which will then shutdown the
// workqueue, wait for workers to complete, and then return.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
    defer runtime.HandleCrash()
    defer c.workqueue.ShutDown()

    klog.Info("Starting IoTProjects controller")

    // prepare the caches

    klog.Info("Waiting for informer caches to sync")
    if ok := cache.WaitForCacheSync(stopCh, c.projectsSynced); !ok {
        return fmt.Errorf("failed to wait for caches to sync")
    }

    // start the workers

    klog.Infof("Starting %s worker(s)", threadiness)
    for i := 0; i < threadiness; i++ {
        go wait.Until(c.runWorker, time.Second, stopCh)
    }

    // wait for shutdown

    klog.Info("Started workers")
    <-stopCh
    klog.Info("Shutting down workers")

    // return

    return nil
}

// fetch any process work
func (c *Controller) runWorker() {
    for c.processNextWorkItem() {
    }
}

// fetch and process the next work item
func (c *Controller) processNextWorkItem() bool {
    obj, shutdown := c.workqueue.Get()

    if shutdown {
        return false
    }

    // scope next section in order to use "defer", poor man's try-finally
    err := func(obj interface{}) error {

        // by the end of the function, we are done with this item
        // either we Forget() about it, or re-queue it
        defer c.workqueue.Done(obj)

        // try-cast to string
        key, ok := obj.(string)

        // the work queue should only contain strings
        if !ok {
            // if it doesn't, drop the item
            c.workqueue.Forget(obj)
            runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
            return nil
        }

        // try-sync change event, on error -> re-queue

        if err := c.syncHandler(key); err != nil {
            c.workqueue.AddRateLimited(key)
            return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
        }

        // handled successfully, drop from work queue

        c.workqueue.Forget(obj)
        klog.Infof("Successfully synced '%s'", key)

        return nil
    }(obj)

    // if an error occurred ...
    if err != nil {
        // ... handle error ...
        runtime.HandleError(err)
        // ... and continue processing
        return true
    }

    // return, indicating that we want more
    return true
}

// Synchronize the requested state with the actual state
func (c *Controller) syncHandler(key string) error {

    // parse into namespace + name
    namespace, name, err := cache.SplitMetaNamespaceKey(key)
    if err != nil {
        runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
        return nil
    }

    // read requested state
    project, err := c.projectLister.IoTProjects(namespace).Get(name)
    if err != nil {

        // something went wrong

        if errors.IsNotFound(err) {

            // we didn't find the object

            klog.Info("Item got deleted. Deleting configuration.")
            err = c.deleteLinkRoute(namespace, name)

            // if something went wrong deleting, then returning
            // and error will re-queue the item
            return err
        }

        return err
    }

    // sync add or update
    _, err = c.syncProject(project)

    // something went wrong syncing the project
    // we will re-queue this by returning the error state
    if err != nil {
        return err
    }

    return nil
}

func (c *Controller) syncResource(currentPointer interface{}, resource qdr.RouterResource, creator func() map[string]string) (bool, error) {

    found, err := c.manage.ReadAsObject(resource, currentPointer)
    if err != nil {
        return false, err
    }

    klog.Infof("Found: %s", found)
    klog.Infof("Current: %s", currentPointer)
    klog.Infof("Request: %s", &resource)

    if found && reflect.DeepEqual(currentPointer, &resource) {
        return false, nil
    }

    if found {
        c.manage.Delete(resource)
    }

    _, err = c.manage.Create(resource, creator())

    return true, err

}

func (c *Controller) syncLinkRoute(route qdr.LinkRoute) (bool, error) {

    return c.syncResource(&qdr.LinkRoute{}, route, func() map[string]string {
        return map[string]string{
            "direction":  route.Direction,
            "pattern":    route.Pattern,
            "connection": route.Connection,
        }
    })

}

func (c *Controller) syncConnector(connector qdr.Connector) (bool, error) {

    return c.syncResource(&qdr.Connector{}, connector, func() map[string]string {
        return map[string]string{
            "host":         connector.Host,
            "port":         connector.Port,
            "role":         connector.Role,
            "saslUsername": connector.SASLUsername,
            "saslPassword": connector.SASLPassword,
        }
    })

}

/*
func (c *Controller) syncConnector(connector qdr.Connector) (bool, error) {

    current, err := c.manage.ReadAsObject(qdr.TYPE_NAME_CONNECTOR, connector.Name, new(qdr.Connector))
    if err != nil {
        return false, err
    }

    if current != nil && reflect.DeepEqual(current.(*qdr.Connector), &connector) {
        return false, nil
    }

    if current != nil {
        c.manage.Delete(qdr.TYPE_NAME_CONNECTOR, connector.Name)
    }

    _, err = c.manage.Create(connector.Type, connector.Name, map[string]string{
        "host":         connector.Host,
        "port":         connector.Port,
        "role":         connector.Role,
        "saslUsername": connector.SASLUsername,
        "saslPassword": connector.SASLPassword,
    })

    return true, err
}
*/
func (c *Controller) syncProject(project *v1alpha1.IoTProject) (bool, error) {

    tenantName := project.Namespace + "." + project.Name
    baseName := tenantName
    addressTenantName := tenantName

    connectorName := "connector-" + baseName

    klog.Infof("Create link routes - tenant: %s", tenantName)

    var change bool = false

    res, err := c.syncConnector(qdr.Connector{
        NamedResource: qdr.NamedResource{Name: connectorName},
        Host:          project.Spec.Host,
        Port:          strconv.Itoa(int(project.Spec.Port)),
        Role:          "route-container",
        SASLUsername:  project.Spec.Username,
        SASLPassword:  project.Spec.Password,
    })
    if err != nil {
        return false, err
    }
    change = change || res

    res, err = c.syncLinkRoute(qdr.LinkRoute{
        NamedResource: qdr.NamedResource{Name: "linkRoute/t/" + baseName},
        Direction:     "in",
        Pattern:       "telemetry/" + addressTenantName + "/#",
        Connection:    connectorName,
    })
    if err != nil {
        return false, err
    }
    change = change || res

    res, err = c.syncLinkRoute(qdr.LinkRoute{
        NamedResource: qdr.NamedResource{Name: "linkRoute/e/" + baseName},
        Direction:     "in",
        Pattern:       "event/" + addressTenantName + "/#",
        Connection:    connectorName,
    })
    if err != nil {
        return false, err
    }
    change = change || res

    res, err = c.syncLinkRoute(qdr.LinkRoute{
        NamedResource: qdr.NamedResource{Name: "linkRoute/c_i/" + baseName},
        Direction:     "in",
        Pattern:       "control/" + addressTenantName + "/#",
        Connection:    connectorName,
    })
    if err != nil {
        return false, err
    }
    change = change || res

    res, err = c.syncLinkRoute(qdr.LinkRoute{
        NamedResource: qdr.NamedResource{Name: "linkRoute/c_o/" + baseName},
        Direction:     "out",
        Pattern:       "control/" + addressTenantName + "/#",
        Connection:    connectorName,
    })
    if err != nil {
        return false, err
    }
    change = change || res

    return change, nil
}

func (c *Controller) deleteLinkRoute(namespace string, name string) error {

    tenantName := namespace + "." + name
    baseName := tenantName
    connectorName := "connector-" + baseName

    klog.Infof("Delete link routes - tenant: %s", tenantName)

    if err := c.manage.Delete(qdr.TypeAndName(qdr.TypeNameLinkRoute, "linkRoute/t/"+baseName)); err != nil {
        return err
    }
    if err := c.manage.Delete(qdr.TypeAndName(qdr.TypeNameLinkRoute, "linkRoute/e/"+baseName)); err != nil {
        return err
    }
    if err := c.manage.Delete(qdr.TypeAndName(qdr.TypeNameLinkRoute, "linkRoute/c_i/"+baseName)); err != nil {
        return err
    }
    if err := c.manage.Delete(qdr.TypeAndName(qdr.TypeNameLinkRoute, "linkRoute/c_o/"+baseName)); err != nil {
        return err
    }
    if err := c.manage.Delete(qdr.TypeAndName(qdr.TypeNameLinkRoute, connectorName)); err != nil {
        return err
    }

    return nil
}
