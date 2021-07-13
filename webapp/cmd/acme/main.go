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
	getLocalCertCmd.Document(`manage obtaining and renewing tls certificates`)
	cmdSet = subcmd.NewCommandSet(getLocalCertCmd)

	testRedirectCmd := subcmd.NewCommand("redirect-test",
		subcmd.MustRegisterFlagStruct(&testRedirectFlags{}, nil, nil),
		testACMERedirect, subcmd.ExactlyNumArguments(0))
	testRedirectCmd.Document(`test redirecting acme http-01 challenges back to a central server that implements the acme client.`)

	cmdSet = subcmd.NewCommandSet(getLocalCertCmd, testRedirectCmd)
	cmdSet.Document(`manage ACME issued TLS certificates

This command forms the basis of managing TLS certificates for
multiple host domains. The configuration relies on running a dedicated
acme client host that is responsible for interacting with an acme service
for obtaining certificates. It will refresh those certificates as they near
expiration. However, since this server is dedicated to managing
certificates it does not support any other services and consequently all
other services do not implement the acme protocol. Rather, these other
services will redirect any http-01 acme challenges back to this dedicated
acme service. This comannd implements two sub commands: 'cert-manager' which
is the dedicated acme manager and 'redirect-test' which illustrates how
other services should redirect back to the host running the 'cert-manager'.

Certificates obtained by the of cert-manager must be distributed to all other
services that serve the hosts for which the certificates were obtained. This
can be achieved by storing the certificates in a shared store accessible to all
services, or by simply copying the certificates. The former is preferred.

A typical configuration, for domain an.example, could be:

  - run cert-manager on host certs.an.example. Port 80 must be accessible to
    the internet. It could be configured with www.an.example and an.example
	as the allowed hosts/domains for which it will manage certificates.
  - all instances of services that run on www.an.example an an.example must
    implement the redirect to certs.an.example as implemented by the
	redirect test.
  - the dns entries for an.example and www.an.example need not include the
    IP address of certs.an.example.
  - cert-manager will periodically issue http GETS against
    https://www.an.example and https://an.example that are directed to itself
    (bypassing DNS) to initiate the refresh process. Note that the same
	effect can be achieved using curl's resolve option - for example:

	   curl --cacert letsencrypt-stg-root-x1.pem --resolve an.exmaple:443:<ip-address-of-cert-manager-host> https://an.example

`)
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

func refreshCertificates(ctx context.Context, interval time.Duration, acmeClientHost string, hosts []string, rootCAPemFile string) {
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
	if len(rootCAPemFile) > 0 {
		certPool, err := webapp.CertPoolForTesting(rootCAPemFile)
		if err != nil {
			log.Printf("Failed to obtain cert pool containing %v", rootCAPemFile)
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
