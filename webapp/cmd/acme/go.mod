module cloudeng.io/webapp/cmd/acme

go 1.24.2

toolchain go1.24.4

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20250707204608-69242ac664a5
	cloudeng.io/cmdutil v0.0.0-20250707204608-69242ac664a5
	cloudeng.io/errors v0.0.12
	cloudeng.io/webapp v0.0.0-20250707204608-69242ac664a5
	golang.org/x/crypto v0.41.0
)

require (
	cloudeng.io/file v0.0.0-20250707204608-69242ac664a5 // indirect
	cloudeng.io/logging v0.0.0-20250707204608-69242ac664a5 // indirect
	cloudeng.io/os v0.0.0-20250707204608-69242ac664a5 // indirect
	cloudeng.io/text v0.0.11 // indirect
	github.com/aws/aws-sdk-go-v2 v1.37.2 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.30.3 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.18.3 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.2 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.37.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.27.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.32.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.36.0 // indirect
	github.com/aws/smithy-go v1.22.5 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
