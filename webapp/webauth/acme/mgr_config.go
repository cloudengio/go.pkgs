// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package acme provides support for working with ACNE service providers
// such as letsencrypt.org.
package acme

import (
	"fmt"
	"net/url"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

const (
	// LetsEncryptStaging is the URL for the letsencrypt.org staging service
	// and is used as the default by this package.
	LetsEncryptStaging = "https://acme-staging-v02.api.letsencrypt.org/directory"
	// LetsEncryptProduction is the URL for the letsencrypt.org production service.
	LetsEncryptProduction = acme.LetsEncryptURL
)

// ServiceFlags represents the flags required to configure an ACME client
// instance for managing TLS certificates for hosts/domains using the
// acme http-01 challenge. Note that wildcard domains are not supported
// by this challenge.
// The currently supported/tested acme service providers are letsencrypt
// staging and production via the values 'letsencrypt-staging' and
// 'letsencrypt' for the --acme-service flag; however any URL can be specified
// via this flag, in particular to use pebble for testing set this to the URL
// of the local pebble instance and also set the --acme-testing-ca
// flag to point to the pebble CA certificate pem file.
type ServiceFlags struct {
	Provider    string        `subcmd:"acme-service,letsencrypt-staging,'the acme service to use, specify letsencrypt or letsencrypt-staging or a url'"`
	RenewBefore time.Duration `subcmd:"acme-renew-before,720h,how early certificates should be renewed before they expire."`
	Email       string        `subcmd:"acme-email,,email to contact for information on the domain"`
	UserAgent   string        `subcmd:"acme-user-agent,cloudeng.io/webapp/webauth/acme,'user agent to use when connecting to the acme service'"`
}

// AutocertConfig converts the flag values to a AutocertConfig instance.
func (f ServiceFlags) AutocertConfig() AutocertConfig {
	return AutocertConfig{
		Provider:    f.Provider,
		RenewBefore: f.RenewBefore,
		Email:       f.Email,
		UserAgent:   f.UserAgent,
	}
}

// AutocertConfig represents the configuration required to create an
// autocert.Manager.
type AutocertConfig struct {
	// Contact email for the ACME account, note, changing this may create
	// a new account with the ACME provider. The key associated with an account
	// is required for revoking certificates issued using that account.
	Email       string        `yaml:"email"`
	UserAgent   string        `yaml:"user_agent"`    // User agent to use when connecting to the ACME service.
	Provider    string        `yaml:"acme_provider"` // ACME service provider URL or 'letsencrypt' or 'letsencrypt-staging'.
	RenewBefore time.Duration `yaml:"renew_before"`  // How early certificates should be renewed before they expire.
}

func (ac AutocertConfig) DirectoryURL() string {
	switch ac.Provider {
	case "letsencrypt":
		return LetsEncryptProduction
	case "letsencrypt-staging":
		return LetsEncryptStaging
	default:
		if len(ac.Provider) == 0 {
			return LetsEncryptStaging
		}
		return ac.Provider
	}
}

// NewAutocertManager creates a new autocert.Manager from the supplied config.
// Any supplied hosts specify the allowed hosts for the manager, ie. those
// for which it will obtain/renew certificates.
func NewAutocertManager(cache autocert.Cache, cl AutocertConfig, allowedHosts ...string) (*autocert.Manager, error) {
	if cache == nil {
		return nil, fmt.Errorf("no cache provided")
	}
	hostPolicy := autocert.HostWhitelist(allowedHosts...)

	provider := cl.Provider
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
		UserAgent:    cl.UserAgent,
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
