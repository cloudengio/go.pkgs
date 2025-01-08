// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

var (
	multipliers = []time.Duration{
		time.Hour * 24 * 365,        // Y
		(time.Hour * 24 * 365) / 12, // M
		time.Hour * 24 * 7,          // W
		time.Hour * 24,              // D
		time.Hour,                   // H
		time.Minute,                 // M
		time.Second,                 // S
	}
)

type identifier int

const (
	year identifier = iota
	month
	week
	day
	hour
	minute
	second
)

var ErrInvalidISO8601Duration = errors.New("invalid ISO8601 duration")

func consumeN(dur string) (float64, byte, int, error) {
	for i := range dur {
		c := dur[i]
		if (c >= '0' && c <= '9') || c == '.' {
			continue
		}
		switch c {
		case 'Y', 'M', 'W', 'D', 'H', 'S':
			n, err := strconv.ParseFloat(dur[:i], 64)
			if err != nil {
				return 0, 0, 0, fmt.Errorf("invalid number: %q: %q: %w", dur[:i], dur, ErrInvalidISO8601Duration)
			}
			return n, c, i + 1, nil
		}
		break
	}
	return 0, 0, 0, fmt.Errorf("invalid number or duration designator: %s: %w", dur, ErrInvalidISO8601Duration)
}

// ParseISO8601Duration parses a duration string in the ISO8601 format.
// [-]PnYnMnDTnHnMnS
func ParseISO8601Duration(dur string) (time.Duration, error) {
	nl := len(dur)
	hasP, nasNP := (nl > 0 && dur[0] == 'P'), (nl > 1 && dur[0] == '-' && dur[1] == 'P')
	if !hasP && !nasNP {
		return 0, fmt.Errorf("duration must start with P or -P: %s: %w", dur, ErrInvalidISO8601Duration)
	}
	dur = dur[1:]
	if nasNP {
		dur = dur[1:]
	}

	var result time.Duration
	state := 0 // 0 = P, 1 = T
	for len(dur) > 0 {
		if dur[0] == 'T' {
			state = 1
			dur = dur[1:]
			continue
		}
		n, designator, idx, err := consumeN(dur)
		if err != nil {
			return 0, err
		}
		dur = dur[idx:]
		switch state {
		case 0:
			switch designator {
			case 'Y':
				result += time.Duration(float64(time.Hour) * 24 * 365 * n)
			case 'M':
				result += time.Duration((float64(time.Hour) * 24 * 365 * n) / 12)
			case 'W':
				result += time.Duration(float64(time.Hour) * 24 * 7 * n)
			case 'D':
				result += time.Duration(float64(time.Hour) * 24 * n)
			default:
				return 0, fmt.Errorf("invalid duration designator: %c: %w", designator, ErrInvalidISO8601Duration)
			}
			continue
		case 1:
			switch designator {
			case 'H':
				result += time.Duration(float64(time.Hour) * n)
			case 'M':
				result += time.Duration(float64(time.Minute) * n)
			case 'S':
				result += time.Duration(float64(time.Second) * n)
			default:
				return 0, fmt.Errorf("invalid duration designator: %c: %w", designator, ErrInvalidISO8601Duration)
			}
		}
	}
	if nasNP {
		result = -result
	}
	return result, nil
}
