// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package devtest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"syscall"

	"cloudeng.io/os/executil"
)

// Pebble manages a pebble instance for testing purposes.
type Pebble struct {
	cmd    *exec.Cmd
	binary string
	closer io.Closer
	ch     chan []byte
}

// NewPebble creates a new Pebble instance. The supplied configFile will be used
// to configure the pebble instance. The server is not started by NewPebble.
func NewPebble(binary string) *Pebble {
	return &Pebble{
		binary: binary,
	}
}

// Start the pebble instance with its output forwarded to the supplied
// writer.
func (p *Pebble) Start(ctx context.Context, dir, cfg string, forward io.WriteCloser) error {
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

func (p *Pebble) WaitForReady(ctx context.Context) error {
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
func (p *Pebble) WaitForIssuedCertificateSerial(ctx context.Context) (string, error) {
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
func (p *Pebble) PID() int {
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

// Stop the pebble instance.
func (p *Pebble) Stop() error {
	p.cmd.Process.Signal(syscall.SIGINT) //nolint:errcheck
	if p.cmd != nil {
		return p.cmd.Wait()
	}
	if p.closer != nil {
		if err := p.closer.Close(); err != nil {
			return err
		}
	}
	return nil
}

// PebbleConfig represents the configuration for a pebble instance
// that's relevant to using it for testing clients.
type PebbleConfig struct {
	Address           string
	ManagementAddress string
	HTTPPort          int
	TLSPort           int
	Certificate       []byte
	Key               []byte
	TestCertBase      string
	RootCertURL       string

	originalConfig map[string]map[string]any
}

/*
func asInt(key string, m map[string]any) (int, error) {
	v, ok := m[key]
	if !ok {
		return 0, fmt.Errorf("missing key %q in pebble config", key)
	}
	f, ok := v.(float64)
	if !ok {
		return 0, fmt.Errorf("invalid type for key %q in pebble config: %T", key, v)
	}
	return int(f), nil
}

func asString(key string, m map[string]any) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("missing key %q in pebble config", key)
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("invalid type for key %q in pebble config: %T", key, v)
	}
	return s, nil
}
*/

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

func NewPebbleConfig() (PebbleConfig, error) {
	var cfg PebbleConfig
	cfg.originalConfig = parsedConfig
	cfg.HTTPPort = 5002
	cfg.TLSPort = 5001
	cfg.TestCertBase = "test/certs"
	cfg.Address = "localhost:14000"
	cfg.ManagementAddress = "localhost:15000"
	u := url.URL{
		Scheme: "https",
		Host:   cfg.ManagementAddress,
		Path:   "/roots/0",
	}
	cfg.RootCertURL = u.String()
	return cfg, nil

	/*
		var errs errors.M
		tmp := parsedConfig["pebble"]

		/*
			ai := func(key string) int {
				v, err := asInt(key, tmp)
				errs.Append(err)
				return v
			}
			as := func(key string) string {
				v, err := asString(key, tmp)
				errs.Append(err)
				return v
			}

			laddr := as("listenAddress")
			maddr := as("managementListenAddress")
			cfg.HTTPPort = ai("httpPort")
			cfg.TLSPort = ai("tlsPort")
			certFile := as(string("certificate"))
			keyFile := as(string("privateKey"))
			if errs.Err() != nil {
				return cfg, errs.Err()
			}
			_, port, err := net.SplitHostPort(laddr)
			if err != nil {
				return cfg, fmt.Errorf("invalid listen address %q: %w", laddr, err)
			}
			_, mport, err := net.SplitHostPort(maddr)
			if err != nil {
				return cfg, fmt.Errorf("invalid management listen address %q: %w", maddr, err)
			}
			base, host := cfg.findBase(certFile, keyFile)
			cfg.Address = net.JoinHostPort(host, port)
			cfg.ManagementAddress = net.JoinHostPort(host, mport)

			u := url.URL{
				Scheme: "https",
				Host:   cfg.ManagementAddress,
				Path:   "/roots/0",
			}
			cfg.RootCertURL = u.String()

			cfg.TestCertBase = base
			if cfg.TestCertBase == "" {
				errs.Append(fmt.Errorf("failed to determine test cert base from %q and %q", cfg.Address, cfg.ManagementAddress))
			}
			if errs.Err() != nil {
				return cfg, errs.Err()
			}
			return cfg, nil*/

}

/*
func (pc PebbleConfig) findLeadingPath(a string) (prefix, host string) {
	parts := strings.Split(a, "/")
	if len(parts) < 2 {
		return "", ""
	}
	return path.Join(parts[0 : len(parts)-2]...), parts[len(parts)-2]
}

func (pc PebbleConfig) findBase(a, b string) (prefix, host string) {
	la, ha := pc.findLeadingPath(a)
	lb, hb := pc.findLeadingPath(b)
	if la == "" || lb == "" {
		return "", ""
	}
	if la != lb {
		return "", ""
	}
	if ha != hb {
		return "", ""
	}
	return la, ha
}
*/

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
func (pc PebbleConfig) CreateCertsAndUpdateConfig(ctx context.Context, outputDir string) (string, error) {
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
	cmd.Dir = outputDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to run minica: %v: %s", err, output)
	}

	ncfg := maps.Clone(pc.originalConfig)
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

	return cfgFile, nil
}

func (pc PebbleConfig) GetRootCert(ctx context.Context) ([]byte, error) {
	u := url.URL{
		Scheme: "https",
		Host:   pc.ManagementAddress,
		Path:   "/roots/0",
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
