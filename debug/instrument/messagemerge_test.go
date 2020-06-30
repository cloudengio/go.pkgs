package instrument_test

import (
	"reflect"
	"testing"
	"time"

	"cloudeng.io/debug/instrument"
)

func isSorted(t *testing.T, mr instrument.MessageRecords) {
	prev := mr[0]
	for i, cur := range mr[1:] {
		if prev.Time.After(cur.Time) {
			t.Errorf("younger recorded preceded by older one: %v: %v not younger than %v", i, prev, cur)
		}
		if prev.Time.Equal(cur.Time) {
			// wait, then receive then sent.
			switch {
			case cur.Status == instrument.MessageWait:
				if prev.Status != instrument.MessageWait {
					t.Errorf("wait preceeded by non wait: %v: %v %v", i, prev, cur)
				}
			case cur.Status == instrument.MessageSent:
				if prev.Status == instrument.MessageReceived {
					t.Errorf("received preceeded by sent: %v: %v %v", i, prev, cur)
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
	if got, want := instrument.MergeMesageTraces(), empty; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	merged := instrument.MergeMesageTraces(fl1)
	if got, want := merged, fl1; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	merged = instrument.MergeMesageTraces(fl1, fl2, fl3)
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

	all := instrument.MergeMesageTraces(
		mt4.Flatten("a"),
		mt5.Flatten("b"),
		mt6.Flatten("c"))
	isSorted(t, all)

	ab := instrument.MergeMesageTraces(
		mt4.Flatten("a"),
		mt5.Flatten("b"))
	isSorted(t, ab)
	if got, want := instrument.MergeMesageTraces(ab, mt6.Flatten("c")), all; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, want := instrument.MergeMesageTraces(mt6.Flatten("c"), ab), all; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
