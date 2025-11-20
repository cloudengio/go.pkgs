# [cloudeng.io/webapp/cmd/webapp](https://pkg.go.dev/cloudeng.io/webapp/cmd/webapp?tab=doc)


Usage of `webapp`

    Run a `webapp` server. Two modes are supported: production and development.

    For production, all assets are built into the production server's binary.

    For development, the front-end code/assets can be managed in two ways:

    1. with embedded assets that can be overridden with newer or new files found on
    the local filesystem. This is generally used when the javascript/`webapp` tooling
    only supports generating new assets rather than any form of dynamic update. The
    user must reload the site/page to see the new version.

    2. with a development server, such as that provided by webpack, that dynamically
    monitors the javascript/`webapp` code and dynamically rebuilds the assets. To use
    this mode, this application will proxy all of the urls that it doesn't itself
    implement to the running development server. The dev server may be started by
    this server via the --webpack-dir option. Alternatively, a running dev server
    may be used via the --webpack-server option. Similarly for vite with the --vite-dir
    and --vite-server options. Note, that only one of webpack or vite may be used
    at a time.

    If a self-signed cerificate is required, the cert command can be used to generate
    one.

    prod - run a production server.
     dev - run a development server.

This command is an example of how to create a simple `webapp` that uses react
for the browser-side app and serves API endpoints for use by that app.
See the comments and command line flag/help messages.

