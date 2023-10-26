module cloudeng.io/webapp/cmd/webapp

go 1.16

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/cmdutil v0.0.0-20231026032435-4ad1389db593
	cloudeng.io/webapp v0.0.0-20231026032435-4ad1389db593
	github.com/julienschmidt/httprouter v1.3.0
)
