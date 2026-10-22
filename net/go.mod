module cloudeng.io/net

go 1.27.0

require (
	cloudeng.io/algo v0.0.0-20261020215554-f8f4678915d5
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/file v0.0.0-20260909165456-ddaa2de546a0
	cloudeng.io/logging v0.0.0-20261020215554-f8f4678915d5
	cloudeng.io/sync v0.0.12-0.20260804222138-e9281ed260ba
	golang.org/x/crypto v0.58.0
)

require (
	cloudeng.io/sys v0.0.0-20261020215554-f8f4678915d5 // indirect
	golang.org/x/sync v0.24.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
)

replace cloudeng.io/algo => ../algo

replace cloudeng.io/errors => ../errors

replace cloudeng.io/file => ../file

replace cloudeng.io/sys => ../sys

replace cloudeng.io/sync => ../sync
