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

func startPebble(ctx context.Context, t *testing.T, tmpDir string, configOpts ...pebble.ConfigOption) (*pebble.T, pebble.Config, *strings.Builder, string, string) {
	t.Helper()
	pebbleCacheDir := filepath.Join(tmpDir, "certcache")
	if err := os.MkdirAll(pebbleCacheDir, 0700); err != nil {
		t.Fatal(err)
	}

	pebbleServer := pebble.New("pebble")
	pebbleTestDir := filepath.Join(tmpDir, "pebble-test")

	cfg := pebble.NewConfig(configOpts...)
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

func defaultManagerFlags(pebbleCfg pebble.Config, pebbleTestDir, pebbleCacheDir string) certManagerFlags {
	return certManagerFlags{
		ServiceFlags: acme.ServiceFlags{
			ClientHost: pebbleCfg.Address,
			Provider:   pebbleCfg.DirectoryURL(),
			Email:      "dev@cloudeng.io",
		},
		HTTPPort:        pebbleCfg.HTTPPort,
		TestingCAPem:    filepath.Join(pebbleTestDir, pebbleCfg.CAFile),
		RefreshInterval: time.Minute,
		TLSCertStoreFlags: TLSCertStoreFlags{
			LocalCacheDir: pebbleCacheDir,
		},
	}
}

func TestNewCert(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(logging.NewJSONFormatter(os.Stderr, "", "  "), nil))
	ctx := ctxlog.WithLogger(t.Context(), logger)

	tmpDir := t.TempDir()

	pebbleServer, pebbleCfg, _, pebbleCacheDir, pebbleTestDir := startPebble(ctx, t, tmpDir)
	defer pebbleServer.EnsureStopped(ctx, time.Second) //nolint:errcheck

	mgrFlags := defaultManagerFlags(pebbleCfg, pebbleTestDir, pebbleCacheDir)
	mgrFlags.RefreshInterval = time.Second

	stopAndWaitForCertManager := runCertManager(ctx, t, &mgrFlags, "pebble-test.example.com")

	// Wait for at least one certificate to be issued.
	if _, err := pebbleServer.WaitForOrderAuthorized(ctx); err != nil {
		t.Fatalf("failed to wait for issued certificate serial: %v", err)
	}

	localhostCert := filepath.Join(pebbleCacheDir, "certs", "pebble-test.example.com")
	leaf, intermediates := waitForNewCert(ctx, t, "new cert", localhostCert, "")

	if err := leaf.VerifyHostname("pebble-test.example.com"); err != nil {
		t.Fatalf("hostname verification failed: %v", err)
	}

	if err := pebbleCfg.ValidateCertificate(ctx, leaf, intermediates); err != nil {
		t.Fatalf("failed to validate certificate: %v", err)
	}

	validFor := leaf.NotAfter.Sub(leaf.NotBefore)
	found := false
	for _, period := range pebbleCfg.PossibleValidityPeriods() {
		if durationWithin(period, validFor, time.Second*10) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected validity period to be one of %v, got %v", pebbleCfg.PossibleValidityPeriods(), validFor)
	}

	stopAndWaitForCertManager(t)
}

func TestCertRenewal(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(logging.NewJSONFormatter(os.Stderr, "", "  "), nil))
	ctx := ctxlog.WithLogger(t.Context(), logger)

	tmpDir := t.TempDir()

	pebbleServer, pebbleCfg, _, pebbleCacheDir, pebbleTestDir := startPebble(ctx, t, tmpDir,
		pebble.WithValidityPeriod(10), // short lived certs to force renewal
	)
	defer pebbleServer.EnsureStopped(ctx, time.Second) //nolint:errcheck

	mgrFlags := defaultManagerFlags(pebbleCfg, pebbleTestDir, pebbleCacheDir)
	mgrFlags.ServiceFlags.RenewBefore = time.Second * 15 // allow immediate renewal
	mgrFlags.RefreshInterval = time.Second

	stopAndWaitForCertManager := runCertManager(ctx, t, &mgrFlags, "pebble-test.example.com")

	var previousSerial string
	for i := range 3 {
		// Wait for a certificate to be issued.
		if _, err := pebbleServer.WaitForOrderAuthorized(ctx); err != nil {
			t.Fatalf("%v: failed to wait for issued certificate serial: %v", i, err)
		}

		localhostCert := filepath.Join(pebbleCacheDir, "certs", "pebble-test.example.com")

		leaf, intermediates := waitForNewCert(ctx, t,
			fmt.Sprintf("waiting for cert %v", i),
			localhostCert, previousSerial)

		if err := leaf.VerifyHostname("pebble-test.example.com"); err != nil {
			t.Fatalf("%v: hostname verification failed: %v", i, err)
		}

		if err := pebbleCfg.ValidateCertificate(ctx, leaf, intermediates); err != nil {
			t.Fatalf("%v: failed to validate certificate: %v", i, err)
		}

		validFor := leaf.NotAfter.Sub(leaf.NotBefore)
		serial := fmt.Sprintf("%0*x", len(leaf.SerialNumber.Bytes())*2, leaf.SerialNumber)
		t.Logf("obtained certificate %v valid for %v (serial %v)", i, validFor, serial)
		previousSerial = serial
	}

	stopAndWaitForCertManager(t)
}

func durationWithin(d1, d2, tolerance time.Duration) bool {
	diff := d1 - d2
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}

func runCertManager(ctx context.Context, t *testing.T, flags *certManagerFlags, host string) func(t *testing.T) {
	t.Helper()
	errCh := make(chan error, 1)
	go func() {
		err := certManagerCmd{}.manageCerts(ctx, flags, []string{host})
		t.Logf("cert manager exited: %v", err)
		errCh <- err
	}()
	ctx, cancel := context.WithCancel(ctx)
	return func(t *testing.T) {
		cancel()
		waitForServer(t, errCh)
	}
}

func waitForServer(t *testing.T, errCh <-chan error) {
	t.Logf("waiting for cert manager to exit")
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Fatalf("cert manager exited with unexpected error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for cert manager to exit")
	}
}

/*
func waitForCert(ctx context.Context, t *testing.T, msg, certPath string) (*x509.Certificate, *x509.CertPool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("%v: timed out waiting for cert %v: %v", msg, certPath, ctx.Err())
		case <-ticker.C:
			if _, err := os.Stat(certPath); err != nil {
				continue
			}
			return getCerts(t, certPath)

		}
	}
}*/

func waitForNewCert(ctx context.Context, t *testing.T, msg, certPath string, previousSerial string) (*x509.Certificate, *x509.CertPool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("%v: timed out waiting for new cert %v: %v", msg, certPath, ctx.Err())
		case <-ticker.C:
			if _, err := os.Stat(certPath); err != nil {
				continue
			}
			leafCert, intermediates := getCerts(t, certPath)
			gotSerial := fmt.Sprintf("%0*x", len(leafCert.SerialNumber.Bytes())*2, leafCert.SerialNumber)
			if gotSerial != previousSerial {
				t.Logf("%v: found new cert %v with serial %v", msg, certPath, gotSerial)
				return leafCert, intermediates
			}
			t.Logf("%v: waiting for new cert, previous serial %v, got %v", msg, previousSerial, gotSerial)
		}
	}
}

/*
	issuingCA, err := pebbleCfg.GetIssuingCA(ctx, 0)
	if err != nil {
		t.Fatalf("failed to get issuing CA: %v", err)
	}
	if _, err := leaf.Verify(x509.VerifyOptions{
		Intermediates: intermediates,
		Roots:         issuingCA,
	}); err != nil {
		t.Fatalf("certificate verification failed: %v", err)
	}*/

/*
	// test restart of cert manager picks up existing certs and renews them
	stopAndWaitForCertManager = runCertManager(ctx, t, &mgrFlags, "pebble-test.example.com")

	// test renewal
	renewedSerial, err := pebbleServer.WaitForIssuedCertificateSerial(ctx)
	if err != nil {
		t.Fatalf("failed to wait for renewed certificate serial: %v", err)
	}
	t.Logf("refreshed certificate issued with serial: %v", serial)
	if renewedSerial == serial {
		t.Fatalf("expected a new serial number, but got the same one: %v", renewedSerial)
	}

	waitForCertWithSerial(ctx, t, "certificate refresh", localhostCert, renewedSerial)

	stopAndWaitForCertManager(t)*/
/*cert := waitForCertWithSerial(ctx, t, "new certififcate", localhostCert, serial)
if cert.NotAfter.Sub(cert.NotBefore) > time.Duration(validityPeriodSecs)*time.Second {
	t.Errorf("expected short lived certificate, got validity %v", cert.NotAfter.Sub(cert.NotBefore))
}*/
/*
		serial, err := pebbleServer.WaitForIssuedCertificateSerial(ctx)
		if err != nil {
			t.Fatalf("failed to wait for issued certificate serial: %v", err)
		}
		t.Logf("initial certificate issued with serial: %v", serial)

	// test initial cert obtain
*/

/*gotSerial := fmt.Sprintf("%0*x", len(leafCert.SerialNumber.Bytes())*2, leafCert.SerialNumber)
if gotSerial == serial {
	t.Logf("%v: found cert %v with serial %v", msg, certPath, serial)
	t.Logf("%v: cert %v valid from %v to %v\n", msg, serial, leafCert.NotBefore, leafCert.NotAfter)
	return leafCert
}
t.Logf("%v: waiting for serial %v, got %v", msg, serial, gotSerial)*/
/*
func waitForCertWithSerial(ctx context.Context, t *testing.T, msg, certPath, serial string) *x509.Certificate {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
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
			leafCert := getCerts(t, certPath)
			gotSerial := fmt.Sprintf("%0*x", len(leafCert.SerialNumber.Bytes())*2, leafCert.SerialNumber)
			if gotSerial == serial {
				t.Logf("%v: found cert %v with serial %v", msg, certPath, serial)
				t.Logf("%v: cert %v valid from %v to %v\n", msg, serial, leafCert.NotBefore, leafCert.NotAfter)
				return leafCert
			}
			t.Logf("%v: waiting for serial %v, got %v", msg, serial, gotSerial)
		}
	}
}
*/

func getCerts(t *testing.T, certPath string) (*x509.Certificate, *x509.CertPool) {
	t.Helper()
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("failed to read cert file %v: %v", certPath, err)
	}
	var leafCert *x509.Certificate
	intermediates := x509.NewCertPool()
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
			continue
		}
		intermediates.AddCert(cert)
	}

	if leafCert == nil {
		t.Fatalf("failed to find leaf certificate in %v", certPath)
	}
	return leafCert, intermediates
}
