module cloudeng.io/webapp/cmd/acme

go 1.16

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20230913164637-56a6ca867a22
	cloudeng.io/cmdutil v0.0.0-20230913164637-56a6ca867a22
	cloudeng.io/errors v0.0.8
	cloudeng.io/webapp v0.0.0-20230913164637-56a6ca867a22
	github.com/aws/aws-sdk-go-v2/config v1.18.41 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.21.3 // indirect
	golang.org/x/crypto v0.13.0
)
