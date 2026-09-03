module cloudeng.io/file

go 1.27

require (
	cloudeng.io/algo v0.0.0-20260903161432-3e39c500cdbf
	cloudeng.io/cmdutil v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/logging v0.0.0-20260824023931-9b6c51abac7f
	cloudeng.io/os v0.0.0-20260825050644-3d0fba22c536
	cloudeng.io/path v0.0.10-0.20260312171538-61fcde6ce278
	cloudeng.io/sync v0.0.12-0.20260804222138-e9281ed260ba
	cloudeng.io/sys v0.0.0-20260825050644-3d0fba22c536
	cloudeng.io/text v0.0.16-0.20260624171915-da98fe9dec2b
	cloudeng.io/windows v0.0.0-20251203211350-c30caae1cc5e
	golang.org/x/net v0.58.0
	golang.org/x/sys v0.47.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloudeng.io/debug v0.0.0-20260527194618-4cb6d4558850 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	golang.org/x/sync v0.22.0 // indirect
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
