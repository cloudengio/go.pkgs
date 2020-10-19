// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package profiling provides stylised support for enabling profiling of
// command line tools.
package profiling

import (
	"fmt"
	"os"
	"runtime/pprof"
	"strings"

	"cloudeng.io/errors"
)

type ProfileSpec struct {
	Name     string
	Filename string
}

type ProfileFlag struct {
	Profiles []ProfileSpec
}

func (pf *ProfileFlag) Set(v string) error {
	parts := strings.Split(v, ":")
	if len(parts) != 2 {
		return fmt.Errorf("%v not in <profile>:<filename> format", v)
	}
	pf.Profiles = append(pf.Profiles, ProfileSpec{Name: parts[0], Filename: parts[1]})
	return nil
}

func (pf *ProfileFlag) String() string {
	out := &strings.Builder{}
	for i, p := range pf.Profiles {
		fmt.Fprintf(out, "%s:%s", p.Name, p.Filename)
		if i < len(pf.Profiles)-1 {
			out.WriteByte(',')
		}
	}
	return out.String()
}

func (pf *ProfileFlag) Get() interface{} {
	return pf.Profiles
}

func EnableCPUProfiling(filename string) (func() error, error) {
	if len(filename) == 0 {
		return func() error { return nil }, nil
	}
	output, err := os.Create(filename)
	if err != nil {
		nerr := fmt.Errorf("could not create CPU profile: %v: %v", filename, err)
		return func() error { return nerr }, nerr
	}
	if err := pprof.StartCPUProfile(output); err != nil {
		nerr := fmt.Errorf("could not start CPU profile: %v", err)
		return func() error { return nerr }, nerr
	}
	return func() error {
		pprof.StopCPUProfile()
		return output.Close()
	}, nil
}

func StartProfile(name, filename string) (func() error, error) {
	if len(name) == 0 || len(filename) == 0 {
		err := fmt.Errorf("missing profile or filename: %q:%q", name, filename)
		return func() error { return err }, err
	}
	if name == "cpu" {
		save, err := EnableCPUProfiling(filename)
		return save, err
	}
	output, err := os.OpenFile(filename, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0760)
	if err != nil {
		return func() error { return err }, err
	}
	p := pprof.Lookup(name)
	if p == nil {
		p = pprof.NewProfile(name)
	}
	return func() error {
		errs := errors.M{}
		errs.Append(p.WriteTo(output, 0))
		errs.Append(output.Close())
		return errs.Err()
	}, nil
}
