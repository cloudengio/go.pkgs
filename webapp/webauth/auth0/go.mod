module cloudeng.io/webapp/webauth/auth0

go 1.21

toolchain go1.21.6

replace cloudeng.io/webapp => ../..

require github.com/square/go-jose/v3 v3.0.0-20200630053402-0a67ce9b0693

require (
	golang.org/x/crypto v0.19.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
