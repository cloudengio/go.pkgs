// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package schedule

import (
	"iter"

	"cloudeng.io/datetime"
)

// AnnualScheduler provides a way to iterate over the scheduled actions for a
// single year.
type AnnualScheduler[T any] struct {
	schedule Annual[T]
}

// NewAnnualScheduler returns a new annual scheduler with the supplied schedule.
func NewAnnualScheduler[T any](schedule Annual[T]) *AnnualScheduler[T] {
	scheduler := &AnnualScheduler[T]{
		schedule: schedule.clone(),
	}
	return scheduler
}

// Scheduled returns an iterator over the scheduled actions for the given year
// and place that returns all of the scheduled actions for each day that has
// scheduled Actions. It will evaluate any dynamic due times and sort the
// actions by their evaluated due time.
func (s *AnnualScheduler[T]) Scheduled(yp datetime.YearPlace, dates Dates, bounds datetime.DateRange) iter.Seq[Scheduled[T]] {
	drl := dates.EvaluateDateRanges(yp.Year, bounds)
	return func(yield func(Scheduled[T]) bool) {
		for _, dr := range drl {
			for day := range dr.Dates(yp.Year) {
				evaluated := s.schedule.DailyActions.Evaluate(day, yp.Place)
				evaluated.Sort()
				if !yield(Scheduled[T]{Date: day, Specs: evaluated}) {
					return
				}
			}
		}
	}
}
