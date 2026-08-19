module cloudeng.io/os

go 1.25.5

require (
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/windows v0.0.0-20251203211350-c30caae1cc5e
	golang.org/x/sys v0.47.0
)

require cloudeng.io/algo v0.0.0-20260818231247-605c3766963e

replace cloudeng.io/errors => ../errors
