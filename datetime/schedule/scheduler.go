// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package schedule

import (
	"iter"
	"slices"
	"sort"

	"cloudeng.io/datetime"
)

type AnnualScheduler[T any] struct {
	schedule       Annual[T]
	actionsStorage [2][]Action[T]
	actionStorage  [2]Action[T]
	actions        [][]Action[T]
}

func NewAnnualScheduler[T any](schedule Annual[T]) *AnnualScheduler[T] {
	scheduler := &AnnualScheduler[T]{
		schedule: schedule.clone(),
	}
	scheduler.prepareActions()
	return scheduler
}

func (s *AnnualScheduler[T]) prepareActions() {
	s.actions = s.actionsStorage[:0]
	if len(s.schedule.Actions) == 0 {
		return
	}
	if len(s.schedule.Actions) == 1 {
		s.actions = s.actionsStorage[:1]
		s.actions[0] = s.actionStorage[:1]
		s.actions[0][0] = s.schedule.Actions[0]
		return
	}
	merged := slices.Clone(s.schedule.Actions)
	sort.Slice(merged, func(i, j int) bool {
		if merged[i].Due == merged[j].Due {
			return merged[i].Name < merged[j].Name
		}
		return merged[i].Due < merged[j].Due
	})
	s.actions = s.actionsStorage[:1]
	s.actions[0] = s.actionStorage[:1]
	s.actions[0][0] = merged[0]
	for i := 1; i < len(merged); i++ {
		if merged[i].Due == merged[i-1].Due {
			s.actions[len(s.actions)-1] = append(s.actions[len(s.actions)-1], merged[i])
			continue
		}
		s.actions = append(s.actions, []Action[T]{merged[i]})
	}
	for i := range s.actions {
		sort.Slice(s.actions[i], func(j, k int) bool {
			return s.actions[i][j].Name < s.actions[i][k].Name
		})
	}
}

// Scheduled returns an iterator over the scheduled actions for the given year and place.
func (s AnnualScheduler[T]) Scheduled(yp datetime.YearAndPlace) iter.Seq[Active[T]] {
	drl := s.schedule.Dates.EvaluateDateRanges(yp.Year)
	return func(yield func(Active[T]) bool) {
		for _, dr := range drl {
			for day := range dr.Dates(yp.Year) {
				for _, da := range s.actions {
					if !yield(Active[T]{Date: day.Date(), Actions: da}) {
						return
					}
				}
			}
		}
	}
}
