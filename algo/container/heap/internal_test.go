// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package heap //nolint:revive // intentional shadowing

import (
	"fmt"
	"testing"
)

func (h *T[K, V]) Verify(t *testing.T) {
	t.Helper()
	h.verify(t, 0)
}

func (h *T[K, V]) verify(t *testing.T, p int) {
	t.Helper()
	n := len(h.Keys)
	l, r := (2*p)+1, (2*p)+2
	if l < n {
		if h.less(l, p) {
			t.Errorf("heap inconsistent: left sub tree for %v (%v > [%v]: %v)", p, h.Keys[p], l, h.Keys[l])
			return
		}
		h.verify(t, l)
	}
	if r < n {
		if h.less(r, p) {
			t.Errorf("heap inconsistent: right sub tree for %v (%v > [%v]: %v)", p, h.Keys[p], r, h.Keys[r])
			return
		}
		h.verify(t, r)
	}
}

func (h *MinMax[K, V]) q1(i int) (bool, string) {
	gp := ((i + 1) / 4) - 1
	lnode := (2 * gp) + 1
	if lnode >= len(h.Keys) {
		return true, ""
	}
	return h.Keys[i] >= h.Keys[lnode], fmt.Sprintf("[%v] %v >= [%v] %v", i, h.Keys[i], lnode, h.Keys[lnode])
}

func (h *MinMax[K, V]) q2(i int) (bool, string) {
	gp := ((i + 1) / 4) - 1
	rnode := (2 * gp) + 2
	if rnode >= len(h.Keys) {
		return true, ""
	}
	return h.Keys[i] <= h.Keys[rnode], fmt.Sprintf("[%v] %v >= [%v] %v", i, h.Keys[i], rnode, h.Keys[rnode])
}

func (h *MinMax[K, V]) VerifyQ(t *testing.T) bool {
	t.Helper()
	switch len(h.Keys) {
	case 0:
		t.Errorf("SMM inconsistent: missing dummy root")
		return false
	case 1, 2:
		return true
	case 3:
		if h.Keys[1] > h.Keys[2] {
			t.Errorf("SMM inconsistent: [1] %v > [2] %v", h.Keys[1], h.Keys[2])
			return false
		}
		return true
	}
	for i := 3; i < len(h.Keys); i++ {
		if ok, msg := h.q1(i); !ok {
			t.Errorf("SMM inconsistent: Q1 failed for [%v] %v: %v", i, h.Keys[i], msg)
			return false
		}
		if ok, msg := h.q2(i); !ok {
			t.Errorf("SMM inconsistent: Q2 failed for [%v] %v: %v", i, h.Keys[i], msg)
			return false
		}
	}
	return true
}

func (h *MinMax[K, V]) Verify(t *testing.T) {
	t.Helper()
	h.VerifyQ(t)
}

func Pretty[K Ordered](k []K) {
	pretty(k)
}
