module cloudeng.io/webapp/cmd/webapp

go 1.24.2

toolchain go1.24.4

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/cmdutil v0.0.0-20250820215211-e1b65c305908
	cloudeng.io/webapp v0.0.0-20250820215211-e1b65c305908
	github.com/julienschmidt/httprouter v1.3.0
)

require (
	cloudeng.io/file v0.0.0-20250820215211-e1b65c305908 // indirect
	cloudeng.io/io v0.0.0-20250820215211-e1b65c305908 // indirect
	cloudeng.io/logging v0.0.0-20250820215211-e1b65c305908 // indirect
	cloudeng.io/os v0.0.0-20250820215211-e1b65c305908 // indirect
	cloudeng.io/text v0.0.11 // indirect
	github.com/go-chi/chi/v5 v5.2.3 // indirect
	golang.org/x/net v0.44.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
