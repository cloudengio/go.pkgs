# Package [cloudeng.io/aws/awsconfig](https://pkg.go.dev/cloudeng.io/aws/awsconfig?tab=doc)

```go
import cloudeng.io/aws/awsconfig
```

Package awsconfig provides support for obtaining configuration and
associated credentials information for use with AWS.

## Functions
### Func AccountID
```go
func AccountID(ctx context.Context, cfg aws.Config) (string, error)
```
AccountID uses the sts service to obtain the calling processes Amazon
Account ID (number).

### Func DebugPrintConfig
```go
func DebugPrintConfig(ctx context.Context, out io.Writer, cfg aws.Config) error
```
DebugPrintConfig dumps the aws.Config to help with debugging configuration
issues. It displays the types of the fields that can't be directly printed.

### Func Load
```go
func Load(ctx context.Context, opts ...ConfigOption) (aws.Config, error)
```
Load attempts to load configuration information from multiple sources,
including the current process' environment, shared configuration files (by
default $HOME/.aws) and also from ec2 instance metadata (currently for the
AWS region).

### Func LoadUsingFlags
```go
func LoadUsingFlags(ctx context.Context, cl AWSFlags) (aws.Config, error)
```
LoadUsingFlags calls awsconfig.Load with options controlled by the the
specified flags.



## Types
### Type AWSFlags
```go
type AWSFlags struct {
	AWS            bool   `subcmd:"aws,false,set to enable AWS functionality" yaml:"aws" cmd:"set to true enable AWS functionality"`
	AWSProfile     string `subcmd:"aws-profile,,aws profile to use for config/authentication" yaml:"aws_profile" cmd:"aws profile to use for config/authentication"`
	AWSRegion      string `subcmd:"aws-region,,'aws region to use for API calls, overrides the region set in the profile'" yaml:"aws_region" cmd:"aws region to use, overrides the region set in the profile"`
	AWSConfigFiles string `subcmd:"aws-config-files,,comma separated list of config files to use in place of those commonly found in $HOME/.aws" yaml:"aws_config_files,flow" cmd:"comma separated list of config files to use in place of those commonly found in $HOME/.aws"`
}
```
AWSFlags defines commonly used flags that control AWS behaviour.


### Type ConfigOption
```go
type ConfigOption func(o *options)
```
ConfigOption represents an option to Load.

### Functions

```go
func ConfigOptionsFromFlags(cl AWSFlags) []ConfigOption
```
ConfigOptionsFromFlags returns the ConfigOptions implied by the flags. NOTE:
it always includes config.WithEC2IMDSRegion so that the region information
is retrieved from EC2 IMDS when it's not found by other means.


```go
func WithConfigOptions(fn ...func(*config.LoadOptions) error) ConfigOption
```
WithConfigOptions will pass the supplied options from the aws config
package.







