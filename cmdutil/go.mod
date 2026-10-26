module cloudeng.io/cmdutil

go 1.27.0

require (
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/file v0.0.0-20261026191912-bcd1dece5d6c
	cloudeng.io/logging v0.0.0-20260923165344-0acebac4c1e9
	cloudeng.io/sync v0.0.12-0.20260804222138-e9281ed260ba
	cloudeng.io/text v0.0.16-0.20260624171915-da98fe9dec2b
	cloudeng.io/types v0.0.0-20261026191912-bcd1dece5d6c
	gopkg.in/yaml.v3 v3.0.1
)

require (
	cloudeng.io/algo v0.0.0-20261026191912-bcd1dece5d6c // indirect
	cloudeng.io/sys v0.0.0-20261026191912-bcd1dece5d6c // indirect
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
)

replace cloudeng.io/errors => ../errors

replace cloudeng.io/file => ../file

replace cloudeng.io/logging => ../logging

replace cloudeng.io/sync => ../sync

replace cloudeng.io/text => ../text

replace cloudeng.io/sys => ../sys

replace cloudeng.io/algo => ../algo
