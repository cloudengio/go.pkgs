module cloudeng.io/webapp/cmd/webapp

go 1.16

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/cmdutil v0.0.0-20230309184059-9263072b1423
	cloudeng.io/webapp v0.0.0-20230309184059-9263072b1423
	github.com/julienschmidt/httprouter v1.3.0
)
