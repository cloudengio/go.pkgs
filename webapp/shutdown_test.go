// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp_test

import (
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

var tlsConfig *tls.Config
var tlsCert []byte
var tlsOnce sync.Once

// newSelfSignedTLSConfig creates a self-signed certificate and returns a tls.Config
// for a server, and the certificate in PEM format for a client to trust.
func newSelfSignedTLSConfigInit(t *testing.T) (*tls.Config, []byte) {
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
		MinVersion:   tls.VersionTLS13,
	}, certPEM
}

func newSelfSignedTLSConfig(t *testing.T) (*tls.Config, []byte) {
	tlsOnce.Do(func() {
		tlsConfig, tlsCert = newSelfSignedTLSConfigInit(t)
	})
	return tlsConfig, tlsCert
}

func TestServeWithShutdown(t *testing.T) {
	var logged strings.Builder
	ctx := ctxlog.WithLogger(t.Context(), slog.New(slog.NewJSONHandler(&logged, nil)))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "hello")
	})

	ln, srv, err := webapp.NewHTTPServer(ctx, "127.0.0.1:0", handler)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := webapp.ServeWithShutdown(ctx, ln, srv, time.Second); err != nil {
			t.Errorf("ServeWithShutdown returned an unexpected error: %v", err)
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

	if !strings.Contains(logged.String(), "server being shut down") {
		t.Errorf("expected log message not found: %q", logged.String())
	}

}

func TestServeTLSWithShutdown(t *testing.T) {
	ctx := t.Context()
	serverTLSConf, certPEM := newSelfSignedTLSConfig(t)
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "hello tls")
	})
	ln, srv, err := webapp.NewTLSServer(ctx, "127.0.0.1:0", handler, serverTLSConf)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := webapp.ServeTLSWithShutdown(ctx, ln, srv, time.Second); err != nil {
			t.Errorf("ServeTLSWithShutdown returned an unexpected error: %v", err)
		}
	}()

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(certPEM)
	clientTLSConf := &tls.Config{
		RootCAs:    certPool,
		MinVersion: tls.VersionTLS13,
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

}

func TestServeWithShutdown_ServerError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}

	err = webapp.ServeWithShutdown(context.Background(), ln, &http.Server{Addr: addr.String(), ReadTimeout: time.Second}, time.Second)
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), "use of closed network connection") {
		t.Errorf("error does not contain expected message: got %v", err)
	}
}

func TestServeTLSWithShutdown_ServerError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr()
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}

	serverTLSConf, _ := newSelfSignedTLSConfig(t)
	srv := &http.Server{
		Addr:        addr.String(),
		TLSConfig:   serverTLSConf,
		ReadTimeout: time.Second,
	}
	srv.TLSConfig.MinVersion = tls.VersionTLS12 // force a ciphersuite error
	srv.TLSConfig.CipherSuites = []uint16{}

	err = webapp.ServeTLSWithShutdown(context.Background(), ln, srv, time.Second)
	if err == nil {
		t.Fatal("expected an error, but got nil")
	}
	if !strings.Contains(err.Error(), "missing an HTTP/2-required ") {
		t.Errorf("error does not contain expected message: got %v", err)
	}
}

func TestServeWithShutdown_ShutdownError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// This handler will hang, preventing a graceful shutdown.
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(time.Second)
		fmt.Fprint(w, "hello")
	})

	ln, srv, err := webapp.NewHTTPServer(ctx, "127.0.0.1:0", handler)
	if err != nil {
		t.Fatal(err)
	}

	var shutdownErr error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Use a very short grace period to ensure Shutdown fails.
		shutdownErr = webapp.ServeWithShutdown(ctx, ln, srv, 10*time.Millisecond)
	}()

	// Make a request to the hanging handler.
	go func() {
		_, _ = http.Get("http://" + ln.Addr().String())
	}()

	// Give the request a moment to be accepted by the server.
	time.Sleep(50 * time.Millisecond)

	// Trigger the shutdown.
	cancel()
	wg.Wait()

	if shutdownErr == nil {
		t.Fatal("expected a shutdown error, but got nil")
	}
	if !strings.Contains(shutdownErr.Error(), "shutdown failed") {
		t.Errorf("error does not contain expected 'shutdown failed' message: got %v", shutdownErr)
	}
}

func ExampleServeWithShutdown() {
	ctx := context.Background()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "Hello, World!")
	})

	ln, srv, err := webapp.NewHTTPServer(ctx, "127.0.0.1:0", handler)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	host, port, _ := net.SplitHostPort(ln.Addr().String())
	if port != "0" {
		fmt.Printf("server listening on: %s:<some-port>\n", host)
	}

	if err := webapp.ServeWithShutdown(ctx, ln, srv, 5*time.Second); err != nil {
		log.Printf("server shutdown error: %v", err)
	}
	fmt.Println("server shutdown complete")
	// Output:
	// server listening on: 127.0.0.1:<some-port>
	// server shutdown complete
}

func TestSplitHostPort(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		in         string
		host, port string
	}{
		{"localhost:8080", "localhost", "8080"},
		{"127.0.0.1:80", "127.0.0.1", "80"},
		{"[::1]:8080", "::1", "8080"},
		{"[::1]", "::1", ""},
		{"localhost", "localhost", ""},
		{"127.0.0.1", "127.0.0.1", ""},
		{":8080", "", "8080"},
		{"", "", ""},
		{":", "", ""},
	}

	for i, tc := range testCases {
		h, p := webapp.SplitHostPort(tc.in)
		if got, want := h, tc.host; got != want {
			t.Errorf("%v: host: got %v, want %v", i, got, want)
		}
		if got, want := p, tc.port; got != want {
			t.Errorf("%v: port: got %v, want %v", i, got, want)
		}
	}
}

func TestParseAddrPortDefaults(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		in, defaultPort, out string
	}{
		{"", "https", ":https"},
		{"", "http", ":http"},
		{":http", "https", ":http"},
		{":https", "http", ":https"},
		{"localhost:8080", "https", "localhost:8080"},
		{"127.0.0.1:8080", "http", "127.0.0.1:8080"},
		{"[::1]:8080", "https", "[::1]:8080"},
		{"8080", "https", "8080:https"},
		{"localhost", "https", "localhost:https"},
		{"google.com", "http", "google.com:http"},
		{"[::1]", "https", "[::1]:https"},
	}

	for i, tc := range testCases {
		if got, want := webapp.ParseAddrPortDefaults(tc.in, tc.defaultPort), tc.out; got != want {
			t.Errorf("%v: got %v, want %v", i, got, want)
		}
	}
}
