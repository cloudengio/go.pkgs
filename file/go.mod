module cloudeng.io/file

go 1.25.5

require (
	cloudeng.io/algo v0.0.0-20260612215057-2c1fdd49d80a
	cloudeng.io/cmdutil v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/logging v0.0.0-20260611161950-23029f4a5674
	cloudeng.io/os v0.0.0-20260612215057-2c1fdd49d80a
	cloudeng.io/path v0.0.10-0.20260312171538-61fcde6ce278
	cloudeng.io/sync v0.0.11
	cloudeng.io/sys v0.0.0-20260612215057-2c1fdd49d80a
	cloudeng.io/text v0.0.16-0.20260312171538-61fcde6ce278
	cloudeng.io/windows v0.0.0-20251203211350-c30caae1cc5e
	golang.org/x/net v0.56.0
	golang.org/x/sys v0.46.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloudeng.io/debug v0.0.0-20260527194618-4cb6d4558850 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	golang.org/x/sync v0.21.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

replace cloudeng.io/algo => ../algo

replace cloudeng.io/cmdutil => ../cmdutil

replace cloudeng.io/errors => ../errors

replace cloudeng.io/logging => ../logging

replace cloudeng.io/os => ../os

replace cloudeng.io/path => ../path

replace cloudeng.io/sync => ../sync

replace cloudeng.io/sys => ../sys

replace cloudeng.io/text => ../text
