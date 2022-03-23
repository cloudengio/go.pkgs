// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !unix

package filewalk

func dumpMemStats(size int) error {
	return nil
}
