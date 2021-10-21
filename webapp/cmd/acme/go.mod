module cloudeng.io/webapp/cmd/acme

go 1.16

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/aws v0.0.0-20211021163203-fcd6492c88b6
	cloudeng.io/cmdutil v0.0.0-20211021175011-f84b924825d7
	cloudeng.io/errors v0.0.6
	cloudeng.io/webapp v0.0.0-20210810185238-8d5e0be9ddb1
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519
)
