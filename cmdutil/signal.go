// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdutil

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
)

// HandleSignals will asynchronously invoke the supplied function when the
// specified signals are received. Passing no signals requests that all
// signals be handled.
func HandleSignals(fn func(), signals ...os.Signal) {
	sigCh := make(chan os.Signal, 1)
	if len(signals) == 0 {
		signal.Notify(sigCh)
	} else {
		signal.Notify(sigCh, signals...)
	}
	go func() {
		sig := <-sigCh
		fmt.Println("stopping on... ", sig)
		fn()
	}()
}

// Exit formats and prints the supplied parameters to os.Stderr and then
// calls os.Exit(1).
func Exit(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
