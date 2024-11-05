// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"fmt"
	"strings"
	"time"
)

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

func (cd CalendarDate) String() string {
	return fmt.Sprintf("%02d %02d %04d", time.Month(cd.Month), cd.Day, cd.Year)
}

type CalendarDateList []CalendarDate

func (cdl CalendarDateList) String() string {
	var out strings.Builder
	for i, d := range cdl {
		if i > 0 && i < len(cdl)-1 {
			out.WriteString(", ")
		}
		out.WriteString(fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day))
	}
	return out.String()
}

func (cdl CalendarDateList) Contains(d CalendarDate) bool {
	for _, cd := range cdl {
		if cd == d {
			return true
		}
	}
	return false
}
