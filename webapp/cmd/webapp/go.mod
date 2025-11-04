module cloudeng.io/webapp/cmd/webapp

go 1.25

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/cmdutil v0.0.0-20251104015021-54a496e297ea
	cloudeng.io/webapp v0.0.0-20251025222319-366d597d8744
	github.com/go-chi/chi/v5 v5.2.3
)

require (
	cloudeng.io/errors v0.0.13-0.20251104015021-54a496e297ea // indirect
	cloudeng.io/file v0.0.0-20251104015021-54a496e297ea // indirect
	cloudeng.io/io v0.0.0-20251104015021-54a496e297ea // indirect
	cloudeng.io/logging v0.0.0-20251104015021-54a496e297ea // indirect
	cloudeng.io/os v0.0.0-20251104015021-54a496e297ea // indirect
	cloudeng.io/sync v0.0.9-0.20251104015021-54a496e297ea // indirect
	cloudeng.io/text v0.0.12-0.20251104001519-e5e3fc20cf3e // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
