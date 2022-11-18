// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package profiling provides support for enabling profiling of
// command line tools via flags.
package profiling

import (
	"fmt"
	"os"
	"runtime/pprof"
	"strings"

	"cloudeng.io/errors"
)

// ProfileSpec represents a named profile and the name of the file to
// write its contents to. CPU profiling can be requested using the
// name 'cpu' rather than the CPUProfiling API calls in runtime/pprof
// that predate the named profiles.
type ProfileSpec struct {
	Name     string
	Filename string
}

// ProfileFlag can be used to represent flags to request arbritrary profiles.
type ProfileFlag struct {
	Profiles []ProfileSpec
}

// Set implements flag.Value.
func (pf *ProfileFlag) Set(v string) error {
	parts := strings.Split(v, ":")
	if len(parts) != 2 {
		return fmt.Errorf("%v not in <profile>:<filename> format", v)
	}
	pf.Profiles = append(pf.Profiles, ProfileSpec{Name: parts[0], Filename: parts[1]})
	return nil
}

// String implements flag.Value.
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

// Get implements flag.Getter.
func (pf *ProfileFlag) Get() interface{} {
	return pf.Profiles
}

func enableCPUProfiling(filename string) (func() error, error) {
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

// Start enables the named profile and returns a function that
// can be used to save its contents to the specified file.
// Typical usage is as follows:
//
//	save, err := profiling.Start("cpu", "cpu.out")
//	if err != nil {
//	   panic(err)
//	}
//	defer save()
//
// For a heap profile simply use Start("heap", "heap.out"). Note that the
// returned save function cannot be used more than once and that Start must
// be called multiple times to create multiple heap output files for example.
// All of the predefined named profiles from runtime/pprof are supported. If
// a new, custom profile is requested, then the caller must obtain a reference
// to it via pprof.Lookup and the create profiling records appropriately.
func Start(name, filename string) (func() error, error) {
	if len(name) == 0 || len(filename) == 0 {
		err := fmt.Errorf("missing profile or filename: %q:%q", name, filename)
		return func() error { return err }, err
	}
	if name == "cpu" {
		save, err := enableCPUProfiling(filename)
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

// StartFromSpecs starts all of the specified profiles.
func StartFromSpecs(specs ...ProfileSpec) (func(), error) {
	deferedSaves := []func() error{}
	for _, profile := range specs {
		save, err := Start(profile.Name, profile.Filename)
		if err != nil {
			return nil, err
		}
		fmt.Printf("profiling: %v %v\n", profile.Name, profile.Filename)
		deferedSaves = append(deferedSaves, save)
	}
	return func() {
		for _, save := range deferedSaves {
			if err := save(); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
		}
	}, nil
}
