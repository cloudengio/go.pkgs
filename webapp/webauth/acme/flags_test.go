package acme_test

import (
	"context"
	"flag"
	"strings"
	"testing"
	"time"

	"cloudeng.io/cmdutil/flags"
	"cloudeng.io/webapp/webauth/acme"
)

func TestFlags(t *testing.T) {
	ctx := context.Background()
	cl := acme.CertFlags{}
	flagSet := &flag.FlagSet{}
	err := flags.RegisterFlagsInStruct(flagSet, "subcmd", &cl, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	err = flagSet.Parse([]string{
		"--acme-client-host=login.domain",
		"--acme-cert-host=allowed-domain-a",
		"--acme-cert-host=allowed-domain-b",
		"--acme-renew-before=1h",
		"--acme-email=foo@bar"})
	if err != nil {
		t.Fatal(err)
	}

	mgr, err := acme.NewManagerFromFlags(ctx, acme.NewNullCache(), cl)
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
