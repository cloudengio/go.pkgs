module cloudeng.io/webapp/cmd/acme

go 1.23.0

toolchain go1.24.2

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20250119024745-8a46e9bdda10
	cloudeng.io/cmdutil v0.0.0-20250119024745-8a46e9bdda10
	cloudeng.io/errors v0.0.10
	cloudeng.io/webapp v0.0.0-20250119024745-8a46e9bdda10
	golang.org/x/crypto v0.37.0
)

require (
	cloudeng.io/file v0.0.0-20250119024745-8a46e9bdda10 // indirect
	cloudeng.io/os v0.0.0-20250119024745-8a46e9bdda10 // indirect
	cloudeng.io/text v0.0.11 // indirect
	github.com/aws/aws-sdk-go-v2 v1.36.3 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.29.14 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.67 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.30 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.35.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.25.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.30.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.19 // indirect
	github.com/aws/smithy-go v1.22.3 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
