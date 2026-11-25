module cloudeng.io/net

go 1.27.0

require (
	cloudeng.io/algo v0.0.0-20261123235553-076c9b54dc11
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/file v0.0.0-20261026191912-bcd1dece5d6c
	cloudeng.io/logging v0.0.0-20261123235553-076c9b54dc11
	cloudeng.io/sync v0.0.12-0.20260804222138-e9281ed260ba
	golang.org/x/crypto v0.59.0
	golang.org/x/sys v0.50.0
)

require (
	cloudeng.io/sys v0.0.0-20261123235553-076c9b54dc11 // indirect
	golang.org/x/sync v0.25.0 // indirect
)

replace cloudeng.io/algo => ../algo

replace cloudeng.io/errors => ../errors

replace cloudeng.io/file => ../file

replace cloudeng.io/sys => ../sys

replace cloudeng.io/sync => ../sync
