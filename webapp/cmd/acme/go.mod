module cloudeng.io/webapp/cmd/acme

go 1.22.0

toolchain go1.23.1

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20241215221655-bd556f44d3de
	cloudeng.io/cmdutil v0.0.0-20241215221655-bd556f44d3de
	cloudeng.io/errors v0.0.10
	cloudeng.io/webapp v0.0.0-20241215221655-bd556f44d3de
	golang.org/x/crypto v0.32.0
)

require (
	cloudeng.io/file v0.0.0-20241215221655-bd556f44d3de // indirect
	cloudeng.io/os v0.0.0-20241215221655-bd556f44d3de // indirect
	cloudeng.io/text v0.0.11 // indirect
	github.com/aws/aws-sdk-go-v2 v1.32.7 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.28.7 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.48 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.26 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.26 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.34.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.3 // indirect
	github.com/aws/smithy-go v1.22.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
