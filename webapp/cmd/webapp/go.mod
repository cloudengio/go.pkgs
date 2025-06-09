module cloudeng.io/webapp/cmd/webapp

go 1.23.3

toolchain go1.24.2

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/cmdutil v0.0.0-20250609000856-e90addcdd7e2
	cloudeng.io/webapp v0.0.0-20250609000856-e90addcdd7e2
	github.com/julienschmidt/httprouter v1.3.0
)

require (
	cloudeng.io/file v0.0.0-20250609000856-e90addcdd7e2 // indirect
	cloudeng.io/io v0.0.0-20250609000856-e90addcdd7e2 // indirect
	cloudeng.io/os v0.0.0-20250609000856-e90addcdd7e2 // indirect
	cloudeng.io/text v0.0.11 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
