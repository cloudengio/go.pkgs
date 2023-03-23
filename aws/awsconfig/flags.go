// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package awsconfig

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// AWSFlags defines commonly used flags that control AWS behaviour.
type AWSFlags struct {
	AWS            bool   `subcmd:"aws,false,set to enable AWS functionality" yaml:"aws"`
	AWSProfile     string `subcmd:"aws-profile,,aws profile to use for config/authentication" yaml:"aws_profile"`
	AWSRegion      string `subcmd:"aws-region,,'aws region to use for API calls, overrides the region set in the profile'" yaml:"aws_region"`
	AWSConfigFiles string `subcmd:"aws-config-files,,comma separated list of config files to use in place of those commonly found in $HOME/.aws" yaml:"aws_config_files,flow"`
}

// LoadUsingFlags calls awsconfig.Load with options controlled by the
// the specified flags.
func LoadUsingFlags(ctx context.Context, cl AWSFlags) (aws.Config, error) {
	if !cl.AWS {
		return aws.Config{}, fmt.Errorf("aws not requested")
	}
	return Load(ctx, ConfigOptionsFromFlags(cl)...)
}

// ConfigOptionsFromFlags returns the ConfigOptions implied by the flags.
// NOTE: it always includes config.WithEC2IMDSRegion so that the region
// information is retrieved from EC2 IMDS when it's not found by other
// means.
func ConfigOptionsFromFlags(cl AWSFlags) []ConfigOption {
	opts := []ConfigOption{}
	if len(cl.AWSConfigFiles) > 0 {
		files := strings.Split(cl.AWSConfigFiles, ",")
		opts = append(opts,
			WithConfigOptions(config.WithSharedConfigFiles(files)),
		)
	}
	if len(cl.AWSProfile) > 0 {
		opts = append(opts,
			WithConfigOptions(config.WithSharedConfigProfile(cl.AWSProfile)))
	}
	if len(cl.AWSRegion) > 0 {
		opts = append(opts,
			WithConfigOptions(config.WithRegion(cl.AWSRegion)))
	}
	opts = append(opts, WithConfigOptions(
		config.WithEC2IMDSRegion(),
	))
	return opts
}
