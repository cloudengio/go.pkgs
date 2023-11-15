module cloudeng.io/webapp/cmd/webapp

go 1.16

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/cmdutil v0.0.0-20231106193145-45237a5eabab
	cloudeng.io/webapp v0.0.0-20231106193145-45237a5eabab
	github.com/julienschmidt/httprouter v1.3.0
)
