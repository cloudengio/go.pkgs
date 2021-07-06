// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package webapp and its sub-packages provide support for building
// webapps. This includes utility routines for  managing http.Server
// instances, generating self-signed TLS certificates etc. The
// sub-packages provide support for managing the assets to be
// served, various forms of authentication and common toolchains
// such as webpack. For production purposes assets are built into
// the server's binary, but for development they are built into
// the binary but can be overridden from a local filesystem or from
// a running development server that manages those assets (eg.
// a webpack dev server instance). This provides the flexibility for
// both simple deployment of production servers and iterative development
// within the same application.
//
// An example/template can be found in cmd/webapp.
package webapp
