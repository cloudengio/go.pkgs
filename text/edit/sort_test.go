package edit

import (
	"testing"
)

func TestSort(t *testing.T) {
	ins := Insert
	rpl := Replace
	del := Delete
	j := func(ds ...Delta) []Delta {
		return ds
	}
	same := func(a, b []Delta) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}
	for i, tc := range []struct {
		before, after []Delta
	}{
		{
			j(ins(11, "a"), ins(11, "b")),
			j(ins(11, "a"), ins(11, "b"))},
		{
			j(ins(11, "b"), ins(11, "a")),
			j(ins(11, "b"), ins(11, "a"))},
		{
			j(rpl(11, 2, "a"), rpl(11, 2, "b")),
			j(rpl(11, 2, "a"), rpl(11, 2, "b"))},
		{
			j(rpl(11, 2, "b"), rpl(11, 2, "a")),
			j(rpl(11, 2, "b"), rpl(11, 2, "a"))},
		{
			j(del(11, 2), del(11, 2)),
			j(del(11, 2), del(11, 2))},
		{
			j(del(11, 2), del(11, 2)),
			j(del(11, 2), del(11, 2))},
		{
			j(ins(11, "ii"), del(11, 2), rpl(11, 2, "rr")),
			j(del(11, 2), rpl(11, 2, "rr"), ins(11, "ii"))},
		{
			j(ins(11, "ii"), rpl(11, 2, "rr"), del(11, 2)),
			j(del(11, 2), rpl(11, 2, "rr"), ins(11, "ii"))},
		{
			j(rpl(11, 2, "rr"), ins(11, "ii"), del(11, 2)),
			j(del(11, 2), rpl(11, 2, "rr"), ins(11, "ii"))},
		{
			j(ins(11, "ii"), ins(11, "jj"), rpl(11, 2, "rr"), del(11, 2)),
			j(del(11, 2), rpl(11, 2, "rr"), ins(11, "ii"), ins(11, "jj"))},
		{
			j(ins(13, "ii"), ins(11, "jj"), rpl(11, 2, "rr"), del(11, 2)),
			j(del(11, 2), rpl(11, 2, "rr"), ins(11, "jj"), ins(13, "ii"))},
		{
			j(ins(11, "jj"), ins(13, "ii"), rpl(11, 2, "rr"), del(11, 2)),
			j(del(11, 2), rpl(11, 2, "rr"), ins(11, "jj"), ins(13, "ii"))},
		{
			j(ins(11, "jj"), rpl(12, 2, "rr"), del(13, 2)),
			j(ins(11, "jj"), rpl(12, 2, "rr"), del(13, 2))},
		{
			j(ins(13, "jj"), rpl(12, 2, "rr"), del(11, 2)),
			j(del(11, 2), rpl(12, 2, "rr"), ins(13, "jj"))},
		{
			j(ins(13, "jj"), rpl(11, 2, "rr"), del(12, 2)),
			j(rpl(11, 2, "rr"), del(12, 2), ins(13, "jj"))},
	} {
		sortDeltas(tc.before)
		if got, want := tc.before, tc.after; !same(got, want) {
			t.Errorf("%v: got %#v (%v), want %#v (%v)", i, got, got, want, want)
		}
	}
}

func TestOverwrite(t *testing.T) {
	for i, tc := range []struct {
		a, b, c string
	}{
		{"", "b", "b"},
		{"b", "", "b"},
		{"a", "b", "b"},
		{"aa", "b", "ba"},
		{"aa", "bb", "bb"},
		{"aa", "bbc", "bbc"},
	} {
		if got, want := overwrite(tc.a, tc.b), tc.c; got != want {
			t.Errorf("%v: got %v, want %v\n", i, got, want)
		}
	}
}
