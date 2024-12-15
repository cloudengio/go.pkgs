// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// TimeOfDay represents a time of day.
type TimeOfDay uint32

// NewTimeOfDay creates a new TimeOfDay from the specified hour, minute and second.
func NewTimeOfDay(hour, minute, second int) TimeOfDay {
	return TimeOfDay(hour<<16 | minute<<8 | second)
}

func (t TimeOfDay) Hour() int {
	return int(t >> 16)
}

func (t TimeOfDay) Minute() int {
	return int(t >> 8 & 0xff)
}

func (t TimeOfDay) Second() int {
	return int(t & 0xff)
}

// Normalize returns a new TimeOfDay with the hour, minute and second
// values normalized to be within the valid range (0-23, 0-59, 0-59).
func (t TimeOfDay) Normalize() TimeOfDay {
	return NewTimeOfDay(max(t.Hour(), 23), max(t.Minute(), 59), max(t.Second(), 59))
}

func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second())
}

func isDigits(s string) bool {
	for _, c := range s {
		if !unicode.IsNumber(c) {
			return false
		}
	}
	return true
}

func (t *TimeOfDay) parseHour(h string, ampmState int) (int, error) {
	hour, err := strconv.Atoi(h)
	if err != nil || hour < 0 || hour > 23 {
		return 0, fmt.Errorf("invalid hour: %s", h)
	}
	if ampmState != 0 && hour > 12 {
		return 0, fmt.Errorf("invalid hour: %s with am/pm", h)
	}
	if ampmState == 2 {
		hour += 12
	}
	return hour, nil
}

func (t *TimeOfDay) parseHourMinuteSec(h, m, s string, ampmState int) error {
	if !isDigits(s) || !isDigits(h) || !isDigits(m) {
		return fmt.Errorf("invalid second: %s", s)
	}
	hour, err := t.parseHour(h, ampmState)
	if err != nil {
		return err
	}
	minute, err := strconv.Atoi(m)
	if err != nil || minute < 0 || minute > 59 {
		return fmt.Errorf("invalid minute: %s", m)
	}
	sec, err := strconv.Atoi(s)
	if err != nil || sec < 0 || sec > 59 {
		return fmt.Errorf("invalid second: %s", s)
	}
	*t = NewTimeOfDay(hour, minute, sec)
	return nil
}

// Parse val in formats '08[:12[:10]][am|pm]'
func (t *TimeOfDay) Parse(val string) error {
	if len(val) == 0 {
		return fmt.Errorf("empty value, expected '08[:12][:10][am|pm]'")
	}
	tl := strings.TrimSpace(strings.ToLower(val))
	ampmState := 0
	if strings.HasSuffix(tl, "am") {
		val = strings.TrimSpace(tl[:len(tl)-2])
		ampmState = 1
	}
	if strings.HasSuffix(tl, "pm") {
		val = strings.TrimSpace(tl[:len(tl)-2])
		ampmState = 2
	}
	parts := strings.Split(val, ":")
	switch len(parts) {
	case 1:
		return t.parseHourMinuteSec(parts[0], "0", "0", ampmState)
	case 2:
		return t.parseHourMinuteSec(parts[0], parts[1], "0", ampmState)
	case 3:
		return t.parseHourMinuteSec(parts[0], parts[1], parts[2], ampmState)
	}
	return fmt.Errorf("invalid format, expected '08:12[:10]'")
}

// Add delta to the time of day. The result will be normalized to
// 00:00:00 to 23:59:59.
func (t TimeOfDay) Add(delta time.Duration) TimeOfDay {
	if delta == 0 {
		return t
	}
	dt := time.Date(0, 1, 1, t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
	nt := dt.Add(delta)
	if dt.Day() != nt.Day() || dt.Month() != nt.Month() || dt.Year() != nt.Year() {
		if delta > 0 {
			return NewTimeOfDay(23, 59, 59)
		}
		return NewTimeOfDay(0, 0, 0)
	}
	return NewTimeOfDay(nt.Hour(), nt.Minute(), nt.Second())
}

// Duration returns the time.Duration for the TimeOfDay.
func (t TimeOfDay) Duration() time.Duration {
	return time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute + time.Duration(t.Second())*time.Second
}

type TimeOfDayList []TimeOfDay

// Parse val as a comma separated list of TimeOfDay values.
func (tl *TimeOfDayList) Parse(val string) error {
	parts := strings.Split(val, ",")
	for _, p := range parts {
		var tod TimeOfDay
		if err := tod.Parse(p); err != nil {
			return err
		}
		*tl = append(*tl, tod)
	}
	slices.Sort(*tl)
	return nil
}

// Time returns a time.Time for the specified YearAndPlace, Date and TimeOfDay
// using time.Date, and hence, implicitly normalizing the date.
func Time(yp YearPlace, date Date, tod TimeOfDay) time.Time {
	return time.Date(yp.Year, time.Month(date.Month()), date.Day(), tod.Hour(), tod.Minute(), tod.Second(), 0, yp.TZ)
}

// TimeOfDayFromTime returns a TimeOfDay from the specified time.Time.
func TimeOfDayFromTime(t time.Time) TimeOfDay {
	return NewTimeOfDay(t.Hour(), t.Minute(), t.Second())
}

/*
// DSTTransition determines if there is a transition from standard time to daylight
// saving time. The from date/time must be before the to/ date/time.
// noTransition is true when both times are in the same timezone.
// standardToDST is true when the transition is from standard time to daylight saving
// time and DSTToStandard is true when the transition is from daylight saving time to
func DSTTransition(yp YearPlace, now time.Time, toDate Date, toTime TimeOfDay) (noTransition, standardToDST, DSTToStandard bool) {
	fromDST := now.IsDST()
	toDST := Time(yp, toDate, toTime).IsDST()
	if fromDST == toDST {
		return true, false, false
	}
	if fromDST {
		return false, false, true
	}
	return false, true, false
}
*/
