/*
 * Copyright 2018, EnMasse authors.
 * License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).
 */

package signals

import (
    "os"
    "os/signal"
)

// global instance to ensure the SetupSignalHandler methods only gets called once
var enforceCalledOnlyOnce = make(chan struct{})

// install signal handler
//
// Setup and install signal handler to gracefully shut down the application on the first signal
// and simply "exit" on the second signal.
func InstallSignalHandler() (stopCh <-chan struct{}) {

    // the next call will fail if the method gets called
    // a second time.
    close(enforceCalledOnlyOnce)

    stop := make(chan struct{})
    c := make(chan os.Signal, 2)
    signal.Notify(c, os.Interrupt)
    go func() {
        // wait for signal
        <-c
        // initiate graceful shutdown
        close(stop)
        // wait for next signal
        <-c
        // second signal, hard exit
        os.Exit(1)
    }()

    return stop
}
