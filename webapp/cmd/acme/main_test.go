// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cloudeng.io/logging"
	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/webapp/webauth/acme"
	"cloudeng.io/webapp/webauth/acme/pebble"
)

type output struct {
	io.Writer
}

func (o *output) Close() error {
	return nil
}

func startPebble(ctx context.Context, t *testing.T, tmpDir string, validitySec int) (*pebble.T, pebble.Config, *strings.Builder, string, string) {
	t.Helper()
	pebbleCacheDir := filepath.Join(tmpDir, "certcache")
	if err := os.MkdirAll(pebbleCacheDir, 0700); err != nil {
		t.Fatal(err)
	}

	pebbleServer := pebble.New("pebble")
	pebbleTestDir := filepath.Join(tmpDir, "pebble-test")
	cfg := pebble.NewConfig(
		pebble.WithValidityPeriod(validitySec), // 5 seconds validity for fast renewal testing
	)
	pebbleCfg, err := cfg.CreateCertsAndUpdateConfig(ctx, pebbleTestDir)
	if err != nil {
		t.Fatalf("failed to create pebble certs: %v", err)
	}

	out := &strings.Builder{}
	if err := pebbleServer.Start(ctx, pebbleTestDir, pebbleCfg, &output{io.MultiWriter(out, os.Stderr)}); err != nil {
		t.Fatalf("failed to start pebble: %v", err)
	}
	if err := pebbleServer.WaitForReady(ctx); err != nil {
		t.Fatalf("pebble not ready: %v\n%s", err, out.String())
	}
	t.Logf("cert cache dir: %s", pebbleCacheDir)
	t.Logf("pebble dir: %s", pebbleTestDir)
	return pebbleServer, cfg, out, pebbleCacheDir, pebbleTestDir
}

func TestACMEMain(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(logging.NewJSONFormatter(os.Stderr, "", "  "), nil))
	ctx := ctxlog.WithLogger(t.Context(), logger)

	validityPeriodSecs := 5 // seconds
	tmpDir := t.TempDir()

	pebbleServer, pebbleCfg, pebbleOut, pebbleCacheDir, pebbleTestDir := startPebble(ctx, t, tmpDir, validityPeriodSecs)
	defer pebbleServer.EnsureStopped(ctx, time.Second) //nolint:errcheck
	_ = pebbleOut
	mgrFlags := certManagerFlags{
		ServiceFlags: acme.ServiceFlags{
			ClientHost:  pebbleCfg.Address,
			Provider:    pebbleCfg.DirectoryURL(),
			RenewBefore: time.Second * 2,
			Email:       "dev@cloudeng.io",
		},
		HTTPPort:        pebbleCfg.HTTPPort,
		TestingCAPem:    filepath.Join(pebbleTestDir, pebbleCfg.CAFile),
		RefreshInterval: time.Second,
		TLSCertStoreFlags: TLSCertStoreFlags{
			LocalCacheDir: pebbleCacheDir,
		},
	}

	var errCh = make(chan error, 1)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		err := certManagerCmd{}.manageCerts(ctx, &mgrFlags, []string{"pebble-test.example.com"})
		t.Logf("cert manager exited: %v", err)
		errCh <- err
	}()

	serial, err := pebbleServer.WaitForIssuedCertificateSerial(ctx)
	if err != nil {
		t.Fatalf("failed to wait for issued certificate serial: %v", err)
	}

	localhostCert := filepath.Join(pebbleCacheDir, "certs", "pebble-test.example.com")

	wctx, wcancel := context.WithTimeout(ctx, 10*time.Second)
	defer wcancel()
	cert := waitForCertWithSerial(wctx, t, "new certififcate", localhostCert, serial)
	if cert.NotAfter.Sub(cert.NotBefore) > time.Duration(validityPeriodSecs)*time.Second {
		t.Errorf("expected short lived certificate, got validity %v", cert.NotAfter.Sub(cert.NotBefore))
	}

	// test renewal
	renewedSerial, err := pebbleServer.WaitForIssuedCertificateSerial(ctx)
	if err != nil {
		t.Fatalf("failed to wait for renewed certificate serial: %v", err)
	}
	if renewedSerial == serial {
		t.Fatalf("expected a new serial number, but got the same one: %v", renewedSerial)
	}

	wctx, wcancel = context.WithTimeout(ctx, 10*time.Second)
	defer wcancel()
	waitForCertWithSerial(wctx, t, "certificate refresh", localhostCert, renewedSerial)

	cancel()

	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Fatalf("cert manager exited with unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for cert manager to exit")
	}

}

func waitForCertWithSerial(ctx context.Context, t *testing.T, msg, certPath, serial string) *x509.Certificate {
	t.Helper()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("%v: timed out waiting for cert %v with serial %v: %v", msg, certPath, serial, ctx.Err())
		case <-ticker.C:
			if _, err := os.Stat(certPath); err != nil {
				continue
			}
			leafCert := getLeafCert(t, certPath)
			gotSerial := fmt.Sprintf("%0*x", len(leafCert.SerialNumber.Bytes())*2, leafCert.SerialNumber)
			if gotSerial == serial {
				t.Logf("%v: found cert %v with serial %v", msg, certPath, serial)
				return leafCert
			}
			t.Logf("%v: waiting for serial %v, got %v", msg, serial, gotSerial)
		}
	}
}

func getLeafCert(t *testing.T, certPath string) *x509.Certificate {
	t.Helper()
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("failed to read cert file %v: %v", certPath, err)
	}
	var leafCert *x509.Certificate
	for {
		var block *pem.Block
		block, certPEM = pem.Decode(certPEM)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			continue
		}
		if !cert.IsCA {
			leafCert = cert
			break
		}
	}

	if leafCert == nil {
		t.Fatalf("failed to find leaf certificate in %v", certPath)
	}
	// Format the serial number as a hex string, with leading zeros to match
	// the length of the serial number.
	return leafCert
}
