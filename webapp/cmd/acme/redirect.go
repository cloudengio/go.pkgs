// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"cloudeng.io/aws/awscertstore"
	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/webapp"
)

type testRedirectFlags struct {
	webapp.HTTPServerFlags
	awsconfig.AWSFlags
}

func redirectCmd() *subcmd.Command {
	testRedirectCmd := subcmd.NewCommand("redirect-test",
		subcmd.MustRegisterFlagStruct(&testRedirectFlags{}, nil, nil),
		testACMERedirect, subcmd.ExactlyNumArguments(0))
	testRedirectCmd.Document(`test redirecting acme http-01 challenges back to a central server that implements the acme client.`)
	return testRedirectCmd
}

func testACMERedirect(ctx context.Context, values interface{}, _ []string) error {
	ctx, done := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer done()
	cl := values.(*testRedirectFlags)

	if len(cl.AcmeRedirectTarget) == 0 {
		return fmt.Errorf("must specific a target for the acme client")
	}

	if err := webapp.RedirectPort80(ctx, ":443", cl.AcmeRedirectTarget); err != nil {
		return err
	}

	storeOpts := []interface{}{}
	if cl.AWS {
		cfg, err := awsconfig.Load(ctx)
		if err != nil {
			return err
		}
		storeOpts = append(storeOpts, awscertstore.WithAWSConfig(cfg))
	}

	cfg, err := webapp.TLSConfigFromFlags(ctx, cl.HTTPServerFlags, storeOpts...)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "hello\n")
	})
	ln, srv, err := webapp.NewTLSServer(cl.Address, mux, cfg)
	if err != nil {
		return err
	}
	fmt.Printf("listening on: %v\n", ln.Addr())
	srv.TLSConfig = cfg
	return webapp.ServeTLSWithShutdown(ctx, ln, srv, time.Minute)
}
