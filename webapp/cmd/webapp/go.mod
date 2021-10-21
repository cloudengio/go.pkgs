module cloudeng.io/webapp/cmd/webapp

go 1.16

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/cmdutil v0.0.0-20211021175011-f84b924825d7
	cloudeng.io/webapp v0.0.0-20210810185238-8d5e0be9ddb1
	github.com/julienschmidt/httprouter v1.3.0
)
