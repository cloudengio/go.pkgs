module cloudeng.io/security

go 1.27.0

require (
	cloudeng.io/encoding v0.0.0-20260909165456-ddaa2de546a0
	cloudeng.io/file v0.0.0-20260909165456-ddaa2de546a0
	cloudeng.io/os v0.0.0-20260909165456-ddaa2de546a0
)

require (
	cloudeng.io/algo v0.0.0-20260909165456-ddaa2de546a0 // indirect
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278 // indirect
	cloudeng.io/types v0.0.0-20260909165456-ddaa2de546a0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
)

replace (
	cloudeng.io/encoding => ../encoding
	cloudeng.io/types => ../types
)
