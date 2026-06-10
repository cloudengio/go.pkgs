module cloudeng.io/cmdutil

go 1.25.5

require (
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/file v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/logging v0.0.0-20260606211206-13a5cf17eb80
	cloudeng.io/sync v0.0.11
	cloudeng.io/text v0.0.16-0.20260312171538-61fcde6ce278
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloudeng.io/algo v0.0.0-20260606211206-13a5cf17eb80 // indirect
	cloudeng.io/sys v0.0.0-20260606211206-13a5cf17eb80 // indirect
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
)

replace cloudeng.io/errors => ../errors

replace cloudeng.io/file => ../file

replace cloudeng.io/logging => ../logging

replace cloudeng.io/sync => ../sync

replace cloudeng.io/text => ../text

replace cloudeng.io/sys => ../sys

replace cloudeng.io/algo => ../algo
