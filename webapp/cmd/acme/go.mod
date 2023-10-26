module cloudeng.io/webapp/cmd/acme

go 1.16

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20231026032435-4ad1389db593
	cloudeng.io/cmdutil v0.0.0-20231026032435-4ad1389db593
	cloudeng.io/errors v0.0.8
	cloudeng.io/webapp v0.0.0-20231026032435-4ad1389db593
	github.com/aws/aws-sdk-go-v2/config v1.19.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.21.6 // indirect
	golang.org/x/crypto v0.14.0
)
