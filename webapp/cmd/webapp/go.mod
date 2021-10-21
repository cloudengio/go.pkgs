module cloudeng.io/webapp/cmd/webapp

go 1.16

replace cloudeng.io/webapp => ../..

require (
	cloudeng.io/cmdutil v0.0.0-20211021192936-4f588727116a
	cloudeng.io/io v0.0.0-20211021192936-4f588727116a // indirect
	cloudeng.io/text v0.0.9 // indirect
	cloudeng.io/webapp v0.0.0-20211021192936-4f588727116a
	github.com/julienschmidt/httprouter v1.3.0
)
