module cloudeng.io/sync

go 1.25.5

require (
	cloudeng.io/debug v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
)

require golang.org/x/sync v0.21.0

replace cloudeng.io/debug => ../debug

replace cloudeng.io/errors => ../errors
