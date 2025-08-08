// Copyright 2021 cloudeng llc. All rights reserved.
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
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/webapp"
)

// newSelfSignedTLSConfig creates a self-signed certificate and returns a tls.Config
// for a server, and the certificate in PEM format for a client to trust.
func newSelfSignedTLSConfig(t *testing.T) (*tls.Config, []byte) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Co"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatalf("failed to create tls key pair: %v", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
	}, certPEM
}

func TestServeWithShutdown(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	ctx := ctxlog.WithLogger(context.Background(), logger)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	})

	ln, srv, err := webapp.NewHTTPServer("127.0.0.1:0", handler)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := webapp.ServeWithShutdown(ctx, ln, srv, time.Second); err != nil {
			log.Printf("ServeWithShutdown returned an error: %v", err)
		}
	}()

	client := http.Client{}
	resp, err := client.Get("http://" + ln.Addr().String())
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	if got, want := string(body), "hello"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	cancel()
	wg.Wait()

	if !strings.Contains(logBuf.String(), "shutting down server") {
		t.Errorf("log output missing 'shutting down server':\n%s", logBuf.String())
	}
}

func TestServeTLSWithShutdown(t *testing.T) {
	serverTLSConf, certPEM := newSelfSignedTLSConfig(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello tls")
	})
	ln, srv, err := webapp.NewTLSServer("127.0.0.1:0", handler, serverTLSConf)
	if err != nil {
		t.Fatal(err)
	}

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	ctx := ctxlog.WithLogger(context.Background(), logger)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := webapp.ServeTLSWithShutdown(ctx, ln, srv, time.Second); err != nil {
			log.Printf("ServeTLSWithShutdown returned an error: %v", err)
		}
	}()

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(certPEM)
	clientTLSConf := &tls.Config{
		RootCAs: certPool,
	}
	transport := &http.Transport{
		TLSClientConfig: clientTLSConf,
	}
	client := &http.Client{Transport: transport}

	resp, err := client.Get("https://" + ln.Addr().String())
	if err != nil {
		t.Fatalf("https.Get: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("io.ReadAll: %v", err)
	}
	if got, want := string(body), "hello tls"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	cancel()
	wg.Wait()

	if !strings.Contains(logBuf.String(), "shutting down server") {
		t.Errorf("log output missing 'shutting down server':\n%s", logBuf.String())
	}
}

func TestServeWithShutdown_Error(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	ctx := ctxlog.WithLogger(context.Background(), logger)
	ctx, cancel := context.WithCancel(ctx)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	// Close listener immediately to cause srv.Serve to fail.
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := webapp.ServeWithShutdown(ctx, ln, &http.Server{}, time.Second)
		if err != nil {
			t.Errorf("ServeWithShutdown returned an unexpected error: %v", err)
		}
	}()

	// Give the server goroutine a moment to start and fail on Serve.
	time.Sleep(100 * time.Millisecond)

	// Now, trigger the shutdown.
	cancel()
	wg.Wait()

	output := logBuf.String()
	if !strings.Contains(output, "server error") || !strings.Contains(output, "use of closed network connection") {
		t.Errorf("expected 'server error' and 'use of closed network connection' in log, got:\n%s", output)
	}
	if !strings.Contains(output, "shutting down server") {
		t.Errorf("log output missing 'shutting down server':\n%s", output)
	}
}

func TestServeTLSWithShutdown_Error(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuf, nil))
	ctx := ctxlog.WithLogger(context.Background(), logger)
	ctx, cancel := context.WithCancel(ctx)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	// Close listener immediately to cause srv.ServeTLS to fail.
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}

	serverTLSConf, _ := newSelfSignedTLSConfig(t)
	srv := &http.Server{
		TLSConfig: serverTLSConf,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := webapp.ServeTLSWithShutdown(ctx, ln, srv, time.Second)
		if err != nil {
			t.Errorf("ServeTLSWithShutdown returned an unexpected error: %v", err)
		}
	}()

	// Give the server goroutine a moment to start and fail on ServeTLS.
	time.Sleep(100 * time.Millisecond)

	// Now, trigger the shutdown.
	cancel()
	wg.Wait()

	output := logBuf.String()
	if !strings.Contains(output, "serveTLS error") || !strings.Contains(output, "use of closed network connection") {
		t.Errorf("expected 'serveTLS error' and 'use of closed network connection' in log, got:\n%s", output)
	}
	if !strings.Contains(output, "shutting down server") {
		t.Errorf("log output missing 'shutting down server':\n%s", output)
	}
}

func ExampleServeWithShutdown() {
	// Create a handler for the server.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, World!")
	})

	// Use NewHTTPServer to create a listener and a server instance.
	// Using port "0" will automatically choose a free port.
	ln, srv, err := webapp.NewHTTPServer("127.0.0.1:0", handler)
	if err != nil {
		log.Fatal(err)
	}

	// Create a context that will be canceled after a short time to trigger shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	if port != "0" {
		fmt.Printf("server listening on: %s:<some-port>\n", host)
	}

	// Run the server. This function will block until the server is shut down.
	if err := webapp.ServeWithShutdown(ctx, ln, srv, 5*time.Second); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	fmt.Println("server shutdown complete")
	// Output:
	// server listening on: 127.0.0.1:<some-port>
	// server shutdown complete
}
