// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dates

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// TimeOfDay represents a time of day.
type TimeOfDay struct {
	Hour   int
	Minute int
	Second int
}

func (t *TimeOfDay) parseHourMinute(h, m string) error {
	hour, err := strconv.Atoi(h)
	if err != nil || hour < 0 || hour > 23 {
		return fmt.Errorf("invalid hour: %s", h)
	}
	minute, err := strconv.Atoi(m)
	if err != nil || minute < 0 || minute > 59 {
		return fmt.Errorf("invalid minute: %s", m)
	}
	t.Hour = hour
	t.Minute = minute
	return nil
}

func (t *TimeOfDay) parseHourMinuteSec(h, m, s string) error {
	if err := t.parseHourMinute(h, m); err != nil {
		return err
	}
	sec, err := strconv.Atoi(s)
	if err != nil || sec < 0 || sec > 59 {
		return fmt.Errorf("invalid second: %s", s)
	}
	t.Second = sec
	return nil
}

// Parse val in formats '08:12[:10]' or '08-12[-10]'.
func (t *TimeOfDay) Parse(val string) error {
	if len(val) == 0 {
		return fmt.Errorf("empty value, expected '08:12[:10]' or '08-12[-10]'")
	}
	var parts []string
	if strings.Contains(val, ":") {
		parts = strings.Split(val, ":")
	} else if strings.Contains(val, "-") {
		parts = strings.Split(val, "-")
	}
	if len(parts) == 2 {
		return t.parseHourMinute(parts[0], parts[1])
	}
	if len(parts) == 3 {
		return t.parseHourMinuteSec(parts[0], parts[1], parts[2])
	}
	return fmt.Errorf("invalid format, expected '08:12[:10]' or '08-12[-10]'")
}

func (t *TimeOfDay) UnmarshalYAML(node *yaml.Node) error {
	return t.Parse(node.Value)
}

// Before returns true if t is before t2.
func (t TimeOfDay) Before(t2 TimeOfDay) bool {
	if t.Hour != t2.Hour {
		return t.Hour < t2.Hour
	}
	if t.Minute != t2.Minute {
		return t.Minute < t2.Minute
	}
	return t.Second < t2.Second
}

type TimeOfDayList []TimeOfDay

func (tl TimeOfDayList) Sort() {
	sort.Slice(tl, func(i, j int) bool { return tl[i].Before(tl[j]) })
}

func DateTime(yp YearAndPlace, date Date, tod TimeOfDay) time.Time {
	return time.Date(yp.Year, time.Month(date.Month), date.Day, tod.Hour, tod.Minute, tod.Second, 0, yp.Place)
}
