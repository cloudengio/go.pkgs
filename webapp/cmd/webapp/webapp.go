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
	"time"

	"cloudeng.io/cmdutil/subcmd"
	"cloudeng.io/webapp"
	"cloudeng.io/webapp/devserver"
	"cloudeng.io/webapp/webassets"
	"github.com/go-chi/chi/v5"
)

// Use go generate to create and build the sample react app.
//go:generate npx create-react-app webapp-sample
//go:generate yarn --cwd webapp-sample build
//go:generate ./create-vite-react.sh

//go:embed webapp-sample/build webapp-sample/build/static/css webapp-sample/build/static/js webapp-sample/build/static/media
var webpackedAssets embed.FS

type ProdServerFlags struct {
	webapp.HTTPServerFlags
	webapp.TLSCertConfig
}

type WebpackFlags struct {
	WebpackDir    string `subcmd:"webpack-dir,,'set to a directory containing a webpack configuration with the webpack dev server configured. This dev server will then be started and requests proxied to it.'"`
	WebpackServer string `subcmd:"webpack-server,,set to the url of an already running webpack dev server to which requests will be proxied."`
}

type ViteFlags struct {
	ViteDir    string `subcmd:"vite-dir,,'set to a directory containing a vite configuration with the vite dev server configured. This dev server will then be started and requests proxied to it.'"`
	ViteServer string `subcmd:"vite-server,,set to the url of an already running vite dev server to which requests will be proxied."`
}

type DevServerFlags struct {
	webapp.HTTPServerFlags
	WebpackFlags
	ViteFlags
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

	cmdSet = subcmd.NewCommandSet(prodServeCmd, devServeCmd)
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
    may be used via the --webpack-server option. Similarly for vite with the
	--vite-dir and --vite-server options. Note, that only one of webpack or vite
	may be used at a time.

	If a self-signed cerificate is required, the cert command can be used to generate one.`)
}

func main() {
	cmdSet.MustDispatch(context.Background())
}

func serveContent(router chi.Router) {
	assets := webassets.NewAssets("webapp-sample/build/", webpackedAssets)
	router.Handle("/static/*", http.FileServer(http.FS(assets)))
	router.NotFound(serveIndexHTML(assets)) // serve index.html on all urls.
}

func prodServe(ctx context.Context, values any, _ []string) error {
	ctx, done := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer done()
	cl := values.(*ProdServerFlags)

	router := chi.NewRouter()
	configureAPIEndpoints(router)
	serveContent(router)

	cfg, err := cl.HTTPServerConfig().TLSConfig()
	if err != nil {
		return err
	}

	ln, srv, err := webapp.NewTLSServer(ctx, cl.Address, router, cfg)
	if err != nil {
		return err
	}

	// Force all http traffic to an https port.
	if err := webapp.RedirectPort80(ctx, webapp.RedirectToHTTPSPort(cl.Address)); err != nil {
		return err
	}

	log.Printf("running on %s", ln.Addr())

	return webapp.ServeTLSWithShutdown(ctx, ln, srv, 5*time.Second)
}

func devServe(ctx context.Context, values any, _ []string) error {
	ctx, done := signal.NotifyContext(ctx, os.Interrupt, os.Kill)
	defer done()
	cl := values.(*DevServerFlags)
	router := chi.NewRouter()
	configureAPIEndpoints(router)

	cfg, err := cl.HTTPServerConfig().TLSConfig()
	if err != nil {
		return err
	}

	ln, srv, err := webapp.NewTLSServer(ctx, cl.Address, router, cfg)
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
		dsURL, err = runWebpackDevServer(ctx, cl.WebpackDir)
		if err == nil {
			routeToProxy(router, "/build/", dsURL)
		}
	case len(cl.ViteServer) > 0:
		dsURL, err = url.Parse(cl.ViteServer)
		if err == nil {
			routeToProxy(router, "/build/", dsURL)
		}
	case len(cl.ViteDir) > 0:
		dsURL, err = runViteDevServer(ctx, cl.ViteDir)
		if err == nil {
			routeToProxy(router, "/build/", dsURL)
		}
	default:
		serveContent(router)
	}
	if err != nil {
		return err
	}

	log.Printf("running on %s", ln.Addr())
	return webapp.ServeTLSWithShutdown(ctx, ln, srv, 5*time.Second)
}

func configureAPIEndpoints(router chi.Router) {
	router.Get("/hello", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "hello")
	})
}

func serveIndexHTML(fsys fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		code, err := webassets.ServeFile(w, fsys, "index.html")
		w.WriteHeader(code)
		if err != nil {
			log.Printf("index.html: %v %v", code, err)
		}
	}
}

func runWebpackDevServer(ctx context.Context, webpackDir string) (*url.URL, error) {
	wpsrv := devserver.NewServer(ctx, webpackDir, "yarn", "start")
	log.Printf("starting webpack dev server in %q\n", webpackDir)
	return wpsrv.StartAndWaitForURL(ctx, os.Stdout, devserver.NewWebpackURLExtractor(nil))
}

func runViteDevServer(ctx context.Context, viteDir string) (*url.URL, error) {
	vitesrv := devserver.NewServer(ctx, viteDir, "npm", "run", "dev", "--", "--host")
	log.Printf("starting vite dev server in %q\n", viteDir)
	return vitesrv.StartAndWaitForURL(ctx, os.Stdout, devserver.NewViteURLExtractor(nil))
}

func routeToProxy(router chi.Router, _ string, url *url.URL) {
	// TODO: understand what publicPath (now _), was intended for, since
	// it's set to /build/ on the call sites, but overridden to / here.
	proxy := httputil.NewSingleHostReverseProxy(url)
	// Proxy websockets to allow for fast refresh - ie. changes made
	// in the javascript react code is immediately reflected in the UI
	// without a page reload.
	router.Get("/sockjs-node", http.HandlerFunc(proxy.ServeHTTP))
	router.Post("/sockjs-node", http.HandlerFunc(proxy.ServeHTTP))
	router.Get("/", http.HandlerFunc(proxy.ServeHTTP))
	router.NotFound(http.HandlerFunc(proxy.ServeHTTP))
}
