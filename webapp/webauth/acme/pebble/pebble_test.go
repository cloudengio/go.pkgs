// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pebble_test

import (
	"context"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"cloudeng.io/os/executil"
	"cloudeng.io/webapp/webauth/acme/pebble"
)

type output struct {
	mu  sync.Mutex
	out []byte
}

func (o *output) Write(p []byte) (n int, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.out = append(o.out, p...)
	return len(p), nil
}

func (o *output) String() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return string(o.out)
}

func (o *output) Close() error {
	return nil
}

func TestPebble(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	mockPebblePath, err := executil.GoBuild(ctx, filepath.Join(tmpDir, "pebble"), "./testdata/pebble-mock")
	if err != nil {
		t.Fatalf("failed to build mock pebble: %v", err)
	}
	p := pebble.New(mockPebblePath)
	out := &output{}
	defer ensureStopped(t, p, out)

	cfg := pebble.NewConfig()

	cfgFile, err := cfg.CreateCertsAndUpdateConfig(ctx, tmpDir)
	if err != nil {
		t.Fatalf("failed to create pebble certs: %v", err)
	}

	if err := p.Start(ctx, ".", cfgFile, out); err != nil {
		t.Fatalf("failed to start pebble: %v", err)
	}

	if err := p.WaitForReady(ctx); err != nil {
		t.Fatalf("WaitForReady: %v", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	serial, err := p.WaitForIssuedCertificateSerial(ctx)
	if err != nil {
		t.Fatalf("WaitForIssuedCertificateSerial: %v", err)
	}
	if got, want := serial, "0123456789abcdef"; got != want {
		t.Errorf("invalid serial: got %q, want %q", got, want)
	}

}

func ensureStopped(t *testing.T, p *pebble.T, out *output) {
	t.Helper()
	if err := p.EnsureStopped(t.Context(), time.Minute); err != nil {
		t.Logf("pebble log output: %s\n", out.String())
		t.Fatalf("failed to stop pebble process %d: %v", p.PID(), err)
	}
}

func TestPebble_RealServer(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()

	p := pebble.New("pebble")
	out := &output{}
	defer ensureStopped(t, p, out)

	cfg := pebble.NewConfig()

	cfgFile, err := cfg.CreateCertsAndUpdateConfig(ctx, tmpDir)
	if err != nil {
		t.Fatalf("failed to create pebble certs: %v", err)
	}

	if err := p.Start(ctx, tmpDir, cfgFile, out); err != nil {
		t.Logf("pebble log output: %s\n", out.String())
		t.Fatalf("failed to start pebble: %v", err)
	}
	if err := p.WaitForReady(ctx); err != nil {
		t.Logf("pebble log output: %s\n", out.String())
		t.Fatalf("WaitForReady: %v", err)
	}

	for range 2 {
		if _, err := cfg.GetIssuingCA(ctx, 0); err != nil {
			// Fix for linux CI runners wjere ipv6 does not seem to work.
			if !strings.Contains(err.Error(), "dial tcp [::1]:15000: connect: connection refused") {
				t.Logf("pebble log output: %s\n", out.String())
				t.Fatalf("GetIssuingCA: %v", err)
			}
		}
	}
}

func TestPossibleValidityPeriods(t *testing.T) {
	cfg := pebble.NewConfig()
	periods := cfg.PossibleValidityPeriods()

	expected := []time.Duration{
		7776000 * time.Second,
		518400 * time.Second,
	}

	// Sort both slices to ensure comparison is order-independent.
	sort.Slice(periods, func(i, j int) bool { return periods[i] < periods[j] })
	sort.Slice(expected, func(i, j int) bool { return expected[i] < expected[j] })

	if !reflect.DeepEqual(periods, expected) {
		t.Errorf("got %v, want %v", periods, expected)
	}
}
