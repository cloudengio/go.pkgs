// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package acme

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/sync/errgroup"
	"cloudeng.io/webapp"
	"golang.org/x/crypto/acme/autocert"
)

// Client implements an ACME client that periodically refreshes
// certificates for a set of hosts using the provided autocert.Manager.
type Client struct {
	logger   *slog.Logger
	mgr      *autocert.Manager
	interval time.Duration
	hosts    []string
}

func NewClient(mgr *autocert.Manager, refreshInterval time.Duration, hosts ...string) *Client {
	return &Client{
		mgr:      mgr,
		interval: refreshInterval,
		hosts:    slices.Clone(hosts),
	}
}

func (s *Client) Start(ctx context.Context) (func() error, error) {
	refreshCtx, cancel := context.WithCancel(ctx)
	s.logger = ctxlog.Logger(ctx).With("component", "acme_client")
	errCh := make(chan error, 1)
	go s.refresh(refreshCtx, errCh)
	return func() error {
		return s.stop(cancel, errCh)
	}, nil
}

func (s *Client) stop(cancel func(), errCh <-chan error) error {
	cancel()
	s.logger.Info("stopping acme client")
	select {
	case err := <-errCh:
		if err != nil {
			s.logger.Error("acme client stopped with error", "error", err)
		} else {
			s.logger.Info("acme client stopped")
		}
		return err
	case <-time.After(5 * time.Second):
		s.logger.Warn("timeout waiting for acme server to stop")
		return fmt.Errorf("timeout waiting for acme server to stop")
	}
}

func (s *Client) refresh(ctx context.Context, errCh chan<- error) {
	grp := &errgroup.T{}
	for _, host := range s.hosts {
		h := host
		grp.Go(func() error {
			ctxlog.Logger(ctx).Info("starting certificate refresh loop", "host", h, "interval", s.interval.String())
			ticker := time.NewTicker(s.interval)
			defer ticker.Stop()
			for {
				if err := s.refreshHost(ctx, h); err != nil {
					ctxlog.Logger(ctx).Error("failed to refresh certificate using tls hello", "host", h, "error", err)
				}
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
				}
			}
		})
	}
	errCh <- grp.Wait()
}

func (s *Client) refreshHost(ctx context.Context, host string) error {
	hello := tls.ClientHelloInfo{
		ServerName:       host,
		CipherSuites:     webapp.PreferredCipherSuites,
		SignatureSchemes: webapp.PreferredSignatureSchemes,
	}
	ctxlog.Logger(ctx).Info("refreshing certificate using tls hello", "host", host)
	cert, err := s.mgr.GetCertificate(&hello)
	if err != nil {
		return err
	}
	leaf := cert.Leaf
	ctxlog.Logger(ctx).Info("refreshed certificate using tls hello", "host", host, "expiry", leaf.NotAfter, "serial", fmt.Sprintf("%0*x", len(leaf.SerialNumber.Bytes())*2, leaf.SerialNumber))
	if time.Now().After(leaf.NotAfter) {
		ctxlog.Logger(ctx).Warn("certificate has expired", "host", host, "expiry", leaf.NotAfter, "serial", fmt.Sprintf("%0*x", len(leaf.SerialNumber.Bytes())*2, leaf.SerialNumber))
	}
	return nil
}
