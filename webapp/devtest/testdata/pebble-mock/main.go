// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	var config string
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	flag.StringVar(&config, "config", "", "path to pebble config file")
	flag.Parse()

	time.Sleep(100 * time.Millisecond)

	_, err := os.ReadFile(config)
	if err != nil {
		fmt.Printf("Failed to read config file: %v\n", err)
		os.Exit(1)
		return
	}

	fmt.Printf("ACME directory available at: https://0.0.0.0:14000/dir\n")
	fmt.Printf("Root CA certificate available at: https://0.0.0.0:15000/roots/0\n")
	fmt.Printf("Issued certificate serial 0123456789abcdef for order\n")
	// Keep running until killed by the test's Stop() method.
	select {
	case <-sigCh:
		fmt.Printf("Received signal, exiting\n")
		return
	case <-time.After(10 * time.Minute):
		fmt.Printf("Timeout reached, exiting\n")
		os.Exit(1)
		return
	}
}
