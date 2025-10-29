// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pebble

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"
	"time"

	"cloudeng.io/os/executil"
)

// T manages a pebble instance for testing purposes.
type T struct {
	cmd    *exec.Cmd
	binary string
	closer io.Closer
	ch     chan []byte
}

// New creates a new Pebble instance. The supplied configFile will be used
// to configure the pebble instance. The server is not started by New.
func New(binary string) *T {
	return &T{
		binary: binary,
	}
}

// Start the pebble instance with its output forwarded to the supplied
// writer.
func (p *T) Start(ctx context.Context, dir, cfg string, forward io.WriteCloser) error {
	pebblePath, err := exec.LookPath(p.binary)
	if err != nil {
		return fmt.Errorf("failed to find pebble binary in PATH: %w", err)
	}
	p.ch = make(chan []byte, 1000)
	filter := executil.NewLineFilter(forward, p.ch)
	p.cmd = exec.CommandContext(ctx, pebblePath, "-config", cfg)
	p.cmd.Dir = dir
	p.cmd.Stdout = filter
	p.cmd.Stderr = filter
	p.closer = filter
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start pebble: %w", err)
	}
	return nil
}

var (
	issuedRE    = regexp.MustCompile(`Issued certificate serial ([a-f0-9]+) for order`)
	acmeReadyRE = regexp.MustCompile(`ACME directory available at:`)
	mgmtReadyRE = regexp.MustCompile(`Root CA certificate available at:`)
)

func (p *T) WaitForReady(ctx context.Context) error {
	seen := 0
	for {
		select {
		case line := <-p.ch:
			if acmeReadyRE.Match(line) {
				seen++
			}
			if mgmtReadyRE.Match(line) {
				seen++
			}
			if seen == 2 {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// WaitForIssuedCertificateSerial waits until a certificate is issued
// and returns its serial number.
func (p *T) WaitForIssuedCertificateSerial(ctx context.Context) (string, error) {
	for {
		select {
		case line := <-p.ch:
			matches := issuedRE.FindSubmatch(line)
			if matches != nil {
				return string(matches[1]), nil
			}
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

// PID returns the process ID of the pebble instance.
func (p *T) PID() int {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

// EnsureStopped ensures that the pebble instance is stopped.
func (p *T) EnsureStopped(ctx context.Context, waitFor time.Duration) error {
	return executil.SignalAndWait(ctx, waitFor, p.cmd, os.Interrupt, syscall.SIGINT, syscall.SIGKILL)
}

// Config represents the configuration for a pebble instance
// that's relevant to using it for testing clients.
type Config struct {
	Address           string
	ManagementAddress string
	HTTPPort          int
	TLSPort           int
	Certificate       []byte
	CertificateFile   string
	CAFile            string
	TestCertBase      string
	RootCertURL       string

	originalConfig map[string]map[string]any
	pebbleCA       *x509.CertPool
	serverRoots    *x509.CertPool
}

var parsedConfig = map[string]map[string]any{}

func init() {
	if err := json.Unmarshal([]byte(pebbleConfig), &parsedConfig); err != nil {
		panic(fmt.Errorf("failed to parse pebble config: %w", err))
	}
}

const pebbleConfig = `
{
    "pebble": {
        "listenAddress": "0.0.0.0:14000",
        "managementListenAddress": "0.0.0.0:15000",
        "certificate": "test/certs/localhost/cert.pem",
        "privateKey": "test/certs/localhost/key.pem",
        "httpPort": 5002,
        "tlsPort": 5001,
        "ocspResponderURL": "",
        "externalAccountBindingRequired": false,
        "domainBlocklist": [
            "blocked-domain.example"
        ],
        "retryAfter": {
            "authz": 3,
            "order": 5
        },
        "profiles": {
            "default": {
                "description": "The profile you know and love",
                "validityPeriod": 7776000
            },
            "shortlived": {
                "description": "A short-lived cert profile, without actual enforcement",
                "validityPeriod": 518400
            }
        }
    }
}`

// NewConfig creates a new Config instance with
// default values.
func NewConfig() Config {
	var cfg Config
	cfg.originalConfig = parsedConfig
	cfg.HTTPPort = 5002
	cfg.TLSPort = 5001
	cfg.TestCertBase = filepath.Join("test", "certs")
	cfg.Address = "localhost:14000"
	cfg.ManagementAddress = "localhost:15000"
	u := url.URL{
		Scheme: "https",
		Host:   cfg.ManagementAddress,
		Path:   "/roots/0",
	}
	cfg.RootCertURL = u.String()
	return cfg
}

func deepCopy(m map[string]map[string]any) (map[string]map[string]any, error) {
	cfgData, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal original pebble config: %v", err)
	}
	var ncfg map[string]map[string]any
	if err := json.Unmarshal(cfgData, &ncfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal original pebble config: %v", err)
	}
	return ncfg, nil
}

// CreateCertsAndUpdateConfig uses minica to create a self-signed certificate for
// use with the pebble instance. The generated certificate and key are placed in outputDir.
// It returns the path to the possibly unpdated configuration file to be used when starting
// pebble.
// Use minica to create a self-signed certificate for the domain as per:
//
//		  minica -ca-cert pebble.minica.pem \
//	          -ca-key pebble.minica.key.pem \
//	          -domains localhost,pebble \
//	          -ip-addresses 127.0.0.1
func (pc *Config) CreateCertsAndUpdateConfig(ctx context.Context, outputDir string) (string, error) {
	certDir := filepath.Join(outputDir, pc.TestCertBase)
	if err := os.MkdirAll(certDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cert dir: %v", err)
	}

	minicaPath, err := exec.LookPath("minica")
	if err != nil {
		return "", fmt.Errorf("failed to find minica binary in PATH: %w", err)
	}

	cmd := exec.CommandContext(ctx, minicaPath,
		"-ca-cert", "pebble.minica.pem",
		"-ca-key", "pebble.minica.key.pem",
		"-domains", "localhost,pebble",
		"-ip-addresses", "127.0.0.1")
	cmd.Dir = certDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to run minica: %v: %s", err, output)
	}

	ncfg, err := deepCopy(pc.originalConfig)
	if err != nil {
		return "", fmt.Errorf("failed to deep copy original pebble config: %v", err)
	}
	ncfg["pebble"]["certificate"] = filepath.Join(pc.TestCertBase, "localhost", "cert.pem")
	ncfg["pebble"]["privateKey"] = filepath.Join(pc.TestCertBase, "localhost", "key.pem")
	cfgData, err := json.MarshalIndent(ncfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal updated pebble config: %v", err)
	}
	cfgFile := filepath.Join(outputDir, "pebble-config.json")
	if err := os.WriteFile(cfgFile, cfgData, 0600); err != nil {
		return "", fmt.Errorf("failed to write updated pebble config to %q: %v", cfgFile, err)
	}

	sysPool, err := x509.SystemCertPool()
	if err != nil {
		return "", fmt.Errorf("failed to load system cert pool: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(outputDir, pc.TestCertBase, "pebble.minica.pem"))
	if err != nil {
		return "", fmt.Errorf("failed to read pebble minica cert: %v", err)
	}
	if !sysPool.AppendCertsFromPEM(data) {
		return "", fmt.Errorf("failed to append pebble minica cert to cert pool")
	}

	pc.pebbleCA = sysPool
	pc.CertificateFile = filepath.Join(pc.TestCertBase, "localhost", "cert.pem")
	pc.CAFile = filepath.Join(pc.TestCertBase, "pebble.minica.pem")

	return cfgFile, nil
}

// GetIssuingCert retrieves the pebble certificate, including intermediates,
// used to sign issued certificates.
func (pc Config) GetIssuingCert(ctx context.Context) ([]byte, error) {
	u := url.URL{
		Scheme: "https",
		Host:   pc.ManagementAddress,
		Path:   "/roots/0",
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    pc.pebbleCA,
				MinVersion: tls.VersionTLS12,
			},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// CA returns a CertPool containing the root pebble CA certificate.
// Use it when configuring clients to connect to the pebble instance.
func (pc Config) CA() *x509.CertPool {
	return pc.pebbleCA
}

// IssuingCA returns a CertPool containing the issuing CA certificate
// used by pebble to sign issued certificates.
func (pc Config) GetIssuingCA(ctx context.Context) (*x509.CertPool, error) {
	data, err := pc.GetIssuingCert(ctx)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(data) {
		return nil, fmt.Errorf("failed to append issuing cert to pool")
	}
	return pool, nil
}
