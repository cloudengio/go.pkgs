module cloudeng.io/webapp/cmd/acme

go 1.21

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20240204011218-453d510d0c93
	cloudeng.io/cmdutil v0.0.0-20240204011218-453d510d0c93
	cloudeng.io/errors v0.0.9
	cloudeng.io/webapp v0.0.0-20240204011218-453d510d0c93
	golang.org/x/crypto v0.18.0
)

require (
	cloudeng.io/file v0.0.0-20240204011218-453d510d0c93 // indirect
	cloudeng.io/os v0.0.0-20240204011218-453d510d0c93 // indirect
	cloudeng.io/path v0.0.9 // indirect
	cloudeng.io/text v0.0.11 // indirect
	github.com/aws/aws-sdk-go-v2 v1.24.1 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.26.6 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.16.16 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.2.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.5.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.10 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.26.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.18.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.21.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.7 // indirect
	github.com/aws/smithy-go v1.19.0 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
