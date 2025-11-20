# [cloudeng.io/webapp/cmd/acme](https://pkg.go.dev/cloudeng.io/webapp/cmd/acme?tab=doc)


Usage of `acme`

    manage ACME issued TLS certificates

    This command forms the basis of managing TLS certificates for multiple host
    domains. The configuration relies on running a dedicated `acme` client host that
    is responsible for interacting with an `acme` service for obtaining certificates.
    It will refresh those certificates as they near expiration. However, since this
    server is dedicated to managing certificates it does not support any other
    services and consequently all other services do not implement the `acme` protocol.
    Rather, these other services will redirect any http-01 `acme` challenges back to
    this dedicated `acme` service. This command implements two sub commands: 'cert-manager'
    which is the dedicated `acme` manager and 'redirect-test' which illustrates how
    other services should redirect back to the host running the 'cert-manager'.

    Certificates obtained by the of cert-manager must be distributed to all other
    services that serve the hosts for which the certificates were obtained. This can
    be achieved by storing the certificates in a shared store accessible to all
    services, or by simply copying the certificates. The former is preferred and a
    shared store using AWS' secretsmanager can be used to do so as per
    cloudeng.io/aws/awscertstore.

    A typical configuration, for domain an.example, could be:

    - run cert-manager on host certs.an.example. Port 80 must be accessible to the
    internet. It could be configured with www.an.example and an.example as the allowed
    hosts/domains for which it will manage certificates. - all instances of services
    that run on www.an.example and an.example must implement the redirect to
    certs.an.example as implemented by the redirect test. - the dns entries for
    an.example and www.an.example need not include the IP address of certs.an.example.
    - cert-manager will periodically issue http GETS against https://www.an.example
    and https://an.example that are directed to itself (bypassing DNS) to initiate
    the refresh process. Note that the same effect can be achieved using curl's
    resolve option - for example:

    curl --cacert letsencrypt-stg-root-x1.pem --resolve an.exmaple:443:<ip-address-of-cert-manager-host>
    https://an.example

    This approach allows for automated management TLS certifcates for server farms
    that live behind firewalls/loadbalancers, are hosted on services such as AWS
    fargate, ECS/EKS etc with no overhead other than implementing the http-01 redirect
    and having access to the certificates.

    servers - run `acme` related servers
      certs - manage ACME issued TLS certificates

