// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package instrument_test

import (
	"reflect"
	"runtime"
	"testing"
	"time"

	"cloudeng.io/debug/instrument"
)

func isSorted(t *testing.T, mr instrument.MessageRecords) {
	_, _, line, _ := runtime.Caller(1)
	prev := mr[0]
	for i, cur := range mr[1:] {
		switch {
		case cur.Level < prev.Level:
			t.Errorf("line %v: current level is lower than previous one (%v):\n\tprev = %v\n\tcurr = %v", line, i, prev, cur)
		case cur.RootID < prev.RootID:
			t.Errorf("line %v: current rootID is lower than previous one (%v):\n\tprev = %v\n\tcurr = %v", line, i, prev, cur)
		case cur.ID < prev.ID:
			t.Errorf("line %v: current ID is lower than previous one (%v):\n\tprev = %v\n\tcurr = %v", line, i, prev, cur)
		}
		if cur.ID == prev.ID {
			switch {
			case prev.Time.After(cur.Time):
				t.Errorf("line %v: younger recorded preceded by older one: %v: %v not younger than %v", line, i, prev, cur)
			case prev.Time.Equal(cur.Time):
				switch cur.Status {
				case instrument.MessageWait:
					if prev.Status != instrument.MessageWait {
						t.Errorf("line %v: wait preceded by non wait: %v: %v %v", line, i, prev, cur)
					}
				case instrument.MessageSent:
					if prev.Status == instrument.MessageReceived {
						t.Errorf("line %v: received preceded by sent: %v: %v %v", line, i, prev, cur)
					}
				}
			}
		}
		prev = cur
	}
}

func TestFlattenAndMerge(t *testing.T) {
	mt1, mt2, mt3 := generateMessageTrace(), generateMessageTrace(), generateMessageTrace()
	fl1, fl2, fl3 := mt1.Flatten("A"), mt2.Flatten("B"), mt3.Flatten("C")

	for _, cur := range fl1 {
		if got, want := cur.Tag, "A"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	isSorted(t, fl1)
	isSorted(t, fl2)
	isSorted(t, fl3)

	empty := instrument.MessageRecords{}
	if got, want := instrument.MergeMessageTraces(), empty; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	merged := instrument.MergeMessageTraces(fl1)
	if got, want := merged, fl1; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	merged = instrument.MergeMessageTraces(fl1, fl2, fl3)
	if got, want := len(merged), len(fl1)*3; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := sanitizeString(merged.String()), `A: 172.16.1.1 -> 172.16.1.2: first
A: 172.16.1.1 <? 172.16.1.2: waiting
A: 172.16.1.1 <? 172.16.1.2: waiting
A: 172.16.1.1 <? 172.16.1.2: waiting
A: 172.16.1.1 <? 172.16.1.2: waiting
B: 172.16.1.1 -> 172.16.1.2: first
B: 172.16.1.1 <? 172.16.1.2: waiting
B: 172.16.1.1 <? 172.16.1.2: waiting
B: 172.16.1.1 <? 172.16.1.2: waiting
B: 172.16.1.1 <? 172.16.1.2: waiting
C: 172.16.1.1 -> 172.16.1.2: first
C: 172.16.1.1 <? 172.16.1.2: waiting
C: 172.16.1.1 <? 172.16.1.2: waiting
C: 172.16.1.1 <? 172.16.1.2: waiting
C: 172.16.1.1 <? 172.16.1.2: waiting
`; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	mt4 := &instrument.MessageTrace{}
	mt5 := &instrument.MessageTrace{}
	mt6 := &instrument.MessageTrace{}
	for _, mt := range []*instrument.MessageTrace{mt4, mt5, mt6} {
		now := time.Now()
		mt.Log(1, instrument.MessageReceived, localAddr, remoteAddr, "rx")
		mt.Log(1, instrument.MessageSent, localAddr, remoteAddr, "tx")
		mt.Log(1, instrument.MessageWait, localAddr, remoteAddr, "wait")
		instrument.SetTime(mt, now)
	}

	all := instrument.MergeMessageTraces(
		mt4.Flatten("a"),
		mt5.Flatten("b"),
		mt6.Flatten("c"))
	isSorted(t, all)

	ab := instrument.MergeMessageTraces(
		mt4.Flatten("a"),
		mt5.Flatten("b"))
	isSorted(t, ab)
	if got, want := instrument.MergeMessageTraces(ab, mt6.Flatten("c")), all; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := instrument.MergeMessageTraces(mt6.Flatten("c"), ab), all; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
