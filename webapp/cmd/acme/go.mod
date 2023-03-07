module cloudeng.io/webapp/cmd/acme

go 1.16

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20230306215119-e71b407605cc
	cloudeng.io/cmdutil v0.0.0-20230306215119-e71b407605cc
	cloudeng.io/errors v0.0.8
	cloudeng.io/webapp v0.0.0-20230306215119-e71b407605cc
	golang.org/x/crypto v0.7.0
)
