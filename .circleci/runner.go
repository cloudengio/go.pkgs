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
)

func main() {
	ctx := context.Background()
	flag.BoolVar(&prepFlag, "prep", false, "prepare file system for tests")
	flag.BoolVar(&modulesFlag, "modules", false, "print modules in this repo")
	flag.BoolVar(&testFlag, "test", false, "run tests")
	flag.Parse()

	if modulesFlag {
		mods, err := subdirs()
		if err != nil {
			panic(err)
		}
		fmt.Println(strings.Join(mods, " "))
	}

	if prepFlag {
		if err := prep(); err != nil {
			panic(err)
		}
	}
	if testFlag {
		if err := runTests(ctx); err != nil {
			panic(err)
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

func runTests(ctx context.Context) error {
	dirs, err := subdirs()
	if err != nil {
		return err
	}
	for _, dir := range dirs {
		if err := runTest(ctx, dir); err != nil {
			return err
		}
	}
	return nil
}

func runTest(ctx context.Context, dir string) error {
	fmt.Printf("%v...\n", dir)
	cmd := exec.CommandContext(ctx, "go", "test", "-failfast", "--covermode=atomic", "-race", "./...")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
