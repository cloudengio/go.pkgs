// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"context"
	"fmt"
	"os"

	"cloudeng.io/aws/awsconfig"
	"cloudeng.io/aws/s3fs"
)

func main() {
	ctx := context.Background()
	name := os.Args[1]

	cfg, err := awsconfig.Load(ctx)
	if err != nil {
		panic(err)
	}
	fs := s3fs.New(cfg)

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
}
