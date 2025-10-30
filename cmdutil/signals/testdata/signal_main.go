// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build igore

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"cloudeng.io/cmdutil/signals"
)

var (
	debounceFlag time.Duration
	sleepFlag    time.Duration
)

func init() {
	flag.DurationVar(&debounceFlag, "debounce", time.Second, "signal debouce delay")
	flag.DurationVar(&sleepFlag, "sleep", time.Second, "post signal sleep duration")

}

func main() {
	flag.Parse()
	signals.DebounceDuration = time.Millisecond * 250
	ctx := context.Background()
	ctx, handler := signals.NotifyWithCancel(ctx, os.Interrupt)
	handler.RegisterCancel(func() {
		fmt.Printf("CANCEL PID=%v\n", os.Getpid())
	})
	fmt.Printf("PID=%v\n", os.Getpid())
	sig := handler.WaitForSignal()
	time.Sleep(sleepFlag)
	fmt.Println(sig.String())
}
