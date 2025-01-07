module cloudeng.io/webapp/cmd/webapp

go 1.21

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/cmdutil v0.0.0-20241215221655-bd556f44d3de
	cloudeng.io/webapp v0.0.0-20241215221655-bd556f44d3de
	github.com/julienschmidt/httprouter v1.3.0
)

require (
	cloudeng.io/file v0.0.0-20241215221655-bd556f44d3de // indirect
	cloudeng.io/io v0.0.0-20241215221655-bd556f44d3de // indirect
	cloudeng.io/os v0.0.0-20241215221655-bd556f44d3de // indirect
	cloudeng.io/text v0.0.11 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
