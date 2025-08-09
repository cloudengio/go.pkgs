// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package devtest

import (
	"encoding/json"
	"html/template"
	"net/http"
)

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.Title}}</title>
</head>
<body>
    <h1>Check the browser console in developer tools windows!</h1>
    <p>This page dynamically loads multiple scripts.</p>
	 {{range .Scripts}}
        <p>{{.}}</p>
    {{end}}

    {{range .Scripts}}
        <script src="{{.}}"></script>
    {{end}}
</body>
</html>
`

var htmlTpl = template.Must(template.New("javascript testing").Parse(htmlTemplate))

// JSServer provides a http.Handler for serving JavaScript files
// using a simple template that executes each file in turn.
// An optional TypescriptSources can be provided to compile TypeScript
// files before generating the http response.
type JSServer struct {
	title     string
	jsScripts []string
	ts        *TypescriptSources
}

func NewJSServer(title string, ts *TypescriptSources, jsScripts ...string) *JSServer {
	return &JSServer{
		title:     title,
		jsScripts: jsScripts,
		ts:        ts,
	}
}

func (jss *JSServer) writeError(rw http.ResponseWriter, status int, err error) {
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(status)
	json.NewEncoder(rw).Encode(err.Error()) //nolint:errcheck
}

// ServeJS handles HTTP requests for serving a series of Javascript
// files.
func (jss *JSServer) ServeJS(rw http.ResponseWriter, r *http.Request) {
	if jss.ts != nil {
		if err := jss.ts.Compile(r.Context()); err != nil {
			jss.writeError(rw, http.StatusInternalServerError, err)
			return
		}
	}
	data := struct {
		Title   string
		Scripts []string
	}{
		Title:   jss.title,
		Scripts: jss.jsScripts,
	}
	rw.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := htmlTpl.Execute(rw, data); err != nil {
		jss.writeError(rw, http.StatusInternalServerError, err)
	}
}
