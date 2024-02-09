// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// This command is an example of how to create a simple webapp that uses
// react for the browser-side app and serves API endpoints for use by that
// app. See the comments and command line flag/help messages.
package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"time"

	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/webapp"
	"cloudeng.io/webapp/webassets"
	"cloudeng.io/webapp/webpack"
	"github.com/julienschmidt/httprouter"
)

// Use go generate to create and build the sample react app.
//go:generate npx create-react-app webapp-sample
//go:generate yarn --cwd webapp-sample build

//go:embed webapp-sample/build webapp-sample/build/static/css webapp-sample/build/static/js webapp-sample/build/static/media
var webpackedAssets embed.FS
var webpackedAssetPrefix = "webapp-sample/build"

type ProdServerFlags struct {
	webapp.HTTPServerFlags
}

type DevServerFlags struct {
	webapp.HTTPServerFlags
	webpack.DevServerFlags
	webassets.AssetsFlags
}

var cmdSet *subcmd.CommandSet

func init() {

	// Production server.
	prodServerFlagSet := subcmd.MustRegisterFlagStruct(&ProdServerFlags{}, nil, nil)
	prodServeCmd := subcmd.NewCommand("prod", prodServerFlagSet, prodServe, subcmd.ExactlyNumArguments(0))
	prodServeCmd.Document(`run a production server.`)

	// Development server.
	devServerFlagSet := subcmd.MustRegisterFlagStruct(&DevServerFlags{}, nil, nil)
	devServeCmd := subcmd.NewCommand("dev", devServerFlagSet, devServe, subcmd.ExactlyNumArguments(0))
	devServeCmd.Document(`run a development server.`)

	certCmd := webapp.SelfSignedCertCommand("self-signed-cert")

	cmdSet = subcmd.NewCommandSet(prodServeCmd, devServeCmd, certCmd)
	cmdSet.Document(`Run a webapp server. Two modes are supported: production and
	development.

	For production, all assets are built into the production server's binary.

	For development, the front-end code/assets can be managed in two ways:

	1. with embedded assets that can be overridden with newer or new files found on the local filesystem. This is generally used when the javascript/webapp tooling only supports generating new assets rather than any form of dynamic update. The user must reload the site/page to see the new version.
 
    2. with a development server, such as that provided by webpack, that dynamically
    monitors the javascript/webapp code and dynamically rebuilds the assets. To use
    this mode, this application will proxy all of the urls that it doesn't itself
    implement to the running development server. The dev server may be started by
    this server via the --webpack-dir option. Alternatively, a running dev server
    may be used via the --webpack-server option.

	If a self-signed cerificate is required, the cert command can be used to generate one.`)
}

func main() {
	cmdSet.MustDispatch(context.Background())
}

func prodServe(ctx context.Context, values interface{}, _ []string) error {
	ctx, done := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer done()
	cl := values.(*ProdServerFlags)
	router := httprouter.New()

	configureAPIEndpoints(router)

	cfg, err := webapp.TLSConfigFromFlags(ctx, cl.HTTPServerFlags)
	if err != nil {
		return err
	}

	ln, srv, err := webapp.NewTLSServer(cl.HTTPServerFlags.Address, router, cfg)
	if err != nil {
		return err
	}

	// Force all http traffic to an https port.
	if err := webapp.RedirectPort80(ctx, cl.HTTPServerFlags.Address, cl.AcmeRedirectTarget); err != nil {
		return err
	}

	log.Printf("running on %s", ln.Addr())

	assets := webassets.NewAssets(webpackedAssetPrefix, webpackedAssets)
	router.ServeFiles("/*filepath", http.FS(assets))
	return webapp.ServeTLSWithShutdown(ctx, ln, srv, 5*time.Second)
}

func devServe(ctx context.Context, values interface{}, _ []string) error {
	ctx, done := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer done()
	cl := values.(*DevServerFlags)
	router := httprouter.New()

	configureAPIEndpoints(router)

	cfg, err := webapp.TLSConfigFromFlags(ctx, cl.HTTPServerFlags)
	if err != nil {
		return err
	}

	ln, srv, err := webapp.NewTLSServer(cl.HTTPServerFlags.Address, router, cfg)
	if err != nil {
		return err
	}

	var dsURL *url.URL
	switch {
	case len(cl.WebpackServer) > 0:
		dsURL, err = url.Parse(cl.WebpackServer)
		if err == nil {
			routeToProxy(router, "/build/", dsURL)
		}
	case len(cl.WebpackDir) > 0:
		dsURL, err = runWebpackDevServer(ctx, cl.WebpackDir, cl.Address)
		if err == nil {
			routeToProxy(router, "/build/", dsURL)
		}
	default:
		assets := webassets.NewAssets(webpackedAssetPrefix, webpackedAssets,
			webassets.OptionsFromFlags(&cl.AssetsFlags)...)
		router.ServeFiles("/*filepath", http.FS(assets))
		router.NotFound = serveIndexHTML(assets) // serve index.html on all urls.
	}
	if err != nil {
		return err
	}

	log.Printf("running on %s", ln.Addr())
	return webapp.ServeTLSWithShutdown(ctx, ln, srv, 5*time.Second)
}

func configureAPIEndpoints(router *httprouter.Router) {
	router.HandlerFunc("GET", "/hello", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "hello")
	})
}

func serveIndexHTML(fsys fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		code, err := webassets.ServeFile(w, fsys, "index.html")
		w.WriteHeader(code)
		if err != nil {
			log.Printf("index.html: %v %v", code, err)
		}
	}
}

func runWebpackDevServer(ctx context.Context, webpackDir, address string) (*url.URL, error) {
	wpsrv := webpack.NewDevServer(ctx, webpackDir, "yarn", "start", "--public", address)
	wpsrv.Configure(webpack.SetSdoutStderr(os.Stdout, os.Stdout),
		webpack.AddrRegularExpression(regexp.MustCompile("Local:")))
	if err := wpsrv.Start(); err != nil {
		return nil, err
	}
	return wpsrv.WaitForURL(ctx)
}

func routeToProxy(router *httprouter.Router, _ string, url *url.URL) {
	// TODO: understand what publicPath (now _), was intended for, since
	// it's set to /build/ on the call sites, but overridden to / here.
	proxy := httputil.NewSingleHostReverseProxy(url)
	// Proxy websockets to allow for fast refresh - ie. changes made
	// in the javascript react code is immediately reflected in the UI
	// without a page reload.
	router.Handler("GET", "/sockjs-node", proxy)
	router.Handler("POST", "/sockjs-node", proxy)
	router.Handler("GET", "/", proxy)
	router.NotFound = proxy
}
