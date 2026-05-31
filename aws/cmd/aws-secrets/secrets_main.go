// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build darwin

package main

import (
	"context"
	"fmt"
	"os"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/aws/awssecretsfs"
	"cloudeng.io/cmdutil/keys"
	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/macos/keychain/plugin"
	"cloudeng.io/security/keys/keychain/plugins"
	"github.com/aws/aws-sdk-go-v2/aws"
)

const cmdSpec = `name: secrets
summary: provide access to aws secretsmanager across multiple operating systems
commands:
  - name: read
    summary: read a secret from aws secretsmanager
    arguments:
  - name: write
    summary: write a secret to aws secretsmanager
    arguments:
      - <filename>
`

func cli() *subcmd.CommandSetYAML {
	cmd := subcmd.MustFromYAML(cmdSpec)
	var secretsCmd secretsCmd
	cmd.Set("read").MustRunner(secretsCmd.Read, &ReadFlags{})
	cmd.Set("write").MustRunner(secretsCmd.Write, &WriteFlags{})
	return cmd
}

func main() {
	ctx := context.Background()
	subcmd.Dispatch(ctx, cli())
}

type secretsCmd struct{}

type ARNFlags struct {
	ARN string `subcmd:"arn,,arn of the secret to use instead of the filename"`
}

type Flags struct {
	awsconfig.AWSFlags
	plugin.ReadFlags
	KeychainItem string `subcmd:"keychain-item,,keychain item to use instead of the filename"`
	ARNFlags
}

type ReadFlags struct {
	Flags
	OutputFile string `subcmd:"output,,'output file to write the secret to, use - for stdout'"`
}

type WriteFlags struct {
	Flags
}

func (sc secretsCmd) config(ctx context.Context, fv Flags) (context.Context, aws.Config, error) {
	if fv.KeychainItem == "" {
		return ctx, aws.Config{}, fmt.Errorf("no keychain item provided")
	}
	if fv.AWSKeyInfoID == "" {
		return ctx, aws.Config{}, fmt.Errorf("no key info ID provided")
	}
	kcCfg, err := fv.ReadFlags.Config()
	if err != nil {
		return ctx, aws.Config{}, fmt.Errorf("failed to get keychain config from flags: %w", err)
	}
	fs := plugins.NewFS(kcCfg.Binary, kcCfg)
	ims := keys.NewInMemoryKeyStore()
	if err := ims.ReadYAML(ctx, fs, fv.KeychainItem); err != nil {
		return ctx, aws.Config{}, fmt.Errorf("failed to read: %v: %w", fv.KeychainItem, err)
	}
	ctx = keys.ContextWithKeyStore(ctx, ims)
	fv.AWS = true
	cfg := fv.AWSFlags.Config()
	awscfg, err := cfg.Load(ctx)
	if err != nil {
		return ctx, aws.Config{}, err
	}
	return ctx, awscfg, nil
}

func (sc secretsCmd) Read(ctx context.Context, f any, _ []string) error {
	fl := f.(*ReadFlags)
	ctx, cfg, err := sc.config(ctx, fl.Flags)
	if err != nil {
		return fmt.Errorf("%w", handleError(err))
	}
	if fl.ARN == "" {
		return fmt.Errorf("missing secret ARN or name; use --arn to specify it")
	}
	fs := awssecretsfs.New(cfg)
	contents, err := fs.ReadFileCtx(ctx, fl.ARN)
	if err != nil {
		return fmt.Errorf("%s: %w", fl.ARN, handleError(err))
	}
	if len(fl.OutputFile) != 0 {
		if fl.OutputFile == "-" {
			os.Stdout.Write(contents)
		} else {
			if err := os.WriteFile(fl.OutputFile, contents, 0600); err != nil {
				return handleError(err)
			}
		}
		return nil
	}
	fmt.Printf("%s: exists, use --output-file to write to a file, use - for stdout\n", fl.ARN)
	return nil
}

func (sc secretsCmd) Write(ctx context.Context, f any, args []string) error {
	fl := f.(*WriteFlags)
	ctx, cfg, err := sc.config(ctx, fl.Flags)
	if err != nil {
		return fmt.Errorf("%s: %w", args[0], handleError(err))
	}
	fs := awssecretsfs.New(cfg, awssecretsfs.WithAllowUpdates(true))
	filename := args[0]
	contents, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	fmt.Printf("writing item %q to keychain\n", fl.ARN)
	err = fs.WriteFileCtx(ctx, fl.ARN, contents, 0000)
	return handleError(err)
}

func handleError(err error) error {
	// placeholder for catching AWS specific errors.
	return err
}

// TODO add support for decoding TLS certificates
