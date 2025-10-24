// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/cmdutil"
	"cloudeng.io/errors"
	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/net/http/httptracing"
	"cloudeng.io/sync/errgroup"
	"cloudeng.io/webapp"
	"cloudeng.io/webapp/devtest"
	"cloudeng.io/webapp/webauth/acme"
	"cloudeng.io/webapp/webauth/acme/certcache"
	"golang.org/x/crypto/acme/autocert"
)

type certManagerFlags struct {
	acme.ServiceFlags
	TestingCAPem string `subcmd:"acme-testing-ca,,'pem file containing a CA to be trusted for testing purposes only, for example, when using letsencrypt\\'s staging service'"`
	TLSCertStoreFlags
	awsconfig.AWSFlags
	HTTPPort        int           `subcmd:"http-port,80,address to run http acme challenge server on"`
	RefreshInterval time.Duration `subcmd:"cert-refresh-interval,6h,interval between certificate refresh attempts"`
}
type certManagerCmd struct{}

func (_ certManagerCmd) manageCerts(ctx context.Context, flags any, args []string) error {
	logger := ctxlog.Logger(ctx)
	logger.Info("starting acme cert manager", "build_info", cmdutil.BuildInfoJSON())

	cl := flags.(*certManagerFlags)
	hosts := args

	cache, err := newCertStore(ctx, cl.TLSCertStoreFlags, cl.AWSFlags, false)
	if err != nil {
		return err
	}

	acmeCfg := cl.ServiceFlags.AutocertConfig()
	mgr, err := acme.NewAutocertManager(ctx, cache, acmeCfg, hosts...)
	if err != nil {
		return err
	}

	if cl.HTTPPort != 80 {
		noPort := certcache.WrapHostPolicyNoPort(mgr.HostPolicy)
		mgr.HostPolicy = func(ctx context.Context, host string) error {
			if err := noPort(ctx, host); err != nil {
				ctxlog.Logger(ctx).Info("acme host policy check failed", "host", host, "error", err)
				return err
			}
			ctxlog.Logger(ctx).Info("acme host policy check succeeded", "host", host)
			return nil
		}
	}

	if cl.TestingCAPem != "" {
		logger.Warn("acme.NewManagerFromFlags: using custom root CA pool containing", "ca", cl.TestingCAPem)
		rootCAs, err := devtest.CertPoolForTesting(cl.TestingCAPem)
		if err != nil {
			return fmt.Errorf("failed to obtain cert pool containing %v: %w", cl.TestingCAPem, err)
		}
		testingRT := &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    rootCAs,
				MinVersion: tls.VersionTLS13,
			}}
		loggingRT := httptracing.NewTracingRoundTripper(testingRT,
			httptracing.WithTracingLogger(logger),
			httptracing.WithTraceRequestBody(httptracing.JSONRequestBodyLogger),
			httptracing.WithTraceResponseBody(httptracing.JSONResponseBodyLogger),
		)
		mgr.Client.HTTPClient = &http.Client{
			Transport: loggingRT,
		}
	}

	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxlog.Logger(r.Context()).Info("http fallback handler called, rejecting request")
		w.WriteHeader(http.StatusForbidden)
	})

	port80TracingHandler := httptracing.NewTracingHandler(
		mgr.HTTPHandler(fallback),
		httptracing.WithHandlerLogger(logger.With("server", "acme-challenge-http")),
		httptracing.WithHandlerRequestBody(httptracing.JSONRequestBodyLogger),
		httptracing.WithHandlerResponseBody(httptracing.JSONHandlerResponseLogger),
	)

	httpListener, httpServer, err := webapp.NewHTTPServer(ctx, fmt.Sprintf(":%d", cl.HTTPPort), port80TracingHandler)
	if err != nil {
		return err
	}

	var stopped sync.WaitGroup
	var errs errors.M
	stopped.Add(2)
	go func() {
		err := webapp.ServeWithShutdown(ctx, httpListener, httpServer, time.Minute)
		errs.Append(err)
		stopped.Done()
	}()

	// issue get requets to initialize all certificates.
	refreshInterval := cl.RenewBefore / 10
	if refreshInterval < (time.Hour * 3) {
		refreshInterval = time.Hour * 3
	}
	logger.Info("certificate refresh interval", "interval", refreshInterval.String())

	webapp.WaitForServers(ctx, time.Second*2, httpListener.Addr().String())

	go func() {
		err := refreshCertificatesUsingHello(ctx, cl.RefreshInterval, mgr, hosts...)
		errs.Append(err)
		stopped.Done()
	}()

	stopped.Wait()
	return errs.Err()
}

func refreshCertificatesUsingHello(ctx context.Context, interval time.Duration, mgr *autocert.Manager, hosts ...string) error {
	grp := &errgroup.T{}
	for _, host := range hosts {
		h := host
		grp.Go(func() error {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				if err := refreshCertificateUsingHello(ctx, mgr, h); err != nil {
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
	return grp.Wait()
}

func refreshCertificateUsingHello(ctx context.Context, mgr *autocert.Manager, host string) error {
	hello := tls.ClientHelloInfo{
		ServerName:       host,
		CipherSuites:     webapp.PreferredCipherSuites,
		SignatureSchemes: webapp.PreferredSignatureSchemes,
	}
	ctxlog.Logger(ctx).Info("refreshing certificate using tls hello", "host", host)
	cert, err := mgr.GetCertificate(&hello)
	if err != nil {
		return err
	}
	ctxlog.Logger(ctx).Info("refreshed certificate using tls hello", "host", host, "expiry", cert.Leaf.NotAfter, "subject", cert.Leaf.Subject)
	return nil
}
