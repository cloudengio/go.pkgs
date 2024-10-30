// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dates

// CalendarDate represents a date with a year, month and day. Day be zero in which
// it is interpreted as per Date.
// TODO: multiyear date ranges etc
type CalendarDate struct {
	Year  int
	Month Month
	Day   int
}

// Date returns the Date for the CalendarDate.
func (cd CalendarDate) Date() Date {
	return Date{cd.Month, cd.Day}
}
