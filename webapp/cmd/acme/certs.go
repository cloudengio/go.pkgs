// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/aws/awssecretsfs"
	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/file/localfs"
	"cloudeng.io/webapp/webauth/acme"
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

func certSubCmd() *subcmd.Command {
	putCertCmd := subcmd.NewCommand("put", subcmd.MustRegisterFlagStruct(&putCertFlags{}, nil, nil), putCert, subcmd.ExactlyNumArguments(1))
	putCertCmd.Document(`store a certificate in a cert store`)
	getCertCmd := subcmd.NewCommand("get", subcmd.MustRegisterFlagStruct(&getCertFlags{}, nil, nil), getCert, subcmd.ExactlyNumArguments(1))
	getCertCmd.Document(`retrieve a certificate from a cert store`)
	summary := `store and retrieve certificates directly from a certificate store.`
	certCmds := subcmd.NewCommandSet(putCertCmd, getCertCmd)
	certCmds.Document(summary)
	cl := subcmd.NewCommandLevel("cert-store", certCmds)
	cl.Document(summary)
	return cl
}

func newCertStore(ctx context.Context, cl TLSCertStoreFlags, awscl awsconfig.AWSFlags, readonly bool) (*acme.CachingStore, error) {
	if cl.UseAWSSecretsManager && !awscl.AWS {
		return nil, fmt.Errorf("aws-secrets-manager flag requires aws configuration to be enabled")
	}
	if !cl.UseAWSSecretsManager || !awscl.AWS {
		return acme.NewCachingStore(cl.LocalCacheDir, localfs.New(), readonly), nil
	}
	awscfg, err := awsconfig.LoadUsingFlags(ctx, awscl)
	if err != nil {
		return nil, err
	}
	var sfs *awssecretsfs.T
	if readonly {
		sfs = awssecretsfs.New(awscfg)
	} else {
		sfs = awssecretsfs.New(awscfg, awssecretsfs.WithAllowCreation(true), awssecretsfs.WithAllowUpdates(true))
	}
	return acme.NewCachingStore(cl.LocalCacheDir, sfs, readonly), nil
}

func getCert(ctx context.Context, values interface{}, args []string) error {
	cl := values.(*getCertFlags)
	store, err := newCertStore(ctx, cl.TLSCertStoreFlags, cl.AWSFlags, true)
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

func putCert(ctx context.Context, values interface{}, args []string) error {
	cl := values.(*putCertFlags)
	store, err := newCertStore(ctx, cl.TLSCertStoreFlags, cl.AWSFlags, false)
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
