module cloudeng.io/webapp/cmd/webapp

go 1.21

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/cmdutil v0.0.0-20231129163852-526b6ff59b6d
	cloudeng.io/webapp v0.0.0-20231130182733-8193ad9948bc
	github.com/julienschmidt/httprouter v1.3.0
)

require (
	cloudeng.io/file v0.0.0-20231130182733-8193ad9948bc // indirect
	cloudeng.io/io v0.0.0-20231130182733-8193ad9948bc // indirect
	cloudeng.io/os v0.0.0-20231130182733-8193ad9948bc // indirect
	cloudeng.io/path v0.0.8 // indirect
	cloudeng.io/text v0.0.11 // indirect
	golang.org/x/net v0.19.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
