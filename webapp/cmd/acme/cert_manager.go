// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/cmdutil"
	"cloudeng.io/errors"
	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/net/http/httptracing"
	"cloudeng.io/webapp"
	"cloudeng.io/webapp/webauth/acme"
	"cloudeng.io/webapp/webauth/acme/certcache"
)

type TestingCAPEMFlag struct {
	TestingCAPEM string `subcmd:"acme-testing-ca,,'pem file containing a CA to be trusted for testing purposes only, for example, when using letsencrypt\\'s staging service'"`
}

type ClientHostFlag struct {
	ClientHost string `subcmd:"acme-client-host,,'host running the acme client responsible for refreshing certificates, https requests to this host for one of the certificate hosts will result in the certificate for the certificate host being refreshed if necessary'"`
}

type AccountKeyAliasFlag struct {
	AccountKeyAlias string `subcmd:"acme-account-key-alias,acme_account.key,'the alias/name in the certificate store for the acme account private key'"`
}

type certManagerFlags struct {
	ClientHostFlag
	acme.ServiceFlags
	TestingCAPEMFlag
	TLSCertStoreFlags
	AccountKeyAliasFlag
	awsconfig.AWSFlags
	HTTPPort        int           `subcmd:"http-port,80,address to run http acme challenge server on"`
	RefreshInterval time.Duration `subcmd:"cert-refresh-interval,6h,interval between certificate refresh attempts"`
	Trace           bool          `subcmd:"trace,false,enable http tracing for acme client operations"`
}
type certManagerCmd struct{}

func (certManagerCmd) manageCerts(ctx context.Context, flags any, args []string) error {
	logger := ctxlog.Logger(ctx)
	logger.Info("starting acme cert manager", "build_info", cmdutil.BuildInfoJSON())

	cl := flags.(*certManagerFlags)
	hosts := append([]string{cl.ClientHost}, args...)

	logger.Info("acme cert manager flags", "flags", cl)

	cache, err := newCertStore(ctx, cl.TLSCertStoreFlags, cl.AWSFlags,
		certcache.WithReadonly(false),
		certcache.WithSaveAccountKey(cl.AccountKeyAlias))
	if err != nil {
		return err
	}

	acmeCfg := cl.AutocertConfig()
	mgr, err := acme.NewAutocertManager(cache, acmeCfg, hosts...)
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

	var clientOpts []webapp.HTTPClientOption
	if cl.Trace {
		clientOpts = append(clientOpts,
			webapp.WithTracingTransport(
				httptracing.WithTracingLogger(logger),
				httptracing.WithTraceRequestBody(httptracing.JSONRequestBodyLogger),
				httptracing.WithTraceResponseBody(httptracing.JSONResponseBodyLogger)),
		)
	}
	clientOpts = append(clientOpts, webapp.WithCustomCAPEMFile(cl.TestingCAPEM))

	mgr.Client.HTTPClient, err = webapp.NewHTTPClient(ctx, clientOpts...)
	if err != nil {
		return fmt.Errorf("failed to create acme manager http client: %w", err)
	}

	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctxlog.Logger(r.Context()).Info("http fallback handler called, rejecting request")
		w.WriteHeader(http.StatusForbidden)
	})

	httpHandler := mgr.HTTPHandler(fallback)
	if cl.Trace {
		httpHandler = httptracing.NewTracingHandler(
			mgr.HTTPHandler(fallback),
			httptracing.WithHandlerLogger(logger.With("server", "acme-challenge-http")),
			httptracing.WithHandlerRequestBody(httptracing.JSONRequestBodyLogger),
			httptracing.WithHandlerResponseBody(httptracing.JSONHandlerResponseLogger),
		)
	}

	httpListener, httpServer, err := webapp.NewHTTPServer(ctx, fmt.Sprintf(":%d", cl.HTTPPort), httpHandler)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		err := webapp.ServeWithShutdown(ctx, httpListener, httpServer, time.Minute)
		errCh <- err
	}()

	if err := webapp.WaitForServers(ctx, time.Second*2, httpListener.Addr().String()); err != nil {
		return fmt.Errorf("http server failed to start: %w", err)
	}

	acmeClient := acme.NewClient(mgr, cl.RefreshInterval, args...)

	stopAcmeClient, err := acmeClient.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start acme client: %w", err)
	}

	var errs errors.M
	errs.Append(<-errCh)
	errs.Append(stopAcmeClient())
	return errs.Err()
}
