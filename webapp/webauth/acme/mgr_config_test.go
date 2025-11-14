// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package acme_test

import (
	"context"
	"flag"
	"strings"
	"testing"
	"time"

	"cloudeng.io/cmdutil/flags"
	"cloudeng.io/webapp/webauth/acme"
	"golang.org/x/crypto/acme/autocert"
)

func TestFlags(t *testing.T) {
	ctx := context.Background()
	cl := acme.ServiceFlags{}
	flagSet := &flag.FlagSet{}
	err := flags.RegisterFlagsInStruct(flagSet, "subcmd", &cl, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = flagSet.Parse([]string{
		"--acme-renew-before=1h",
		"--acme-email=foo@bar"})
	if err != nil {
		t.Fatal(err)
	}

	mgr, err := acme.NewAutocertManager(autocert.DirCache(t.TempDir()), cl.AutocertConfig(), "login.domain", "allowed-domain-a", "allowed-domain-b")
	if err != nil {
		t.Fatal(err)
	}

	hostPolicy := mgr.HostPolicy
	for _, host := range []string{"login.domain", "allowed-domain-a", "allowed-domain-b"} {
		if err := hostPolicy(ctx, host); err != nil {
			t.Fatalf("unexpected error for host %v: %v", host, err)
		}
	}

	err = hostPolicy(ctx, "not-there")
	if err == nil || !strings.Contains(err.Error(), `host "not-there" not configured in HostWhitelist`) {
		t.Errorf("missing or unexpected error: %v", err)
	}

	if got, want := mgr.Client.DirectoryURL, acme.LetsEncryptStaging; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := mgr.RenewBefore, time.Hour; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

}
