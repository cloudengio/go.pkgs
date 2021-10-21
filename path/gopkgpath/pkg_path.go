// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package gopkgpath provides support for obtaining and working with
// go package paths when go modules are used. It does not support
// vendor or GOPATH configurations.
package gopkgpath

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/mod/modfile"
)

type pathCache struct {
	sync.Mutex
	paths map[string]string
}

func enclosingGoMod(dir string) (string, error) {
	for {
		gomodfile := filepath.Join(dir, "go.mod")
		if fi, err := os.Stat(gomodfile); err == nil && !fi.IsDir() {
			return dir, nil
		}
		d := filepath.Dir(dir)
		if d == dir {
			return "", fmt.Errorf("failed to find enclosing go.mod for dir %v", dir)
		}
		dir = d
	}
}

var pkgPathCache = pathCache{
	paths: make(map[string]string),
}

func (pc *pathCache) has(dir string) (string, bool) {
	pc.Lock()
	defer pc.Unlock()
	p, ok := pc.paths[dir]
	return p, ok
}

func (pc *pathCache) set(dir, pkg string) {
	pc.Lock()
	defer pc.Unlock()
	pc.paths[dir] = pkg
}

func (pc *pathCache) pkgPath(file string) (string, error) {
	dir := filepath.Clean(filepath.Dir(file))
	if p, ok := pc.has(dir); ok {
		return p, nil
	}
	root, err := enclosingGoMod(dir)
	if err != nil {
		return "", err
	}
	gomodfile := filepath.Join(root, "go.mod")
	gomod, err := ioutil.ReadFile(gomodfile)
	if err != nil {
		return "", err
	}
	module := modfile.ModulePath(gomod)
	if len(module) == 0 {
		return "", fmt.Errorf("failed to read module path from %v", gomodfile)
	}

	pkgPath := strings.TrimPrefix(dir, root)
	pkgPath = strings.ReplaceAll(pkgPath, string(filepath.Separator), "/")
	if !strings.HasPrefix(pkgPath, module) {
		pkgPath = path.Join(module, pkgPath)
	}
	pc.set(dir, pkgPath)
	return pkgPath, nil
}

// Type returns the package path for the type of the supplied argument.
// That type must be a defined/named type, anoymous types, function
// variables etc will return "".
func Type(v interface{}) string {
	return reflect.TypeOf(v).PkgPath()
}

// Caller is the same as CallerDepth(0).
func Caller() (string, error) {
	_, file, _, _ := runtime.Caller(1)
	return pkgPathCache.pkgPath(file)
}

// CallerDepth returns the package path of the caller at the specified
// depth where a depth of 0 is the immediate caller. It determines the
// module name by finding and parsing the enclosing go.mod file and as
// such requires that go modules are being used.
func CallerDepth(depth int) (string, error) {
	_, file, _, _ := runtime.Caller(depth + 1)
	return pkgPathCache.pkgPath(file)
}
