// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/aws/s3fs"
	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
)

func main() {
	ctx := context.Background()
	name := os.Args[1]

	cfg, err := awsconfig.Load(ctx)
	if err != nil {
		panic(err)
	}
	fs := s3fs.NewS3FS(cfg)

	/*
		f, err := fs.OpenCtx(ctx, name)
		if err != nil {
			panic(err)
		}
		info, err := f.Stat()
		if err != nil {
			panic(err)
		}

		cksum := sha1.New()

		n, err := io.Copy(cksum, f)
		if err != nil {
			panic(err)
		}

		if n != info.Size() {
			panic(fmt.Sprintf("short read %v %v != %v", name, n, info.Size()))
		}

		fmt.Printf("%x %v\n", cksum.Sum(nil), name)

		finfo, err := fs.Stat(ctx, name)
		if err != nil {
			panic(err)
		}
		xattr, err := fs.XAttr(ctx, name, finfo)
		if err != nil {
			panic(err)
		}

		fmt.Printf("%#v\n", xattr)
	*/

	sc := fs.LevelScanner(name)
	for sc.Scan(ctx, 1) {
		for _, c := range sc.Contents() {
			fmt.Printf("%v %v\n", c.Name, c.Type)
			_ = c
		}
	}
	if err := sc.Err(); err != nil {
		panic(err)
	}

	w := &s3walker{fs: fs}
	walker := filewalk.New(fs, w)
	if err := walker.Walk(ctx, name); err != nil {
		log.Fatal(err)
	}

	err = fs.DeleteAll(ctx, fs.Join(name, ".git/logs/refs/remotes/origin/dependabot/go_modules"))
	if err != nil {
		log.Fatal(err)
	}
}

type s3walker struct {
	fs filewalk.FS
}

func (w *s3walker) Prefix(_ context.Context, state *struct{}, prefix string, _ file.Info, err error) (bool, file.InfoList, error) {
	fmt.Printf("%v/\n", prefix)
	return false, nil, nil
}

func (w *s3walker) Contents(ctx context.Context, state *struct{}, prefix string, contents []filewalk.Entry) (file.InfoList, error) {
	children := make(file.InfoList, 0, len(contents))
	for _, c := range contents {
		key := w.fs.Join(prefix, c.Name)
		if !c.IsDir() {
			fmt.Printf("%v\n", key)
			continue
		}
		info, err := w.fs.Stat(ctx, key)
		if err != nil {
			return nil, err
		}
		children = append(children, info)
	}
	return children, nil
}

func (w *s3walker) Done(_ context.Context, state *struct{}, prefix string, err error) error {
	return err
}
