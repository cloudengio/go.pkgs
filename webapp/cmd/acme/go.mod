module cloudeng.io/webapp/cmd/acme

go 1.25

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20251025222319-366d597d8744
	cloudeng.io/cmdutil v0.0.0-20251029021443-ece2b02e8072
	cloudeng.io/errors v0.0.13-0.20251029021443-ece2b02e8072
	cloudeng.io/file v0.0.0-20251029021443-ece2b02e8072
	cloudeng.io/logging v0.0.0-20251029021443-ece2b02e8072
	cloudeng.io/net v0.0.0-20251029021443-ece2b02e8072
	cloudeng.io/sync v0.0.9-0.20251029021443-ece2b02e8072
	cloudeng.io/webapp v0.0.0-20251025222319-366d597d8744
	golang.org/x/crypto v0.43.0
)

require (
	cloudeng.io/algo v0.0.0-20251029021443-ece2b02e8072 // indirect
	cloudeng.io/os v0.0.0-20251029021443-ece2b02e8072 // indirect
	cloudeng.io/sys v0.0.0-20251029021443-ece2b02e8072 // indirect
	cloudeng.io/text v0.0.11 // indirect
	github.com/aws/aws-sdk-go-v2 v1.39.4 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.31.15 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.18.19 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.39.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.29.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.38.9 // indirect
	github.com/aws/smithy-go v1.23.1 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
