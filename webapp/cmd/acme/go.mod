module cloudeng.io/webapp/cmd/acme

go 1.16

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20230307023515-8a194fbc7867
	cloudeng.io/cmdutil v0.0.0-20230307023515-8a194fbc7867
	cloudeng.io/errors v0.0.8
	cloudeng.io/webapp v0.0.0-20230307023515-8a194fbc7867
	golang.org/x/crypto v0.7.0
)
