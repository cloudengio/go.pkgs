module cloudeng.io/webapp/cmd/webapp

go 1.23.0

toolchain go1.24.2

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/cmdutil v0.0.0-20250119024745-8a46e9bdda10
	cloudeng.io/webapp v0.0.0-20250119024745-8a46e9bdda10
	github.com/julienschmidt/httprouter v1.3.0
)

require (
	cloudeng.io/file v0.0.0-20250119024745-8a46e9bdda10 // indirect
	cloudeng.io/io v0.0.0-20250119024745-8a46e9bdda10 // indirect
	cloudeng.io/os v0.0.0-20250119024745-8a46e9bdda10 // indirect
	cloudeng.io/text v0.0.11 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
