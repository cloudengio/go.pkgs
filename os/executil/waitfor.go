// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import (
	"context"
	"fmt"
	"time"

	"cloudeng.io/algo/ratecontrol"
)

// WaitFor repeatedly calls the provided check function until it returns
// done=true, or the context is done. It waits for the specified
// interval between calls. If check returns an error, it is ignored
// unless done=true, in which case it is returned immediately.
func WaitFor(ctx context.Context, interval time.Duration, check func(ctx context.Context) (done bool, err error)) error {
	if interval <= 0 {
		return fmt.Errorf("executil: WaitFor: interval must be positive: %v", interval)
	}

	if done, err := check(ctx); done {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if done, err := check(ctx); done {
				return err
			}
		}
	}
}

// WaitForBackoff repeatedly calls the provided check function until it returns
// done=true, or the context is done. It waits for the next backoff delay
// between calls. If check returns an error, it is ignored unless done=true,
// in which case it is returned immediately.
func WaitForBackoff(ctx context.Context, backoff ratecontrol.Backoff, check func(ctx context.Context) (done bool, err error)) error {
	if done, err := check(ctx); done {
		return err
	}

	for {
		nextCh := backoff.Next()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-nextCh:
			if !ok {
				return fmt.Errorf("executil: WaitForBackoff: backoff done")
			}
			if done, err := check(ctx); done {
				return err
			}
		}
	}
}
