// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pebbletest

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cloudeng.io/webapp/webauth/acme/pebble"
)

type Testing interface {
	Fatalf(format string, args ...any)
	Helper()
	Logf(format string, args ...any)
}

// Recorder is an io.WriteCloser that records all data written to it.
type Recorder struct {
	mu  sync.Mutex
	buf []byte
}

func (o *Recorder) Close() error {
	return nil
}

func (o *Recorder) Write(p []byte) (n int, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.buf = append(o.buf, p...)
	return len(p), nil
}

func (o *Recorder) String() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return string(o.buf)
}

// Start starts a pebble ACME server for testing purposes.
func Start(ctx context.Context, t Testing, tmpDir string, configOpts ...pebble.ConfigOption) (*pebble.T, pebble.Config, *Recorder, string, string) {
	t.Helper()
	pebbleCacheDir := filepath.Join(tmpDir, "certcache")
	if err := os.MkdirAll(pebbleCacheDir, 0700); err != nil {
		t.Fatalf("failed to create pebble cache dir: %v", err)
	}

	pebbleServer := pebble.New("pebble")
	pebbleTestDir := filepath.Join(tmpDir, "pebble-test")

	cfg := pebble.NewConfig(configOpts...)
	pebbleCfg, err := cfg.CreateCertsAndUpdateConfig(ctx, pebbleTestDir)
	if err != nil {
		t.Fatalf("failed to create pebble certs: %v", err)
	}
	out := &Recorder{}
	if err := pebbleServer.Start(ctx, pebbleTestDir, pebbleCfg, out); err != nil {
		t.Fatalf("failed to start pebble: %v", err)
	}
	if err := pebbleServer.WaitForReady(ctx); err != nil {
		t.Fatalf("pebble not ready: %v\n%s", err, out.String())
	}
	t.Logf("cert cache dir: %s", pebbleCacheDir)
	t.Logf("pebble dir: %s", pebbleTestDir)
	return pebbleServer, cfg, out, pebbleCacheDir, pebbleTestDir
}

// WaitForNewCert waits for a new certificate to be issued at certPath with a
// serial number different from previousSerial.
func WaitForNewCert(ctx context.Context, t Testing, msg, certPath string, previousSerial string) (*x509.Certificate, *x509.CertPool) {
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

func getCerts(t Testing, certPath string) (*x509.Certificate, *x509.CertPool) {
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
			t.Logf("warning: failed to parse certificate in %v: %v", certPath, err)
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
