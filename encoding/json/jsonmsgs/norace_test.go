// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build !race

package jsonmsgs_test

// raceEnabled is true if the race detector is, which allocates more.
const raceEnabled = false
