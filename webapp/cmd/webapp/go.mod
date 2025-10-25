module cloudeng.io/webapp/cmd/webapp

go 1.25

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/cmdutil v0.0.0-20251024233845-64530cbb2507
	cloudeng.io/webapp v0.0.0-20251024233845-64530cbb2507
	github.com/go-chi/chi/v5 v5.2.3
)

require (
	cloudeng.io/errors v0.0.12 // indirect
	cloudeng.io/file v0.0.0-20251024233845-64530cbb2507 // indirect
	cloudeng.io/io v0.0.0-20251024233845-64530cbb2507 // indirect
	cloudeng.io/logging v0.0.0-20251024233845-64530cbb2507 // indirect
	cloudeng.io/os v0.0.0-20251024233845-64530cbb2507 // indirect
	cloudeng.io/sync v0.0.8 // indirect
	cloudeng.io/text v0.0.11 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
