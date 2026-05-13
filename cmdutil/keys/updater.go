// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package keys

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/file"
	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/sync/ctxsync"
)

// KeyStoreUpdater provides methods to automatically refresh keys in an
// InMemoryKeyStore from YAML or JSON files.
type KeyStoreUpdater struct {
	ims    *InMemoryKeyStore
	doneCh chan struct{}
	mu     sync.Mutex
	wg     ctxsync.WaitGroup
	closed bool
}

const (
	// DefaultRefreshInterval is the default interval at which the KeyStoreUpdater wil
	// refresh keys from the underlying KeyStore.
	DefaultRefreshInterval = time.Minute
)

// NewKeyStoreUpdater creates a new KeyStoreUpdater that will refresh keys in
// the provided InMemoryKeyStore. Call ScheduleRefreshYAML or ScheduleRefreshJSON
// to start the refresh process. Call Stop to stop the refresh process and wait
// for any in-flight refreshes to complete.
func NewKeyStoreUpdater(ims *InMemoryKeyStore) *KeyStoreUpdater {

	return &KeyStoreUpdater{ims: ims, doneCh: make(chan struct{})}
}

// ScheduleRefreshYAML starts a goroutine that will refresh keys in the underlying
// KeyStore from the specified YAML files at the configured refresh interval.
// The refresh process will continue until the context is canceled or Stop is called.
func (u *KeyStoreUpdater) ScheduleRefreshYAML(ctx context.Context, fs file.ReadFileFS, files ...string) {
	u.scheduleRefresh(ctx, true, 0, fs, files...)
}

// ScheduleRefreshJSON starts a goroutine that will refresh keys in the underlying
// KeyStore from the specified JSON files at the configured refresh interval.
// The refresh process will continue until the context is canceled or Stop is called.
func (u *KeyStoreUpdater) ScheduleRefreshJSON(ctx context.Context, fs file.ReadFileFS, files ...string) {
	u.scheduleRefresh(ctx, false, 0, fs, files...)
}

func (u *KeyStoreUpdater) Stop(ctx context.Context) {
	u.mu.Lock()
	if u.closed {
		u.mu.Unlock()
		return
	}
	u.closed = true
	u.mu.Unlock()
	close(u.doneCh)
	u.wg.Wait(ctx)
}

func (u *KeyStoreUpdater) scheduleRefresh(ctx context.Context, yamlOrJSON bool, interval time.Duration, fs file.ReadFileFS, files ...string) {
	if interval <= 0 {
		interval = DefaultRefreshInterval
	}
	ticker := time.NewTicker(interval)
	logger := ctxlog.Logger(ctx).With("component", "KeyStoreUpdater", "files", strings.Join(files, ","))

	u.wg.Go(func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				err := u.refresh(ctx, yamlOrJSON, fs, files...)
				if err != nil {
					logger.Error("error refreshing keys", "error", err)
				}
			case <-ctx.Done():
				return
			case <-u.doneCh:
				return
			}
		}
	})
}

func (u *KeyStoreUpdater) refresh(ctx context.Context, yamlOrJSON bool, fs file.ReadFileFS, files ...string) error {
	var errs errors.M
	for _, file := range files {
		var err error
		if yamlOrJSON {
			err = u.ims.ReadYAML(ctx, fs, file)
		} else {
			err = u.ims.ReadJSON(ctx, fs, file)
		}
		if err != nil {
			errs.Append(fmt.Errorf("refreshing keys from file %q: %w", file, err))
		}
	}
	return errs.Err()
}
