// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"cloudeng.io/net/http/httptracing"
	"cloudeng.io/webapp"
)

func noCertError() string {
	switch runtime.GOOS {
	case "windows":
		return "x509: certificate signed by unknown authority"
	case "darwin":
		return "failed to verify certificate: x509: “localhost” certificate is not trusted"
	case "linux":
		return "x509: certificate signed by unknown authority"
	default:
		return "failed to verify certificate: x509: certificate signed by unknown authority"
	}
}

func newCert(t *testing.T, name string, isCA bool, signer *x509.Certificate, signerKey *rsa.PrivateKey) (*x509.Certificate, *rsa.PrivateKey) {
	t.Helper()
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(time.Now().Unix()),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}

	if isCA {
		template.IsCA = true
		template.KeyUsage |= x509.KeyUsageCertSign
	}

	var parent *x509.Certificate
	var parentKey *rsa.PrivateKey
	if signer == nil {
		parent = template
		parentKey = privKey
	} else {
		parent = signer
		parentKey = signerKey
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, parent, &privKey.PublicKey, parentKey)
	if err != nil {
		t.Fatal(err)
	}

	cert, err := x509.ParseCertificate(derBytes)
	if err != nil {
		t.Fatal(err)
	}

	return cert, privKey
}

func TestNewHTTPClient(t *testing.T) {
	ctx := context.Background()

	t.Run("default", func(t *testing.T) {
		client, err := webapp.NewHTTPClient(ctx)
		if err != nil {
			t.Fatalf("NewHTTPClient failed: %v", err)
		}
		if client == nil {
			t.Fatal("expected a client, got nil")
		}
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("expected a default http.Transport, got %T", client.Transport)
		}
		if transport.TLSClientConfig.RootCAs != nil {
			t.Error("expected default RootCAs to be nil")
		}
	})

	t.Run("with-custom-ca", func(t *testing.T) {
		// 1. Create a root CA and a server cert signed by it.
		rootCert, rootKey := newCert(t, "test-ca", true, nil, nil)
		serverCert, serverKey := newCert(t, "localhost", false, rootCert, rootKey)

		// 2. Start a TLS server with the server cert.
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		server.TLS = &tls.Config{
			MinVersion: tls.VersionTLS13,
			Certificates: []tls.Certificate{{
				Certificate: [][]byte{serverCert.Raw},
				PrivateKey:  serverKey,
			}},
		}
		server.StartTLS()
		defer server.Close()

		// 3. Write the root CA to a temp file.
		tmpDir := t.TempDir()
		caPemFile := filepath.Join(tmpDir, "ca.pem")
		pemBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: rootCert.Raw})
		if err := os.WriteFile(caPemFile, pemBytes, 0600); err != nil {
			t.Fatal(err)
		}

		// 4. Create a client with the custom CA and make a request.
		client, err := webapp.NewHTTPClient(ctx, webapp.WithCustomCAPEMFile(caPemFile))
		if err != nil {
			t.Fatalf("NewHTTPClient with custom CA failed: %v", err)
		}

		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("request with custom CA failed: %v", err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status OK, got %v", resp.Status)
		}

		// 5. Create a default client and ensure the request fails.
		defaultClient := &http.Client{}
		_, err = defaultClient.Get(server.URL)
		if err == nil {
			t.Fatal("expected request with default client to fail")
		}
		expected := noCertError()
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("expected %q, got: %v", expected, err)
		}
	})

	t.Run("with-tracing", func(t *testing.T) {
		var logBuf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

		client, err := webapp.NewHTTPClient(ctx, webapp.WithTracingTransport(
			httptracing.WithTracingLogger(logger),
		))
		if err != nil {
			t.Fatalf("NewHTTPClient with tracing failed: %v", err)
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("request with tracing client failed: %v", err)
		}
		resp.Body.Close()

		logOutput := logBuf.String()
		if !strings.Contains(logOutput, `"method":"GET`) {
			t.Errorf("log output does not contain request trace: %s", logOutput)
		}
	})
}
