module cloudeng.io/webapp/cmd/webapp

go 1.21

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/cmdutil v0.0.0-20240522031305-dfb750a2120a
	cloudeng.io/webapp v0.0.0-20240522031305-dfb750a2120a
	github.com/julienschmidt/httprouter v1.3.0
)

require (
	cloudeng.io/file v0.0.0-20240522031305-dfb750a2120a // indirect
	cloudeng.io/io v0.0.0-20240522031305-dfb750a2120a // indirect
	cloudeng.io/os v0.0.0-20240522031305-dfb750a2120a // indirect
	cloudeng.io/text v0.0.11 // indirect
	golang.org/x/net v0.29.0 // indirect
	golang.org/x/text v0.18.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
