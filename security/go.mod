module cloudeng.io/security

go 1.27.0

require (
	cloudeng.io/encoding v0.0.0-20261026191912-bcd1dece5d6c
	cloudeng.io/file v0.0.0-20261026191912-bcd1dece5d6c
	cloudeng.io/os v0.0.0-20261020215554-f8f4678915d5
)

require (
	cloudeng.io/algo v0.0.0-20261026191912-bcd1dece5d6c // indirect
	cloudeng.io/errors v0.0.14-0.20260312171538-61fcde6ce278 // indirect
	cloudeng.io/types v0.0.0-20261026191912-bcd1dece5d6c // indirect
	golang.org/x/sys v0.49.0 // indirect
)

replace (
	cloudeng.io/encoding => ../encoding
	cloudeng.io/types => ../types
)
