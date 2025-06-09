module cloudeng.io/webapp/webauth/auth0

go 1.23.0

toolchain go1.24.2

replace cloudeng.io/webapp => ../..

require github.com/square/go-jose/v3 v3.0.0-20200630053402-0a67ce9b0693

require (
	golang.org/x/crypto v0.39.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
