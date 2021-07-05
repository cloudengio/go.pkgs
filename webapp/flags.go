// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package webapp

// HTTPServerFlags defines commonly used flags for running an http server.
type HTTPServerFlags struct {
	Address         string `subcmd:"https,:8080,address to run https web server on"`
	CertificateFile string `subcmd:"ssl-cert,localhost.crt,ssl certificate file"`
	KeyFile         string `subcmd:"ssl-key,localhost.key,ssl private key file"`
}
