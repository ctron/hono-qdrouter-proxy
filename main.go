package main

import (
    "flag"
    "os"

    "github.com/ctron/hono-qdrouter-proxy/pkg/signals"

    kubeinformers "k8s.io/client-go/informers"

    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
    "k8s.io/klog"

    "time"

    clientset "github.com/ctron/hono-qdrouter-proxy/pkg/client/clientset/versioned"
    informers "github.com/ctron/hono-qdrouter-proxy/pkg/client/informers/externalversions"
)

var (
    masterURL  string
    kubeconfig string
)

func main() {
    flag.Parse()

    klog.SetOutput(os.Stdout)

    // set up signals so we handle the first shutdown signal gracefully
    stopCh := signals.SetupSignalHandler()

    klog.Infof("Starting up...")

    cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
    if err != nil {
        klog.Fatalf("Error building kubeconfig: %s", err.Error())
    }

    kubeClient, err := kubernetes.NewForConfig(cfg)
    if err != nil {
        klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
    }

    iotClient, err := clientset.NewForConfig(cfg)
    if err != nil {
        klog.Fatalf("Error building example clientset: %s", err.Error())
    }

    kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
    iotInformerFactory := informers.NewSharedInformerFactory(iotClient, time.Second*30)

    controller := NewController(kubeClient, iotClient,
        iotInformerFactory.Hono().V1alpha1().IoTProjects())

    // notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
    // Start method is non-blocking and runs all registered informers in a dedicated goroutine.
    kubeInformerFactory.Start(stopCh)
    iotInformerFactory.Start(stopCh)

    if err = controller.Run(2, stopCh); err != nil {
        klog.Fatalf("Error running controller: %s", err.Error())
    }
}

func init() {
    flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
    flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
