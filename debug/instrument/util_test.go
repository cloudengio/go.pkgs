// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package instrument_test

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	captureLeaderRE = regexp.MustCompile(`([ ]*)\(([^)]+)\)[ ]*(.*)`)
	idsRE           = regexp.MustCompile(`(\d+)/(\d+)`)
)

func sanitizeString(s string) string {
	out := &strings.Builder{}
	sc := bufio.NewScanner(bytes.NewBufferString(s))
	for sc.Scan() {
		l := sc.Text()
		if strings.Contains(l, "testing.tRunner testing.go:") {
			fmt.Fprintf(out, "    testing.tRunner testing.go:XXX\n")
			continue
		}
		parts := captureLeaderRE.FindStringSubmatch(l)
		if len(parts) == 4 {
			fmt.Fprintf(out, "%s%s\n", parts[1], parts[3])
			continue
		}
		out.WriteString(l)
		out.WriteString("\n")
	}
	return out.String()
}

type timeEtc struct {
	when     time.Time
	id       int64
	parentID int64
	args     string
}

func getTimeAndIDs(s string) ([]timeEtc, error) {
	recs := []timeEtc{}
	sc := bufio.NewScanner(bytes.NewBufferString(s))
	for sc.Scan() {
		l := sc.Text()
		parts := captureLeaderRE.FindStringSubmatch(l)
		if len(parts) != 4 {
			return nil, fmt.Errorf("failed to match line: %v", l)
		}
		tmp := parts[2][:26]
		when, err := time.Parse("060102 15:04:05.000000 MST", tmp)
		if err != nil {
			return nil, fmt.Errorf("malformed time: %v: %v", tmp, err)
		}
		tmp = parts[2][27:]
		idparts := idsRE.FindStringSubmatch(tmp)
		if len(idparts) != 3 {
			return nil, fmt.Errorf("failed to find ids in %v from line: %v", tmp, l)
		}
		id, err := strconv.ParseInt(idparts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse id %v from line: %v", idparts[1], l)
		}
		parent, err := strconv.ParseInt(idparts[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse parent id %v from line: %v", idparts[2], l)
		}
		recs = append(recs, timeEtc{
			id:       id,
			parentID: parent,
			when:     when,
			args:     parts[3],
		})
	}
	return recs, nil
}
