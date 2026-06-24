module cloudeng.io/net

go 1.26.4

require (
	cloudeng.io/algo v0.0.0-20260622224828-d069000db737
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/file v0.0.0-20260527194618-4cb6d4558850
)

require (
	cloudeng.io/sync v0.0.11 // indirect
	cloudeng.io/sys v0.0.0-20260622224828-d069000db737 // indirect
	golang.org/x/sys v0.46.0 // indirect
)

replace cloudeng.io/algo => ../algo

replace cloudeng.io/errors => ../errors

replace cloudeng.io/file => ../file

replace cloudeng.io/sys => ../sys

replace cloudeng.io/sync => ../sync
