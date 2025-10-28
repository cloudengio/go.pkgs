// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package devtest_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"cloudeng.io/os/executil"
	"cloudeng.io/webapp/devtest"
)

type output struct {
	*strings.Builder
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
	pebble := devtest.NewPebble(mockPebblePath)

	cfg, err := devtest.NewPebbleConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	cfgFile, err := cfg.CreateCertsAndUpdateConfig(ctx, tmpDir)
	if err != nil {
		t.Fatalf("failed to create pebble certs: %v", err)
	}

	out := &output{&strings.Builder{}}
	if err := pebble.Start(ctx, ".", cfgFile, out); err != nil {
		t.Fatalf("failed to start pebble: %v", err)
	}

	if err := pebble.WaitForReady(ctx); err != nil {
		t.Fatalf("WaitForReady: %v", err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	serial, err := pebble.WaitForIssuedCertificateSerial(ctx)
	if err != nil {
		t.Fatalf("WaitForIssuedCertificateSerial: %v", err)
	}
	if got, want := serial, "0123456789abcdef"; got != want {
		t.Errorf("invalid serial: got %q, want %q", got, want)
	}

	ensureStopped(t, pebble)

}

func ensureStopped(t *testing.T, pebble *devtest.Pebble) {
	t.Helper()

	pid := pebble.PID()
	if pid == 0 {
		t.Fatalf("invalid pebble pid: %d", pid)
	}
	if err := pebble.Stop(); err != nil {
		t.Fatalf("failed to close pebble: %v", err)
	}

	// 5. Verify the process is gone.
	// On Unix, os.FindProcess always succeeds, so we need to signal it.
	// On Windows, FindProcess will error if the process doesn't exist.
	time.Sleep(100 * time.Millisecond) // Give it a moment to die.
	proc, err := os.FindProcess(pid)
	if err != nil && runtime.GOOS == "windows" {
		// On windows, this is sufficient to know it's gone.
		return
	}
	// On Unix, we need to send a signal 0 to check for existence.
	if err := proc.Signal(syscall.Signal(0)); err == nil {
		// If there's no error, the process still exists.
		t.Errorf("process %d still exists after close", pid)
		proc.Kill() //nolint:errcheck
	}
}

func TestPebble_RealServer(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()

	pebble := devtest.NewPebble("pebble")

	cfg, err := devtest.NewPebbleConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	cfgFile, err := cfg.CreateCertsAndUpdateConfig(ctx, filepath.Join(tmpDir, "test", "certs"))
	if err != nil {
		t.Fatalf("failed to create pebble certs: %v", err)
	}

	out := &output{&strings.Builder{}}
	if err := pebble.Start(ctx, tmpDir, cfgFile, out); err != nil {
		t.Logf("pebble log output: %s\n", out.String())
		t.Fatalf("failed to start pebble: %v", err)
	}
	if err := pebble.WaitForReady(ctx); err != nil {
		t.Logf("pebble log output: %s\n", out.String())
		t.Fatalf("WaitForReady: %v", err)
	}

	data, err := cfg.GetRootCert(ctx)
	fmt.Printf("PEM data: %s\n", data)
	t.Fail()
	fmt.Printf("out %s\n", out.String())
	if err != nil {
		t.Fatalf("GetRootCert: %v", err)
	}

	ensureStopped(t, pebble)
}
