// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"

	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/errors"
	"cloudeng.io/webapp"
	"cloudeng.io/webapp/webauth/acme"
	"golang.org/x/crypto/acme/autocert"
)

var cmdSet *subcmd.CommandSet

type certManagerFlags struct {
	acme.CertFlags
	webapp.TLSCertStoreFlags
}

type testRedirectFlags struct {
	webapp.HTTPServerFlags
}

func init() {
	webapp.RegisterCertStoreFactory(acme.AutoCertDiskStore)
	webapp.RegisterCertStoreFactory(acme.AutoCertNullStore)
}

func init() {
	getLocalCertCmd := subcmd.NewCommand("cert-manager",
		subcmd.MustRegisterFlagStruct(&certManagerFlags{}, nil, nil),
		getLocalCert, subcmd.ExactlyNumArguments(0))
	getLocalCertCmd.Document(`manage obtaining and renewing tls certificate`)
	cmdSet = subcmd.NewCommandSet(getLocalCertCmd)

	testRedirectCmd := subcmd.NewCommand("redirect-test",
		subcmd.MustRegisterFlagStruct(&testRedirectFlags{}, nil, nil),
		testACMERedirect, subcmd.ExactlyNumArguments(0))
	testRedirectCmd.Document(`test redirecting acme http-01 challenges back to a central server that implements the acme client.`)

	cmdSet = subcmd.NewCommandSet(getLocalCertCmd, testRedirectCmd)
	cmdSet.Document(`manage ACME issued TLS certificates`)
}

func main() {
	cmdSet.MustDispatch(context.Background())
}

func getLocalCert(ctx context.Context, values interface{}, args []string) error {
	ctx, done := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer done()
	cl := values.(*certManagerFlags)

	var cache autocert.Cache
	switch {
	case cl.CertStoreType == acme.AutoCertDiskStore.Type():
		cache = acme.NewDirCache(cl.CertStore, false)
	case cl.CertStoreType == acme.AutoCertNullStore.Type():
		cache = acme.NewNullCache()
	default:
		return fmt.Errorf("unsupported cert store type: %v", cl.CertStoreType)
	}

	mgr, err := acme.NewManagerFromFlags(ctx, cache, cl.CertFlags)
	if err != nil {
		return err
	}

	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	port80, port80Srv, err := webapp.NewHTTPServer(":80", mgr.HTTPHandler(fallback))
	if err != nil {
		return err
	}

	var stopped sync.WaitGroup
	var errs errors.M
	stopped.Add(3)
	go func() {
		err := webapp.ServeWithShutdown(ctx, port80, port80Srv, time.Minute)
		errs.Append(err)
		stopped.Done()
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "acme only please\n")
	})

	cfg := mgr.TLSConfig()
	port443Srv := &http.Server{
		Addr:      ":https",
		Handler:   mux,
		TLSConfig: cfg,
	}

	go func() {
		err = webapp.ServeWithShutdown(ctx, mgr.Listener(), port443Srv, time.Minute)
		errs.Append(err)
		stopped.Done()
	}()

	// issue get requets to initialize all certificates.
	refreshInterval := cl.RenewBefore / 10
	if refreshInterval < (time.Hour * 3) {
		refreshInterval = time.Hour * 3
	}
	log.Printf("certificate refresh interval is %v", refreshInterval)

	waitForServers(ctx)
	go func() {
		refreshCertificates(ctx, time.Hour*6, cl.AcmeClientHost, cl.Hosts.Values, cl.TestingCAPem)
		stopped.Done()
	}()

	stopped.Wait()
	return errs.Err()
}

func waitForServers(ctx context.Context) {
	for {
		time.Sleep(time.Second)
		_, err := net.DialTimeout("tcp", "127.0.0.1:80", time.Second)
		if err != nil {
			continue
		}
		_, err = net.DialTimeout("tcp", "127.0.0.1:443", time.Second)
		if err == nil {
			break
		}
	}
}

func refreshCertificates(ctx context.Context, interval time.Duration, acmeClientHost string, hosts []string, CAPemFile string) {
	rt := &http.Transport{
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	rt.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, "tcp", acmeClientHost+":443")
	}
	if len(CAPemFile) > 0 {
		certPool, err := webapp.CertPoolForTesting(CAPemFile)
		if err != nil {
			log.Printf("Failed to obtain cert pool containing %v", CAPemFile)
		} else {
			rt.TLSClientConfig = &tls.Config{
				RootCAs: certPool,
			}
		}
	}
	client := &http.Client{Transport: rt}
	for {
		for _, host := range hosts {
			u := url.URL{
				Scheme: "https",
				Host:   host,
				Path:   "/",
			}
			resp, err := client.Get(u.String())
			if err == nil {
				log.Printf("%v: %v\n", u.String(), resp.StatusCode)
			} else {
				log.Printf("%v: error %v\n", u.String(), err)
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

func testACMERedirect(ctx context.Context, values interface{}, args []string) error {
	ctx, done := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer done()
	cl := values.(*testRedirectFlags)

	if len(cl.AcmeRedirectTarget) == 0 {
		return fmt.Errorf("must specific a target for the acme client")
	}

	if err := webapp.RedirectPort80(ctx, ":443", cl.AcmeRedirectTarget); err != nil {
		return err
	}

	cfg, err := webapp.TLSConfigFromFlags(ctx, cl.HTTPServerFlags)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("hello\n")
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
