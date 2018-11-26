/*
 * Copyright 2018, EnMasse authors.
 * License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).
 */

package main

import (
    "flag"
    "os"

    iot "github.com/ctron/hono-qdrouter-proxy/pkg/client/clientset/versioned"
    "github.com/ctron/hono-qdrouter-proxy/pkg/signals"

    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/klog"

    "time"

    informers "github.com/ctron/hono-qdrouter-proxy/pkg/client/informers/externalversions"
)

var (
    masterURL  string
    kubeconfig string
)

func main() {
    flag.Parse()

    // init log system
    klog.SetOutput(os.Stdout)

    // install signal handler for graceful shutdown, or hard exit
    stopCh := signals.InstallSignalHandler()

    klog.Infof("Starting up...")

    cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
    if err != nil {
        klog.Fatalf("Error building kubeconfig: %s", err.Error())
    }

    kubeClient, err := kubernetes.NewForConfig(cfg)
    if err != nil {
        klog.Fatalf("Error building kubernetes client: %s", err.Error())
    }

    iotClient, err := iot.NewForConfig(cfg)
    if err != nil {
        klog.Fatalf("Error building IoT project client: %s", err.Error())
    }

    iotInformerFactory := informers.NewSharedInformerFactory(iotClient, time.Second*30)

    controller := NewController(
        kubeClient, iotClient,
        iotInformerFactory.Hono().V1alpha1().IoTProjects(),
    )

    iotInformerFactory.Start(stopCh)

    if err = controller.Run(2, stopCh); err != nil {
        klog.Fatalf("Error running controller: %s", err.Error())
    }
}
