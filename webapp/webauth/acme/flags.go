// Package acme provides support for working with acme/letsencrypt
// providers.
package acme

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"cloudeng.io/cmdutil/flags"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

const (
	// LetsEncryptStaging is the URL for letsencrypt.org's staging service
	// and is used as the default by this package.
	LetsEncryptStaging = "https://acme-staging-v02.api.letsencrypt.org/directory"
	// LetsEncryptProduction is the URL for letsencrypt.org's production service.
	LetsEncryptProduction = acme.LetsEncryptURL
)

// CertFlags represents the flags required to configure an autocert.Manager
// isntance for managing TLS certificates for hosts/domains using the
// acme http-01 challenge. Note that wildcard domains are not supported
// by this challenge.
// The currently supported/tested acme service providers are letsencrypt
// staging and production via the values 'letsencrypt-staging' and
// 'letsencrypt' for the --acme-service flag; however any URL can be specified
// via this flag.
type CertFlags struct {
	AcmeClientHost string          `subcmd:"acme-client-host,,'host running the acme client responsible for refreshing certificates, https requests to this host for one of the certificate hosts will result in the certificate for the certificate host being refreshed if necessary'"`
	Hosts          flags.Repeating `subcmd:"acme-cert-host,,'host for which certs are to be obtained'"`
	AcmeProvider   string          `subcmd:"acme-service,letsencrypt-staging,'the acme service to use, specify letsencrypt or letsencrypt-staging or a url'"`
	RenewBefore    time.Duration   `subcmd:"acme-renew-before,720h,how early certificates should be renewed before they expire."`
	Email          string          `subcmd:"acme-email,,email to contact for information on the domain"`
	TestingCAPem   string          `subcmd:"acme-testing-ca,,'pem file containing a CA to be trusted for testing purposes only, for example, when using letsencrypt\\'s staging service'"`
}

// NewManagerFromFlags creates a new autocert.Manager from the flag values.
// The cache may be not be nil.
func NewManagerFromFlags(ctx context.Context, cache autocert.Cache, cl CertFlags) (*autocert.Manager, error) {
	if cache == nil {
		return nil, fmt.Errorf("no cache provided")
	}
	if len(cl.AcmeClientHost) == 0 {
		return nil, fmt.Errorf("must specify a value for acme-client-host")
	}
	allowed := []string{cl.AcmeClientHost}

	hosts := cl.Hosts.Values
	if len(hosts) == 0 {
		return nil, fmt.Errorf("must specify at least one host")
	}
	allowed = append(allowed, hosts...)
	hostPolicy := autocert.HostWhitelist(allowed...)

	provider := cl.AcmeProvider
	switch provider {
	case "letsencrypt":
		provider = LetsEncryptProduction
	case "letsencrypt-staging":
		provider = LetsEncryptStaging
	default:
		if len(provider) == 0 {
			provider = LetsEncryptStaging
		} else if _, err := url.Parse(provider); err != nil {
			return nil, fmt.Errorf("invalid url: %v: %v", provider, err)
		}
	}
	client := &acme.Client{
		DirectoryURL: provider,
		UserAgent:    "cloudeng.io/webapp/webauth/acme",
	}
	mgr := &autocert.Manager{
		Prompt:      acme.AcceptTOS,
		Cache:       cache,
		Client:      client,
		Email:       cl.Email,
		HostPolicy:  hostPolicy,
		RenewBefore: cl.RenewBefore,
	}
	return mgr, nil
}
