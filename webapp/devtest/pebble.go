// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package devtest

import (
	"context"
	"fmt"
	"io"
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
	config string
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

// CreateCerts uses minica to create a self-signed certificate for use with
// the pebble instance. The generated certificate and key are placed in outputDir.
// It returns the path to the pebble configuration file to be used when starting
// pebble.
func (p *Pebble) CreateCerts(ctx context.Context, outputDir string) (string, error) {
	// Use minica to create a self-signed certificate for the domain as per:
	// 	  minica -ca-cert pebble.minica.pem \
	//           -ca-key pebble.minica.key.pem \
	//           -domains localhost,pebble \
	//           -ip-addresses 127.0.0.1
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
	return filepath.Join("testdata", "pebble-config.json"), nil
}

// Start the pebble instance with its output forwarded to the supplied
// writer.
func (p *Pebble) Start(ctx context.Context, cfg string, forward io.WriteCloser) error {
	pebblePath, err := exec.LookPath(p.binary)
	if err != nil {
		return fmt.Errorf("failed to find pebble binary in PATH: %w", err)
	}
	p.config = cfg
	p.ch = make(chan []byte, 1000)
	filter := executil.NewLineFilter(forward, p.ch, predefindREs...)
	p.cmd = exec.CommandContext(ctx, pebblePath, "-config", p.config)
	p.cmd.Stdout = filter
	p.cmd.Stderr = filter
	p.closer = filter
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start pebble: %w", err)
	}
	return nil
}

var (
	issuedRE     = regexp.MustCompile(`Issued certificate serial ([a-f0-9]+) for order`)
	predefindREs = []*regexp.Regexp{
		issuedRE,
	}
)

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
