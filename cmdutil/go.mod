module cloudeng.io/cmdutil

go 1.26.4

require (
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/file v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/logging v0.0.0-20260806150854-f21c21e021b8
	cloudeng.io/sync v0.0.12-0.20260804222138-e9281ed260ba
	cloudeng.io/text v0.0.16-0.20260624171915-da98fe9dec2b
	golang.org/x/exp v0.0.0-20260813180055-c1d0aacb2297
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloudeng.io/algo v0.0.0-20260807191443-11b7f4ecaaa0 // indirect
	cloudeng.io/sys v0.0.0-20260807191443-11b7f4ecaaa0 // indirect
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace cloudeng.io/errors => ../errors

replace cloudeng.io/file => ../file

replace cloudeng.io/logging => ../logging

replace cloudeng.io/sync => ../sync

replace cloudeng.io/text => ../text

replace cloudeng.io/sys => ../sys

replace cloudeng.io/algo => ../algo
