module cloudeng.io/webapp/webauth/auth0

go 1.21

toolchain go1.21.6

replace cloudeng.io/webapp => ../..

require github.com/square/go-jose/v3 v3.0.0-20200630053402-0a67ce9b0693

require (
	cloudeng.io/file v0.0.0-20240214013242-3c0d4550fc32 // indirect
	golang.org/x/crypto v0.18.0 // indirect
)
