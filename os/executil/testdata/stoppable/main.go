// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var hang = flag.Bool("hang", false, "hang instead of exiting on signal")

func main() {
	flag.Parse()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// The test process will read this from stdout to get the pid.
	log.Printf("pid: %d, hanging %v\n", os.Getpid(), *hang)

	if *hang {
		// Ignore the first signal and just wait to be killed.
		<-sigs
		log.Printf("hanging\n")
		// Now, hang forever until killed.
		select {}
	} else {
		// Exit gracefully on the first signal.
		<-sigs
		log.Printf("graceful exit\n")
		os.Exit(0)
	}
}
