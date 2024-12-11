// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package schedule

import (
	"time"

	"cloudeng.io/algo/container/heap"
	"cloudeng.io/datetime"
)

type heapEntry[T any] struct {
	name       string
	repeat     time.Duration
	bounded    bool
	numRepeats int
	t          T
}

type repeatManager[T any] struct {
	h *heap.T[int64, heapEntry[T]]
}

func newRepeatManager[T any](actions ActionSpecs[T], cd datetime.CalendarDate, place datetime.Place) *repeatManager[T] {
	evaluated := actions.Evaluate(cd, place)
	h := heap.NewMin(heap.WithSliceCap[int64, heapEntry[T]](len(actions)))
	rm := &repeatManager[T]{h: h}
	for _, a := range evaluated {
		he := heapEntry[T]{
			name:       a.Name,
			repeat:     a.Repeat.Interval,
			numRepeats: a.Repeat.Repeats,
			t:          a.T,
		}
		if a.Repeat.Interval != 0 && a.Repeat.Repeats != 0 {
			he.bounded = true
		}
		rm.h.Push(cd.Time(a.Due, place.TZ).Unix(), he)
	}
	return rm
}

func (rm *repeatManager[T]) hasActions() bool {
	return rm.h.Len() > 0
}

func (rm *repeatManager[T]) manage(loc *time.Location) (time.Time, heapEntry[T]) {
	secs, he := rm.h.Pop()
	when := time.Unix(secs, 0).In(loc)
	//fmt.Printf("% 8v: pop %v (%v)\n", he.name, secs, he.repeat)
	if he.repeat == 0 || (he.bounded && he.numRepeats == 0) {
		return when, he
	}
	nextTime := when.Add(he.repeat)
	if nextTime.Day() != when.Day() ||
		nextTime.Month() != when.Month() ||
		nextTime.Year() != when.Year() {
		// The next time would be on a different day.
		return when, he
	}
	if he.bounded {
		he.numRepeats--
	}
	//fmt.Printf("% 8v: push %v\n", he.name, nextTime)
	rm.h.Push(nextTime.Unix(), he)
	return when, he
}
