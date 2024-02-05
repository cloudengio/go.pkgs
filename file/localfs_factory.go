// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import (
	"context"
)

type LocalFSFactory struct{}

func (f *LocalFSFactory) NewFS(_ context.Context) (FS, error) {
	return &Local{}, nil
}

func (f *LocalFSFactory) NewObjectFS(_ context.Context) (ObjectFS, error) {
	return &Local{}, nil
}
