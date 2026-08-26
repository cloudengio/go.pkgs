module cloudeng.io/cmdutil

go 1.27

require (
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/file v0.0.0-20260824023931-9b6c51abac7f
	cloudeng.io/logging v0.0.0-20260824023931-9b6c51abac7f
	cloudeng.io/sync v0.0.12-0.20260804222138-e9281ed260ba
	cloudeng.io/text v0.0.16-0.20260624171915-da98fe9dec2b
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloudeng.io/algo v0.0.0-20260825050644-3d0fba22c536 // indirect
	cloudeng.io/sys v0.0.0-20260825050644-3d0fba22c536 // indirect
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
