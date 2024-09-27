// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"cloudeng.io/aws/awscertstore"
	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/errors"
	"cloudeng.io/webapp"
)

type putCertFlags struct {
	webapp.TLSCertStoreFlags
	awsconfig.AWSFlags
}

type getCertFlags struct {
	webapp.TLSCertStoreFlags
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

func newCertStore(ctx context.Context, cl webapp.TLSCertStoreFlags, awscl awsconfig.AWSFlags) (webapp.CertStore, error) {
	if cl.ListStoreTypes {
		return nil, errors.New(strings.Join(webapp.RegisteredCertStores(), "\n"))
	}
	if !awscl.AWS {
		return webapp.NewCertStore(ctx, cl.CertStoreType, cl.CertStore)
	}
	awscfg, err := awsconfig.LoadUsingFlags(ctx, awscl)
	if err != nil {
		return nil, err
	}
	return webapp.NewCertStore(ctx, cl.CertStoreType, cl.CertStore,
		awscertstore.WithAWSConfig(awscfg))
}

func getCert(ctx context.Context, values interface{}, args []string) error {
	cl := values.(*getCertFlags)
	store, err := newCertStore(ctx, cl.TLSCertStoreFlags, cl.AWSFlags)
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
	store, err := newCertStore(ctx, cl.TLSCertStoreFlags, cl.AWSFlags)
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
