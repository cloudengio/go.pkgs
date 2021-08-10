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
	"strings"
	"sync"
	"time"

	"cloudeng.io/aws/awscertstore"
	"cloudeng.io/aws/awsconfig"
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
	awsconfig.AWSFlags
}

func init() {
	// Register the available certificate stores.
	webapp.RegisterCertStoreFactory(acme.AutoCertDiskStore)
	webapp.RegisterCertStoreFactory(acme.AutoCertNullStore)
	webapp.RegisterCertStoreFactory(awscertstore.AutoCertStore)
}

func init() {
	certManagerCmd := subcmd.NewCommand("cert-manager",
		subcmd.MustRegisterFlagStruct(&certManagerFlags{}, nil, nil),
		manageCerts, subcmd.ExactlyNumArguments(0))
	certManagerCmd.Document(`manage obtaining and renewing tls certificates using an acme service such as letsencrypt.org.`)
	cmdSet = subcmd.NewCommandSet(certManagerCmd)

	cmdSet = subcmd.NewCommandSet(
		certManagerCmd,
		redirectCmd(),
		certSubCmd(),
		validateCmd())
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
services, or by simply copying the certificates. The former is preferred and
a shared store using AWS' secretsmanager can be used to do so as per
cloudeng.io/aws/awscertstore.

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

This approach allows for automated management TLS certifcates for server
farms that live behind firewalls/loadbalancers, are hosted on services
such as AWS fargate, ECS/EKS etc with no overhead other than implementing
the http-01 redirect and having access to the certificates.
`)
}

func main() {
	cmdSet.MustDispatch(context.Background())
}

func newAutoCertCacheFromFlags(ctx context.Context, cl webapp.TLSCertStoreFlags, awscl awsconfig.AWSFlags) (autocert.Cache, error) {
	opts := []awscertstore.AWSCacheOption{}
	if awscl.AWS {
		awscfg, err := awsconfig.LoadUsingFlags(ctx, awscl)
		if err != nil {
			return nil, err
		}
		opts = append(opts, awscertstore.WithAWSConfig(awscfg))
	}
	if cl.ListStoreTypes {
		return nil, fmt.Errorf(strings.Join(webapp.RegisteredCertStores(), "\n"))
	}
	var cache autocert.Cache
	switch {
	case cl.CertStoreType == acme.AutoCertDiskStore.Type():
		cache = acme.NewDirCache(cl.CertStore, false)
	case cl.CertStoreType == acme.AutoCertNullStore.Type():
		cache = acme.NewNullCache()
	case cl.CertStoreType == awscertstore.AutoCertStore.Type():
		cache = awscertstore.NewHybridCache(cl.CertStore, opts...)
	default:
		return nil, fmt.Errorf("unsupported cert store type: %v", cl.CertStoreType)
	}
	return cache, nil
}

func manageCerts(ctx context.Context, values interface{}, args []string) error {
	ctx, done := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer done()
	cl := values.(*certManagerFlags)

	cache, err := newAutoCertCacheFromFlags(ctx, cl.TLSCertStoreFlags, cl.AWSFlags)
	if err != nil {
		return err
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
		MaxIdleConns:          10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	dialer := &net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 60 * time.Second,
	}
	rt.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, "tcp", acmeClientHost+":443")
	}
	customCertPool(rt, rootCAPemFile)
	client := &http.Client{Transport: rt}
	for {
		for _, host := range hosts {
			pingHost(client, host)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
		}
	}
}

func customCertPool(ht *http.Transport, rootCAPemFile string) {
	if len(rootCAPemFile) > 0 {
		ht.TLSClientConfig = customTLSConfig(rootCAPemFile)
	}
}

func customTLSConfig(rootCAPemFile string) *tls.Config {
	if len(rootCAPemFile) > 0 {
		certPool, err := webapp.CertPoolForTesting(rootCAPemFile)
		if err != nil {
			log.Printf("Failed to obtain cert pool containing %v", rootCAPemFile)
		} else {
			return &tls.Config{RootCAs: certPool}
		}
	}
	return &tls.Config{}
}

func pingHost(client *http.Client, hostName string) error {
	u := url.URL{
		Scheme: "https",
		Host:   hostName,
		Path:   "/",
	}
	resp, err := client.Get(u.String())
	if err == nil {
		log.Printf("%v: %v\n", u.String(), resp.StatusCode)
	} else {
		log.Printf("%v: error %v\n", u.String(), err)
	}
	return err
}
