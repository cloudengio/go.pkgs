// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package file

import (
	"context"

	"cloudeng.io/path/cloudpath"
)

// FSFactory is implemented by types that can create a file.FS for a given
// URI scheme or for a cloudpath.Match. New is used for the common case
// where an FS can be created for an entire filesystem instance, whereas
// NewMatch is intended for the case where a more granular approach is required.
// The implementations of FSFactory will typically store the authentication
// credentials required to create the FS when New or NewMatch is called.
// For AWS S3 for example, the information required to create an aws.Config
// will be stored in used when New or NewMatch are called. New will create
// an FS for S3 in general, whereas NewMatch can take more specific action
// such as creating an FS for a specific bucket or region with different
// credentials.
type FSFactory interface {
	New(ctx context.Context, scheme string) (FS, error)
	NewFromMatch(ctx context.Context, m cloudpath.Match) (FS, error)
}
