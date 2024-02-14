module cloudeng.io/webapp/cmd/acme

go 1.21

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20240213043943-f6a8f92f083f
	cloudeng.io/cmdutil v0.0.0-20240213043943-f6a8f92f083f
	cloudeng.io/errors v0.0.9
	cloudeng.io/webapp v0.0.0-20240213043943-f6a8f92f083f
	golang.org/x/crypto v0.19.0
)

require (
	cloudeng.io/file v0.0.0-20240214044655-223c29824207 // indirect
	cloudeng.io/os v0.0.0-20240213043943-f6a8f92f083f // indirect
	cloudeng.io/text v0.0.11 // indirect
	github.com/aws/aws-sdk-go-v2 v1.25.0 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.27.0 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.0 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.15.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.0 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.11.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.11.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.27.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.19.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.22.0 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.27.0 // indirect
	github.com/aws/smithy-go v1.20.0 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
