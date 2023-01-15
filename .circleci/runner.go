// Copyright 2021 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	prepFlag    bool
	testFlag    bool
	modulesFlag bool
	lintFlag    bool
)

func done(msg string, err error) {
	fmt.Printf("Failed: %s: %s\n", msg, err)
	os.Exit(1)
}

func main() {
	ctx := context.Background()
	flag.BoolVar(&prepFlag, "prep", false, "prepare file system for tests")
	flag.BoolVar(&modulesFlag, "modules", false, "print modules in this repo")
	flag.BoolVar(&testFlag, "test", false, "run tests")
	flag.BoolVar(&lintFlag, "lint", false, "run lint")

	flag.Parse()

	if !(modulesFlag || prepFlag || testFlag) {
		fmt.Fprintf(os.Stderr, "at least one flag is required\n")
		flag.Usage()
		os.Exit(1)
	}

	if modulesFlag {
		mods, err := subdirs()
		if err != nil {
			done("finding modules", err)
		}
		fmt.Println(strings.Join(mods, " "))
		return
	}
	mods := flag.Args()
	if len(mods) == 0 {
		var err error
		mods, err = subdirs()
		if err != nil {
			done("finding modules", err)
		}
	}
	if prepFlag {
		if err := prep(); err != nil {
			done("prep", err)
		}
	}

	if testFlag {
		if err := runTests(ctx, mods); err != nil {
			done("tests", err)
		}
	}

	if lintFlag {
		if err := runLints(ctx, mods); err != nil {
			done("lint", err)
		}
	}

}

func prep() error {
	for _, dir := range []string{
		filepath.Join("webapp", "cmd", "webapp", "webapp-sample", "build"),
		filepath.Join("webapp", "cmd", "webapp", "webapp-sample", "build", "static", "css"),
		filepath.Join("webapp", "cmd", "webapp", "webapp-sample", "build", "static", "js"),
		filepath.Join("webapp", "cmd", "webapp", "webapp-sample", "build", "static", "media"),
	} {
		if err := os.MkdirAll(dir, 0777); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, "dummy-for-embed"), nil, 0666); err != nil {
			return err
		}
	}
	return nil
}

func subdirs() ([]string, error) {
	var dirs []string
	err := filepath.Walk(".", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if _, err := os.Open(filepath.Join(path, "go.mod")); err == nil {
				dirs = append(dirs, path)
				return filepath.SkipDir
			}
		}
		return nil
	})
	return dirs, err
}

func runTests(ctx context.Context, dirs []string) error {
	failed := false
	for _, dir := range dirs {
		if err := runTest(ctx, dir); err != nil {
			fmt.Fprintf(os.Stderr, "%v: failed: %v\n", dir, err)
			failed = true
		}
	}
	if failed {
		return fmt.Errorf("tests failed")
	}
	return nil
}

func runTest(ctx context.Context, dir string) error {
	fmt.Printf("%v...\n", dir)
	cmd := exec.CommandContext(ctx, "go", "test", "-failfast", "--covermode=atomic", "--vet=off", "-race", "./...")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err == nil {
		fmt.Printf("%v... ok\n", dir)
	} else {
		fmt.Printf("%v... failed\n", dir)
	}
	return err
}

func runLints(ctx context.Context, dirs []string) error {
	failed := false
	for _, dir := range dirs {
		if err := runLint(ctx, dir); err != nil {
			fmt.Fprintf(os.Stderr, "%v: failed: %v\n", dir, err)
			failed = true
		}
	}
	if failed {
		return fmt.Errorf("lint failed")
	}
	return nil
}

func runLint(ctx context.Context, dir string) error {
	fmt.Printf("%v...\n", dir)
	cmd := exec.CommandContext(ctx, "go", "test", "-failfast", "--covermode=atomic", "--vet=off", "-race", "./...")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err == nil {
		fmt.Printf("%v... ok\n", dir)
	} else {
		fmt.Printf("%v... failed\n", dir)
	}
	return err
}
