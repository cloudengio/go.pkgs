// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import (
	"context"
)

// FSFactory is implemented by types that can create a file.FS for a given
// URI scheme. The implementations of FSFactory will typically store the
// authentication credentials required to create the FS when NewFS is called.
// For AWS S3 for example, the information required to create an aws.Config
// will be stored in used when NewFS is called.
type FSFactory interface {
	NewFS(ctx context.Context) (FS, error)
}

// ObjectFSFactory is like FSFactory but for ObjectFS.
type ObjectFSFactory interface {
	NewObjectFS(ctx context.Context) (ObjectFS, error)
}
