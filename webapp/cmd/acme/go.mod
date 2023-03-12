module cloudeng.io/webapp/cmd/acme

go 1.16

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20230309184059-9263072b1423
	cloudeng.io/cmdutil v0.0.0-20230309184059-9263072b1423
	cloudeng.io/errors v0.0.8
	cloudeng.io/webapp v0.0.0-20230309184059-9263072b1423
	github.com/aws/aws-sdk-go-v2/config v1.18.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/secretsmanager v1.19.0 // indirect
	golang.org/x/crypto v0.7.0
)
