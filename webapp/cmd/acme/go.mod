module cloudeng.io/webapp/cmd/acme

go 1.21

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20240522031305-dfb750a2120a
	cloudeng.io/cmdutil v0.0.0-20240522031305-dfb750a2120a
	cloudeng.io/errors v0.0.10
	cloudeng.io/webapp v0.0.0-20240522031305-dfb750a2120a
	golang.org/x/crypto v0.27.0
)

require (
	cloudeng.io/file v0.0.0-20240522031305-dfb750a2120a // indirect
	cloudeng.io/os v0.0.0-20240522031305-dfb750a2120a // indirect
	cloudeng.io/text v0.0.11 // indirect
	github.com/aws/aws-sdk-go-v2 v1.31.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.27.38 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.36 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.14 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.18 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.18 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.20 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.33.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.23.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.27.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.31.2 // indirect
	github.com/aws/smithy-go v1.21.0 // indirect
	golang.org/x/net v0.29.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
