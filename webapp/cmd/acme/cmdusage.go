// panic: flag tls-cert-store-type already defined for this flag.FlagSet
//
// goroutine 1 [running]:
// cloudeng.io/cmdutil/subcmd.MustRegisterFlagStruct(...)
//
//	/Users/cnicolaou/LocalOnly/dev/github.com/cloudengio/go.pkgs/cmdutil/subcmd/subcmd.go:242
//
// main.redirectCmd()
//
//	/Users/cnicolaou/LocalOnly/dev/github.com/cloudengio/go.pkgs/webapp/cmd/acme/redirect.go:30 +0x134
//
// main.init.1()
//
//	/Users/cnicolaou/LocalOnly/dev/github.com/cloudengio/go.pkgs/webapp/cmd/acme/main.go:55 +0x1c4
package main
