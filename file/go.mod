module cloudeng.io/file

go 1.25.5

require (
	cloudeng.io/algo v0.0.0-20260605174237-2d6c1041426f
	cloudeng.io/cmdutil v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/logging v0.0.0-20260602205728-76c4accb8394
	cloudeng.io/os v0.0.0-20260605174237-2d6c1041426f
	cloudeng.io/path v0.0.10-0.20260114020737-744f6c0f8e64
	cloudeng.io/sync v0.0.11
	cloudeng.io/sys v0.0.0-20260605174237-2d6c1041426f
	cloudeng.io/text v0.0.16-0.20260312171538-61fcde6ce278
	cloudeng.io/windows v0.0.0-20251203211350-c30caae1cc5e
	golang.org/x/net v0.55.0
	golang.org/x/sys v0.45.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloudeng.io/debug v0.0.0-20260527194618-4cb6d4558850 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	golang.org/x/sync v0.20.0 // indirect
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
