module cloudeng.io/os

go 1.26.0

require (
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/windows v0.0.0-20251203211350-c30caae1cc5e
	golang.org/x/sys v0.50.0
)

require cloudeng.io/algo v0.0.0-20261111225710-e0ef01d9df9b

replace cloudeng.io/errors => ../errors
