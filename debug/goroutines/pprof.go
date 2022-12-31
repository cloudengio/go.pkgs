// Copyright 2022 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package goroutines

import (
	"bytes"
	"context"
	"fmt"
	"runtime/pprof"
	"strings"

	"github.com/google/pprof/profile"
)

// RunUnderPProf uses pprof's label support to attach the specified key/value
// label to all goroutines spawed by the supplied runner. RunUnderPprof returns
// when runner returns.
func RunUnderPProf(ctx context.Context, key, value string, runner func(context.Context)) {
	pprof.Do(ctx, pprof.Labels(key, value), runner)
}

func findAndParse() (*profile.Profile, error) {
	var pb bytes.Buffer
	profiler := pprof.Lookup("goroutine")
	if profiler == nil {
		return nil, fmt.Errorf("no goroutine profile")
	}
	err := profiler.WriteTo(&pb, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile date: %w", err)
	}
	p, err := profile.ParseData(pb.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to parse profile data: %w", err)
	}
	return p, nil
}

// PProfLabelExists returns true if a goroutine with the pprof key/value label
// exists.
func PProfLabelExists(key, value string) (bool, error) {
	p, err := findAndParse()
	if err != nil {
		return false, err
	}
	for _, sample := range p.Sample {
		if hasLabel(sample, key, value) {
			return true, nil
		}
	}
	return false, nil
}

// FormatPProfGoroutines returns a nicely formatted dump of the goroutines
// with the pprof key/value label.
func FormatPProfGoroutines(key, value string) (string, error) {
	p, err := findAndParse()
	if err != nil {
		return "", err
	}
	out := &strings.Builder{}
	for _, sample := range p.Sample {
		if !hasLabel(sample, key, value) {
			continue
		}
		fmt.Fprintf(out, "count %d @", sample.Value[0])
		for _, loc := range sample.Location {
			for i, ln := range loc.Line {
				if i == 0 {
					fmt.Fprintf(out, "#   %#8x", loc.Address)
					if loc.IsFolded {
						fmt.Fprint(out, " [F]")
					}
				} else {
					fmt.Fprint(out, "#           ")
				}
				if fn := ln.Function; fn != nil {
					fmt.Fprintf(out, " %-50s %s:%d", fn.Name, fn.Filename, ln.Line)
				} else {
					fmt.Fprintf(out, " ???")
				}
				fmt.Fprintf(out, "\n")
			}
		}
		fmt.Fprintf(out, "\n")
	}
	return out.String(), nil
}

func hasLabel(sample *profile.Sample, key, value string) bool {
	values, hasLabel := sample.Label[key]
	if !hasLabel {
		return false
	}
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
