// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package lcs provides implementations of algorithms to find the
// longest common subsequence/shortest edit script (LCS/SES) between two
// slices suitable for use with unicode/utf8 and other alphabets.
package lcs

// TODO(cnicolaou): improve DP implementation to use only one row+column to
// store lcs lengths rather than row * column.
// TODO(cnicolaou): improve the Myers implementation as described in
// An O(NP) Sequence Comparison Algorithm, Wu, Manber, Myers.
