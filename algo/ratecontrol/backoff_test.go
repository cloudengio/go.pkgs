// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ratecontrol_test

import (
	"context"
	"testing"
	"time"

	"cloudeng.io/algo/ratecontrol"
)

func TestBackoffOffset(t *testing.T) {
	ctx := context.Background()
	numRetries := 10
	bo := ratecontrol.NewExponentialBackoffOffset(time.Millisecond, numRetries)

	for i := range numRetries {
		done, err := bo.Wait(ctx, nil)
		if err != nil {
			t.Fatalf("retry %d: %v", i, err)
		}
		if done {
			t.Fatalf("expected to not be done on retry %d", i)
		}
	}

	done, err := bo.Wait(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !done {
		t.Fatal("expected to be done after max steps")
	}

	if got, want := bo.Retries(), numRetries; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
