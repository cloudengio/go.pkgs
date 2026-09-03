module cloudeng.io/security

go 1.27.0

require (
	cloudeng.io/encoding v0.0.0-00010101000000-000000000000
	cloudeng.io/file v0.0.0-20260902201442-bb723e109d00
	cloudeng.io/os v0.0.0-20260903161432-3e39c500cdbf
)

require (
	cloudeng.io/algo v0.0.0-20260903161432-3e39c500cdbf // indirect
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278 // indirect
	cloudeng.io/types v0.0.0-00010101000000-000000000000 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace (
	cloudeng.io/encoding => ../encoding
	cloudeng.io/types => ../types
)
