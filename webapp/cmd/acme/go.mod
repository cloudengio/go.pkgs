module cloudeng.io/webapp/cmd/acme

go 1.16

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20231106193145-45237a5eabab
	cloudeng.io/cmdutil v0.0.0-20231106193145-45237a5eabab
	cloudeng.io/errors v0.0.9
	cloudeng.io/webapp v0.0.0-20231106193145-45237a5eabab
	github.com/aws/aws-sdk-go-v2/config v1.25.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.23.1 // indirect
	golang.org/x/crypto v0.15.0
)
