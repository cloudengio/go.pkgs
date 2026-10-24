// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cmdyaml

import (
	"context"
	"fmt"

	"cloudeng.io/cmdutil/cmdconfig"
	"cloudeng.io/cmdutil/envfile"
)

// Load locates the files held by ff, as per spec, and parses the YAML files
// among them into cfg. Parsing is strict, see WithStrictFields, and $VAR and
// ${VAR} references in the YAML are expanded from the process environment
// and, for any variable that it does not define, from the environment
// variables files, see envfile.ExpandingMapping; where several define a
// variable the later file takes precedence. opts are applied after, and so can
// override, those. The files that were located are returned, even if there
// was an error parsing them, so that they can be reported, but not if there
// was an error locating them.
//
// It is not an error for there to be no YAML file, in which case cfg is not
// modified: a caller for which one is required should check the result.
func Load(ctx context.Context, cfg any, ff *cmdconfig.FilesFlag, spec cmdconfig.Spec, opts ...Option) (*cmdconfig.Located, error) {
	loc, err := ff.Locate(spec)
	if err != nil {
		return nil, err
	}
	return loc, LoadFiles(ctx, cfg, loc, opts...)
}

// LoadFiles is like Load for files that have already been located. Callers
// can add to or edit the files in loc first, for example to add a YAML file
// that is not named by a FilesFlag.
func LoadFiles(ctx context.Context, cfg any, loc *cmdconfig.Located, opts ...Option) error {
	vars := &envfile.Variables{}
	for _, file := range loc.Env {
		if err := vars.MergeFile(file); err != nil {
			return fmt.Errorf("read environment variables from %v: %w", file, err)
		}
	}
	if len(loc.YAML) == 0 {
		return nil
	}
	opts = append([]Option{WithStrictFields(true), WithExpandMapping(envfile.ExpandingMapping(vars))}, opts...)
	return NewParser(opts...).ParseFiles(ctx, cfg, loc.YAML...)
}
