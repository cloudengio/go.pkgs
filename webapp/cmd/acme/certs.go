// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/aws/awssecretsfs"
	"cloudeng.io/webapp/webauth/acme/certcache"
)

// TLSCertStoreFlags defines commonly used flags for specifying a TLS/SSL
// certificate store. This is generally used in conjunction with
// TLSConfigFromFlags for apps that simply want to use stored certificates.
// Apps that manage/obtain/renew certificates may use them directly.
type TLSCertStoreFlags struct {
	UseAWSSecretsManager bool   `subcmd:"aws-secrets,false,'use AWS Secrets Manager as the backend for the certificate store'"`
	LocalCacheDir        string `subcmd:"local-cache-dir,,'if set use a local directory as a cache layer in front of the certificate store'"`
}

type putCertFlags struct {
	TLSCertStoreFlags
	awsconfig.AWSFlags
}

type getCertFlags struct {
	TLSCertStoreFlags
	awsconfig.AWSFlags
}

type certsCmd struct{}

func newCertStore(ctx context.Context, cl TLSCertStoreFlags, awscl awsconfig.AWSFlags, opts ...certcache.Option) (*certcache.CachingStore, error) {
	if cl.UseAWSSecretsManager && !awscl.AWS {
		return nil, fmt.Errorf("aws-secrets-manager flag requires aws configuration to be enabled")
	}
	if cl.LocalCacheDir == "" {
		return nil, fmt.Errorf("local-cache-dir must be specified")
	}
	if !cl.UseAWSSecretsManager || !awscl.AWS {
		lb, err := certcache.NewLocalStore(filepath.Join(cl.LocalCacheDir, "certs"))
		if err != nil {
			return nil, err
		}
		return certcache.NewCachingStore(cl.LocalCacheDir, lb, opts...)
	}
	awscfg, err := awsconfig.LoadUsingFlags(ctx, awscl)
	if err != nil {
		return nil, err
	}
	var sfs *awssecretsfs.T
	if certcache.HasReadonlyOption(opts) {
		sfs = awssecretsfs.New(awscfg)
	} else {
		sfs = awssecretsfs.New(awscfg, awssecretsfs.WithAllowCreation(true), awssecretsfs.WithAllowUpdates(true))
	}
	return certcache.NewCachingStore(cl.LocalCacheDir, sfs, opts...)
}

func getCert(ctx context.Context, values any, args []string) error {
	cl := values.(*getCertFlags)
	store, err := newCertStore(ctx, cl.TLSCertStoreFlags, cl.AWSFlags, certcache.WithReadonly(true))
	if err != nil {
		return err
	}
	host := args[0]
	cert, err := store.Get(ctx, host)
	if err != nil {
		return err
	}
	fmt.Println(string(cert))
	return nil
}

func putCert(ctx context.Context, values any, args []string) error {
	cl := values.(*putCertFlags)
	store, err := newCertStore(ctx, cl.TLSCertStoreFlags, cl.AWSFlags, certcache.WithReadonly(false))
	if err != nil {
		return err
	}
	file := args[0]
	buf, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return store.Put(ctx, file, buf)
}

/*
https://acme-staging-v02.api.letsencrypt.org/directory
hjttps://acme-v02.api.letsencrypt.org/directory


https://letsencrypt.org/certs/isrgrootx1.pem
https://letsencrypt.org/certs/isrg-root-x2.pem

https://letsencrypt.org/certs/staging/letsencrypt-stg-root-x1.pem
https://letsencrypt.org/certs/staging/letsencrypt-stg-root-x2.pem
*/
