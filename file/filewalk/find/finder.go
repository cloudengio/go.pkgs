// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package find

import (
	"context"
	"fmt"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
	"cloudeng.io/file/matcher"
)

func New(fs filewalk.FS, prefixMatcher, fileMatcher matcher.T, prune bool) filewalk.Handler[struct{}] {
	return &handler{
		fs:       fs,
		pm:       prefixMatcher,
		fm:       fileMatcher,
		prune:    prune,
		needstat: fileMatcher.NeedsFileMode() || fileMatcher.NeedsModTime(),
	}
}

type handler struct {
	fs       filewalk.FS
	pm, fm   matcher.T
	needstat bool
	prune    bool
}

type val struct {
	file.Info
}

func (h *handler) Prefix(_ context.Context, _ *struct{}, prefix string, fi file.Info, err error) (bool, file.InfoList, error) {
	if err != nil {
		return false, nil, err
	}
	if h.pm.Eval(val{fi}) {
		fmt.Println(prefix)
		return h.prune, nil, nil
	}
	return false, nil, nil
}

func (h *handler) Contents(ctx context.Context, _ *struct{}, prefix string, contents []filewalk.Entry) (file.InfoList, error) {
	children := make(file.InfoList, 0, len(contents))
	for _, c := range contents {
		filename := h.fs.Join(prefix, c.Name)
		var fi file.Info
		if h.needstat || c.IsDir() {
			var err error
			fi, err = h.fs.Stat(ctx, filename)
			if err != nil {
				return nil, err
			}
		}
		if h.fm.Eval(val{fi}) {
			fmt.Println(filename)
		}
		if c.IsDir() {
			children = append(children, fi)
		}
	}
	return children, nil
}

func (h *handler) Done(_ context.Context, _ *struct{}, _ string, _ error) error {
	return nil
}
