module cloudeng.io/cmdutil

go 1.27.0

require (
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/file v0.0.0-20260909165456-ddaa2de546a0
	cloudeng.io/logging v0.0.0-20260909165456-ddaa2de546a0
	cloudeng.io/sync v0.0.12-0.20260804222138-e9281ed260ba
	cloudeng.io/text v0.0.16-0.20260624171915-da98fe9dec2b
	cloudeng.io/types v0.0.0-20260909165456-ddaa2de546a0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloudeng.io/algo v0.0.0-20260909165456-ddaa2de546a0 // indirect
	cloudeng.io/sys v0.0.0-20260909165456-ddaa2de546a0 // indirect
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
)

replace cloudeng.io/errors => ../errors

replace cloudeng.io/file => ../file

replace cloudeng.io/logging => ../logging

replace cloudeng.io/sync => ../sync

replace cloudeng.io/text => ../text

replace cloudeng.io/sys => ../sys

replace cloudeng.io/algo => ../algo
