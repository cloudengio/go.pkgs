// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package datetime

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
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

func state0(designator byte, n float64) (time.Duration, error) {
	switch designator {
	case 'Y':
		return time.Duration(float64(time.Hour) * 24 * 365 * n), nil
	case 'M':
		return time.Duration((float64(time.Hour) * 24 * 365 * n) / 12), nil
	case 'W':
		return time.Duration(float64(time.Hour) * 24 * 7 * n), nil
	case 'D':
		return time.Duration(float64(time.Hour) * 24 * n), nil
	}
	return 0, fmt.Errorf("invalid duration designator: %c: %w", designator, ErrInvalidISO8601Duration)
}

func state1(designator byte, n float64) (time.Duration, error) {
	switch designator {
	case 'H':
		return time.Duration(float64(time.Hour) * n), nil
	case 'M':
		return time.Duration(float64(time.Minute) * n), nil
	case 'S':
		return time.Duration(float64(time.Second) * n), nil
	}
	return 0, fmt.Errorf("invalid duration designator: %c: %w", designator, ErrInvalidISO8601Duration)
}

// ParseISO8601Period parses an ISO 8601 'period' of the form [-]PnYnMnDTnHnMnS
func ParseISO8601Period(dur string) (time.Duration, error) {
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
			d, err := state0(designator, n)
			if err != nil {
				return 0, err
			}
			result += d
			continue
		case 1:
			d, err := state1(designator, n)
			if err != nil {
				return 0, err
			}
			result += d
		}
	}
	if nasNP {
		result = -result
	}
	return result, nil
}

func AsISO8601Period(dur time.Duration) string {
	var out strings.Builder
	if dur < 0 {
		out.WriteByte('-')
		dur = -dur
	}
	out.WriteByte('P')
	if years := dur / (time.Hour * 24 * 365); years > 0 {
		fmt.Fprintf(&out, "%dY", years)
		dur -= years * (time.Hour * 24 * 365)
	}
	if dur == 0 {
		return out.String()
	}
	if months := dur / ((time.Hour * 24 * 365) / 12); months > 0 {
		fmt.Fprintf(&out, "%dM", months)
		dur -= months * ((time.Hour * 24 * 365) / 12)
	}
	if dur == 0 {
		return out.String()
	}
	if weeks := dur / (time.Hour * 24 * 7); weeks > 0 {
		fmt.Fprintf(&out, "%dW", weeks)
		dur -= weeks * (time.Hour * 24 * 7)
	}
	if dur == 0 {
		return out.String()
	}
	if days := dur / (time.Hour * 24); days > 0 {
		fmt.Fprintf(&out, "%dD", days)
		dur -= days * (time.Hour * 24)
	}
	if dur == 0 {
		return out.String()
	}
	out.WriteByte('T')
	if hours := dur / time.Hour; hours > 0 {
		fmt.Fprintf(&out, "%dH", hours)
		dur -= hours * time.Hour
	}
	if dur == 0 {
		return out.String()
	}
	if minutes := dur / time.Minute; minutes > 0 {
		fmt.Fprintf(&out, "%dM", minutes)
		dur -= minutes * time.Minute
	}
	if dur == 0 {
		return out.String()
	}
	if seconds := dur / time.Second; seconds > 0 {
		fmt.Fprintf(&out, "%dS", seconds)
		dur -= seconds * time.Second
	}
	return out.String()
}
