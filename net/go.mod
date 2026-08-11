module cloudeng.io/net

go 1.26.4

require (
	cloudeng.io/algo v0.0.0-20260807191443-11b7f4ecaaa0
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/file v0.0.0-20260527194618-4cb6d4558850
)

require (
	cloudeng.io/sync v0.0.12-0.20260804222138-e9281ed260ba // indirect
	cloudeng.io/sys v0.0.0-20260807191443-11b7f4ecaaa0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace cloudeng.io/algo => ../algo

replace cloudeng.io/errors => ../errors

replace cloudeng.io/file => ../file

replace cloudeng.io/sys => ../sys

replace cloudeng.io/sync => ../sync
