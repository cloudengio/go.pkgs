module cloudeng.io/vms

go 1.26.2

require (
	cloudeng.io/cicd v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278
	cloudeng.io/os v0.0.0-20260527194618-4cb6d4558850
	cloudeng.io/sync v0.0.11
)

require (
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
)

replace cloudeng.io/cicd => ../cicd

replace cloudeng.io/errors => ../errors

replace cloudeng.io/os => ../os

replace cloudeng.io/sync => ../sync
