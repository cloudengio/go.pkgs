module cloudeng.io/webapp/cmd/webapp

go 1.21

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/cmdutil v0.0.0-20240213043943-f6a8f92f083f
	cloudeng.io/webapp v0.0.0-20240213043943-f6a8f92f083f
	github.com/julienschmidt/httprouter v1.3.0
)

require (
	cloudeng.io/file v0.0.0-20240213043943-f6a8f92f083f // indirect
	cloudeng.io/io v0.0.0-20240213043943-f6a8f92f083f // indirect
	cloudeng.io/os v0.0.0-20240213043943-f6a8f92f083f // indirect
	cloudeng.io/text v0.0.11 // indirect
	golang.org/x/net v0.21.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
