module cloudeng.io/net

go 1.27

require (
	cloudeng.io/algo v0.0.0-20260903161432-3e39c500cdbf
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/file v0.0.0-20260824023931-9b6c51abac7f
)

require (
	cloudeng.io/sync v0.0.12-0.20260804222138-e9281ed260ba // indirect
	cloudeng.io/sys v0.0.0-20260825050644-3d0fba22c536 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace cloudeng.io/algo => ../algo

replace cloudeng.io/errors => ../errors

replace cloudeng.io/file => ../file

replace cloudeng.io/sys => ../sys

replace cloudeng.io/sync => ../sync
