# [cloudeng.io/webapp/cmd/acme](https://pkg.go.dev/cloudeng.io/webapp/cmd/acme?tab=doc)
[![CircleCI](https://circleci.com/gh/cloudengio/go.gotools.svg?style=svg)](https://circleci.com/gh/cloudengio/go.gotools) [![Go Report Card](https://goreportcard.com/badge/cloudeng.io/webapp/cmd/acme)](https://goreportcard.com/report/cloudeng.io/webapp/cmd/acme)


Usage of `acme`

    manage ACME issued TLS certificates

    This command forms the basis of managing TLS certificates formultiple host
    domains. The configuration relies on running a dedicated`acme` client host that
    is responsible for interacting with an `acme` servicefor obtaining certificates.
    It will refresh those certificates as they nearexpiration. However, since this
    server is dedicated to managingcertificates it does not support any other
    services and consequently allother services do not implement the `acme` protocol.
    Rather, these otherservices will redirect any http-01 `acme` challenges back to
    this dedicated`acme` service. This comannd implements two sub commands: 'cert-manager'
    whichis the dedicated `acme` manager and 'redirect-test' which illustrates how
    other services should redirect back to the host running the 'cert-manager'.

    Certificates obtained by the of cert-manager must be distributed to all other
    services that serve the hosts for which the certificates were obtained. Thiscan
    be achieved by storing the certificates in a shared store accessible to all
    services, or by simply copying the certificates. The former is preferred anda
    shared stored using AWS' secretsmanager can be used to do so.

    A typical configuration, for domain an.example, could be:

    - run cert-manager on host certs.an.example. Port 80 must be accessible tothe
    internet. It could be configured with www.an.example and an.exampleas the allowed
    hosts/domains for which it will manage certificates.- all instances of services
    that run on www.an.example an an.example mustimplement the redirect to
    certs.an.example as implemented by theredirect test.- the dns entries for
    an.example and www.an.example need not include theIP address of certs.an.example.
    - cert-manager will periodically issue http GETS againsthttps://www.an.example
    and https://an.example that are directed to itself(bypassing DNS) to initiate
    the refresh process. Note that the sameeffect can be achieved using curl's
    resolve option - for example:

    curl --cacert letsencrypt-stg-root-x1.pem --resolve an.exmaple:443:<ip-address-of-cert-manager-host>
    https://an.example

    This approach allows for automated management TLS certifcates for serverfarms
    that live behind firewalls/loadbalancers, are hosted on servicessuch as AWS
    fargate, ECS/EKS etc with no overhead other than implementingthe http-01 redirect
    and having access to the certificates.

     cert-manager - manage obtaining and renewing tls certificates using an `acme` service such as letsencrypt.org.
    redirect-test - test redirecting `acme` http-01 challenges back to a central server that implements the `acme` client.
       cert-store - store and retrieve certificates directly from a certificate store.

flag: help requested

