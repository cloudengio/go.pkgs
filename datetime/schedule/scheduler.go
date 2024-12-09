// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package schedule

import (
	"iter"

	"cloudeng.io/datetime"
)

type AnnualScheduler[T any] struct {
	schedule Annual[T]
}

func NewAnnualScheduler[T any](schedule Annual[T]) *AnnualScheduler[T] {
	scheduler := &AnnualScheduler[T]{
		schedule: schedule.clone(),
	}
	scheduler.schedule.Actions.Sort()
	return scheduler
}

// Scheduled returns an iterator over the scheduled actions for the given year
// and place that returns all of the scheduled actions for each day that has
// scheduled Actions.
func (s AnnualScheduler[T]) Scheduled(yp datetime.YearPlace) iter.Seq[Active[T]] {
	drl := s.schedule.Dates.EvaluateDateRanges(yp.Year)
	return func(yield func(Active[T]) bool) {
		for _, dr := range drl {
			for day := range dr.Dates(yp.Year) {
				if !yield(Active[T]{Date: day.Date(), Actions: s.schedule.Actions}) {
					return
				}
			}
		}
	}
}
