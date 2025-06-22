// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package bitmap

func NextValueForTesting(c *Contiguous) int {
	if c == nil {
		return -1
	}
	return c.next
}
