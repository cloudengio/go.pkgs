// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awsconfig_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"cloudeng.io/aws/awsconfig"
)

func TestLoad(t *testing.T) {
	ctx := context.Background()
	cfgFiles := filepath.Join("testdata", "aws.config")
	cl := awsconfig.AWSFlags{
		AWS:            true,
		AWSConfigFiles: cfgFiles,
	}
	cfg, err := awsconfig.LoadUsingFlags(ctx, cl)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.Region, "us-somewhere-2"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	_, err = cfg.Credentials.Retrieve(ctx)
	if err == nil || !strings.Contains(err.Error(), "no EC2 IMDS role found") {
		t.Fatalf("missing or unexpected error: %v", err)
	}

	cl.AWSProfile = "test"
	cfg, err = awsconfig.LoadUsingFlags(ctx, cl)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := cfg.Region, "us-somewhere-3"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	creds, err := cfg.Credentials.Retrieve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := creds.AccessKeyID, "AAAAA"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
