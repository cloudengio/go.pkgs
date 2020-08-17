// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// +build igore

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"cloudeng.io/cmdutil/signals"
)

var debounceFlag time.Duration

func init() {
	flag.DurationVar(&debounceFlag, "debounce", time.Second, "signal debouce delay")
}

func main() {
	flag.Parse()
	signals.DebounceDuration = time.Millisecond * 250
	ctx := context.Background()
	ctx, wait := signals.NotifyWithCancel(ctx, os.Interrupt)
	fmt.Printf("PID=%v\n", os.Getpid())
	sig := wait()
	time.Sleep(signals.DebounceDuration * 2)
	fmt.Println(sig.String())
}
