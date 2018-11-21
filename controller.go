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
    "os"
    "os/exec"
    "strconv"
    "time"

    corev1 "k8s.io/api/core/v1"
    "k8s.io/apimachinery/pkg/api/errors"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/util/runtime"
    utilruntime "k8s.io/apimachinery/pkg/util/runtime"
    "k8s.io/apimachinery/pkg/util/wait"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/kubernetes/scheme"
    typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
    "k8s.io/client-go/tools/cache"
    "k8s.io/client-go/tools/record"
    "k8s.io/client-go/util/workqueue"
    "k8s.io/klog"

    clientset "github.com/ctron/hono-qdrouter-proxy/pkg/client/clientset/versioned"
    iotscheme "github.com/ctron/hono-qdrouter-proxy/pkg/client/clientset/versioned/scheme"
    informers "github.com/ctron/hono-qdrouter-proxy/pkg/client/informers/externalversions/iotproject/v1alpha1"
    listers "github.com/ctron/hono-qdrouter-proxy/pkg/client/listers/iotproject/v1alpha1"
)

const controllerAgentName = "sample-controller"

const (
    // SuccessSynced is used as part of the Event 'reason' when a Foo is synced
    SuccessSynced = "Synced"
    // ErrResourceExists is used as part of the Event 'reason' when a Foo fails
    // to sync due to a Deployment of the same name already existing.
    ErrResourceExists = "ErrResourceExists"

    // MessageResourceExists is the message used for Events when a resource
    // fails to sync due to a Deployment already existing
    MessageResourceExists = "Resource %q already exists and is not managed by Project"
    // MessageResourceSynced is the message used for an Event fired when a Foo
    // is synced successfully
    MessageResourceSynced = "Project synced successfully"
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
    // recorder is an event recorder for recording Event resources to the
    // Kubernetes API.
    recorder record.EventRecorder
}

// NewController returns a new sample controller
func NewController(
    kubeclientset kubernetes.Interface,
    iotclientset clientset.Interface,
    projectInformer informers.IoTProjectInformer) *Controller {

    // Create event broadcaster
    // Add sample-controller types to the default Kubernetes Scheme so Events can be
    // logged for sample-controller types.
    utilruntime.Must(iotscheme.AddToScheme(scheme.Scheme))
    klog.V(4).Info("Creating event broadcaster")
    eventBroadcaster := record.NewBroadcaster()
    eventBroadcaster.StartLogging(klog.Infof)
    eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
    recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

    controller := &Controller{
        kubeclientset:  kubeclientset,
        iotclientset:   iotclientset,
        projectLister:  projectInformer.Lister(),
        projectsSynced: projectInformer.Informer().HasSynced,
        workqueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "IoTProjects"),
        recorder:       recorder,
    }

    klog.Info("Setting up event handlers")
    // Set up an event handler for when Foo resources change
    projectInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
        AddFunc: func(obj interface{}) {
            klog.Infof("Add: %s", obj)
            controller.enqueueFoo(obj)
        },
        UpdateFunc: func(old, new interface{}) {
            klog.Infof("Update - old: %s, new: %s", old, new)
            controller.enqueueFoo(new)
        },
        DeleteFunc: func(obj interface{}) {
            klog.Infof("Delete: %s", obj)
        },
    })

    return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
    defer runtime.HandleCrash()
    defer c.workqueue.ShutDown()

    // Start the informer factories to begin populating the informer caches
    klog.Info("Starting IoTProjects controller")

    // Wait for the caches to be synced before starting workers
    klog.Info("Waiting for informer caches to sync")
    if ok := cache.WaitForCacheSync(stopCh, c.projectsSynced); !ok {
        return fmt.Errorf("failed to wait for caches to sync")
    }

    klog.Info("Starting workers")
    // Launch two workers to process Foo resources
    for i := 0; i < threadiness; i++ {
        go wait.Until(c.runWorker, time.Second, stopCh)
    }

    klog.Info("Started workers")
    <-stopCh
    klog.Info("Shutting down workers")

    return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *Controller) runWorker() {
    for c.processNextWorkItem() {
    }
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
    obj, shutdown := c.workqueue.Get()

    if shutdown {
        return false
    }

    // We wrap this block in a func so we can defer c.workqueue.Done.
    err := func(obj interface{}) error {
        // We call Done here so the workqueue knows we have finished
        // processing this item. We also must remember to call Forget if we
        // do not want this work item being re-queued. For example, we do
        // not call Forget if a transient error occurs, instead the item is
        // put back on the workqueue and attempted again after a back-off
        // period.
        defer c.workqueue.Done(obj)
        var key string
        var ok bool
        // We expect strings to come off the workqueue. These are of the
        // form namespace/name. We do this as the delayed nature of the
        // workqueue means the items in the informer cache may actually be
        // more up to date that when the item was initially put onto the
        // workqueue.
        if key, ok = obj.(string); !ok {
            // As the item in the workqueue is actually invalid, we call
            // Forget here else we'd go into a loop of attempting to
            // process a work item that is invalid.
            c.workqueue.Forget(obj)
            runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
            return nil
        }
        // Run the syncHandler, passing it the namespace/name string of the
        // Foo resource to be synced.
        if err := c.syncHandler(key); err != nil {
            // Put the item back on the workqueue to handle any transient errors.
            c.workqueue.AddRateLimited(key)
            return fmt.Errorf("error syncing '%s': %s, requeuing", key, err.Error())
        }
        // Finally, if no error occurs we Forget this item so it does not
        // get queued again until another change happens.
        c.workqueue.Forget(obj)
        klog.Infof("Successfully synced '%s'", key)
        return nil
    }(obj)

    if err != nil {
        runtime.HandleError(err)
        return true
    }

    return true
}

// syncHandler compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the Foo resource
// with the current status of the resource.
func (c *Controller) syncHandler(key string) error {
    // Convert the namespace/name string into a distinct namespace and name
    namespace, name, err := cache.SplitMetaNamespaceKey(key)
    if err != nil {
        runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
        return nil
    }

    // Get the Foo resource with this namespace/name
    project, err := c.projectLister.IoTProjects(namespace).Get(name)
    if err != nil {
        // The Foo resource may no longer exist, in which case we stop
        // processing.
        if errors.IsNotFound(err) {

            c.deleteLinkRoute(project)

            runtime.HandleError(fmt.Errorf("project '%s' in work queue no longer exists", key))
            return nil
        }

        return err
    }

    // manage qdrouter
    c.deleteLinkRoute(project)
    c.createLinkRoute(project)

    // If an error occurs during Update, we'll requeue the item so we can
    // attempt processing again later. THis could have been caused by a
    // temporary network failure, or any other transient reason.
    if err != nil {
        return err
    }

    c.recorder.Event(project, corev1.EventTypeNormal, SuccessSynced, MessageResourceSynced)
    return nil
}

func (c *Controller) manage(operation string, attributes map[string]string) error {
    args := []string{}

    args = append(args, "-b", "amqp://localhost:5762")
    args = append(args, operation)

    for k, v := range attributes {
        args = append(args, k+"="+v)
    }

    klog.Infof("Call with args: %s", args)

    cmd := exec.Command("/usr/bin/qdmanage", args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    err := cmd.Run()

    if err != nil  {
        klog.Infof("Result: %s", err.Error())
    }

    return err
}

func (c *Controller) createLinkRoute(project *v1alpha1.IoTProject) {

    tenantName := project.Namespace + "." + project.Name
    baseName := tenantName

    klog.Infof("Create link routes - tenant: %s", tenantName)

    c.manage("create", map[string]string{
        "type":         "connector",
        "name":         "connector/" + baseName,
        "host":         project.Spec.Host,
        "port":         strconv.Itoa(int(*project.Spec.Port)),
        "role":         "route-container",
        "saslUsername": project.Spec.Username,
        "saslPassword": project.Spec.Password,
    })

    c.manage("create", map[string]string{
        "type":       "linkRoute",
        "name":       "linkRoute/t/" + baseName,
        "direction":  "in",
        "pattern":    "telemetry/" + tenantName + "/#",
        "connection": "connector/" + baseName,
    })

    c.manage("create", map[string]string{
        "type":       "linkRoute",
        "name":       "linkRoute/e/" + baseName,
        "direction":  "in",
        "pattern":    "event/" + tenantName + "/#",
        "connection": "connector/" + baseName,
    })

    c.manage("create", map[string]string{
        "type":       "linkRoute",
        "name":       "linkRoute/c/" + baseName,
        "direction":  "in",
        "pattern":    "control/" + tenantName + "/#",
        "connection": "connector/" + baseName,
    })
}

func (c *Controller) deleteLinkRoute(project *v1alpha1.IoTProject) {

    tenantName := project.Namespace + "." + project.Name
    baseName := tenantName

    klog.Infof("Delete link routes - tenant: %s", tenantName)

    c.manage("delete", map[string]string{
        "--name": "connector/" + baseName,
        "--type": "connector",
    })
    c.manage("delete", map[string]string{
        "--name": "linkRoute/t/" + baseName,
        "--type": "linkRoute",
    })
    c.manage("delete", map[string]string{
        "--name": "linkRoute/e/" + baseName,
        "--type": "linkRoute",
    })
    c.manage("delete", map[string]string{
        "--name": "linkRoute/c/" + baseName,
        "--type": "linkRoute",
    })
}

// enqueueFoo takes a Foo resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than Foo.
func (c *Controller) enqueueFoo(obj interface{}) {
    var key string
    var err error
    if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
        runtime.HandleError(err)
        return
    }
    c.workqueue.AddRateLimited(key)
}

// handleObject will take any resource implementing metav1.Object and attempt
// to find the Foo resource that 'owns' it. It does this by looking at the
// objects metadata.ownerReferences field for an appropriate OwnerReference.
// It then enqueues that Foo resource to be processed. If the object does not
// have an appropriate OwnerReference, it will simply be skipped.
func (c *Controller) handleObject(obj interface{}) {
    var object metav1.Object
    var ok bool
    if object, ok = obj.(metav1.Object); !ok {
        tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
        if !ok {
            runtime.HandleError(fmt.Errorf("error decoding object, invalid type"))
            return
        }
        object, ok = tombstone.Obj.(metav1.Object)
        if !ok {
            runtime.HandleError(fmt.Errorf("error decoding object tombstone, invalid type"))
            return
        }
        klog.V(4).Infof("Recovered deleted object '%s' from tombstone", object.GetName())
    }
    klog.V(4).Infof("Processing object: %s", object.GetName())
    if ownerRef := metav1.GetControllerOf(object); ownerRef != nil {
        // If this object is not owned by a Foo, we should not do anything more
        // with it.
        if ownerRef.Kind != "Foo" {
            return
        }

        foo, err := c.projectLister.IoTProjects(object.GetNamespace()).Get(ownerRef.Name)
        if err != nil {
            klog.V(4).Infof("ignoring orphaned object '%s' of foo '%s'", object.GetSelfLink(), ownerRef.Name)
            return
        }

        c.enqueueFoo(foo)
        return
    }
}
