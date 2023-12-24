module cloudeng.io/webapp/cmd/acme

go 1.21

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20231219174858-fd89ad37703c
	cloudeng.io/cmdutil v0.0.0-20231219174858-fd89ad37703c
	cloudeng.io/errors v0.0.9
	cloudeng.io/webapp v0.0.0-20231219174858-fd89ad37703c
	golang.org/x/crypto v0.17.0
)

require (
	cloudeng.io/file v0.0.0-20231224020430-ceb8702695a4 // indirect
	cloudeng.io/os v0.0.0-20231219174858-fd89ad37703c // indirect
	cloudeng.io/path v0.0.8 // indirect
	cloudeng.io/text v0.0.11 // indirect
	github.com/aws/aws-sdk-go-v2 v1.24.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.26.2 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.16.13 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.14.10 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.2.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.5.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.7.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.10.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.10.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.26.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.18.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.21.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.26.6 // indirect
	github.com/aws/smithy-go v1.19.0 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/sys v0.15.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
