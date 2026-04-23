// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil

import (
	"context"
	"fmt"
	"time"
)

// WaitFor repeatedly calls the provided check function until it returns
// done=true or the context is done. It waits for the specified interval between
// calls. If check returns an error, it is returned immediately.
func WaitFor(ctx context.Context, interval time.Duration, check func(ctx context.Context) (done bool, err error)) error {
	if interval <= 0 {
		return fmt.Errorf("vms: WaitForSomething: interval must be positive: %v", interval)
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
