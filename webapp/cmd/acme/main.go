// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"log/slog"
	"os"

	"cloudeng.io/cmdutil"
	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/logging/ctxlog"
)

const cmdSpecYAML = `name: acme
summary: manage ACME issued TLS certificates
commands:
  - name: "servers"
    summary: run acme related servers
    commands:
      - name: cert-manager
        summary: manage obtaining and renewing tls certificates using an acme service such as letsencrypt.org
        args:
          - <hosts>...  # hosts for which to manage certificates
      - name: redirect
        summary: run an http server that redirects acme http-01 challenges back to a central server that implements the acme client, as run by cert-manager for example.
  - name: certs
    summary: manage ACME issued TLS certificates
    commands:
      - name: validate-hosts
        summary: validate the certificates served for specified hosts
        args:
          - <hosts>+  # hosts to validate certificates for          
      - name: validate-pem-files
        summary: validate the pem encoded certificates served for specified hosts
        args:
          - <pem-files>...  # pem files to validate certificates for
      - name: store
        summary: store and retrieve certificates directly from a certificate store
        commands:
          - name: put
            summary: store a certificate in a cert store
          - name: get
            summary: retrieve a certificate from a cert store
      - name: revoke
        summary: revoke a certificate stored in a cert store using either the private key of the certificate or the private key of the acme account used to obtain the certificate
        args:
          - <name> # name of the certificate to revoke
`

func cli() *subcmd.CommandSetYAML {
	cmd := subcmd.MustFromYAML(cmdSpecYAML)
	certManagerCmd := certManagerCmd{}
	cmd.Set("servers", "cert-manager").MustRunner(certManagerCmd.manageCerts, &certManagerFlags{})
	redirectCmd := testRedirectCmd{}
	cmd.Set("servers", "redirect").MustRunner(redirectCmd.redirect, &testRedirectFlags{})

	certsCmd := certsCmd{}
	cmd.Set("certs", "validate-hosts").MustRunner(certsCmd.validateHostCertificatesCmd, &validateHostFlags{})
	cmd.Set("certs", "validate-pem-files").MustRunner(certsCmd.validatePEMFilesCmd, &validateFileFlags{})
	cmd.Set("certs", "store", "put").MustRunner(putCert, &putCertFlags{})
	cmd.Set("certs", "store", "get").MustRunner(getCert, &getCertFlags{})

	revokeCmd := revokeCmd{}
	cmd.Set("certs", "revoke").MustRunner(revokeCmd.revokeUsingKey, &revokeFlags{})

	cmd.Document(`manage ACME issued TLS certificates

This command forms the basis of managing TLS certificates for
multiple host domains. The configuration relies on running a dedicated
acme client host that is responsible for interacting with an acme service
for obtaining certificates. It will refresh those certificates as they near
expiration. However, since this server is dedicated to managing
certificates it does not support any other services and consequently all
other services do not implement the acme protocol. Rather, these other
services will redirect any http-01 acme challenges back to this dedicated
acme service. This command implements two sub commands: 'cert-manager' which
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
  - all instances of services that run on www.an.example and an.example must
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
	return cmd
}

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stderr,
		&slog.HandlerOptions{Level: slog.LevelInfo, AddSource: true}))
	ctx = ctxlog.WithLogger(ctx, logger)
	ctx, cancel := cmdutil.HandleInterrupt(ctx)
	defer cancel(nil)
	if err := cli().Dispatch(ctx); err != nil {
		if context.Cause(ctx) == cmdutil.ErrInterrupt {
			cmdutil.Exit("%v", cmdutil.ErrInterrupt)
		}
		cmdutil.Exit("%v", err)
	}
}
