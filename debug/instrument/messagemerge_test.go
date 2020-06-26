package goroutine_test

import (
	"testing"
	"time"

	"cloudeng.io/debug/goroutine"
)

func isSorted(t *testing.T, mr goroutine.MessageRecords) {
	prev := mr[0]
	for i, cur := range mr[1:] {
		if prev.Time.After(cur.Time) {
			t.Errorf("younger recorded preceded by older one: %v: %v not younger than %v", i, prev, cur)
		}
		if prev.Time.Equal(cur.Time) {
			// wait, then receive then sent.
			switch {
			case cur.Status == goroutine.MessageWait:
				if prev.Status != goroutine.MessageWait {
					t.Errorf("wait preceeded by non wait: %v: %v %v", i, prev, cur)
				}
			case cur.Status == goroutine.MessageSent:
				if prev.Status == goroutine.MessageReceived {
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
		if got, want := cur.Name, "A"; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	isSorted(t, fl1)
	isSorted(t, fl2)
	isSorted(t, fl3)

	merged := goroutine.MergeMesageTraces(fl1, fl2, fl3)
	if got, want := len(merged), len(fl1)*3; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	mt4 := &goroutine.MessageTrace{}
	mt5 := &goroutine.MessageTrace{}
	mt6 := &goroutine.MessageTrace{}
	for _, mt := range []*goroutine.MessageTrace{mt4, mt5, mt6} {
		now := time.Now()
		mt.Log(1, goroutine.MessageReceived, localAddr, remoteAddr, "rx")
		mt.Log(1, goroutine.MessageSent, localAddr, remoteAddr, "tx")
		mt.Log(1, goroutine.MessageWait, localAddr, remoteAddr, "wait")
		goroutine.SetTime(mt, now)
	}

	merged = goroutine.MergeMesageTraces(
		mt4.Flatten("a"),
		mt5.Flatten("b"),
		mt6.Flatten("c"))
	isSorted(t, merged)
}
