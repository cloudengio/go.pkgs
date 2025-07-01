//go:build ignore

package main

import (
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"os"
	"time"

	"cloudeng.io/cmdutil/profiling"
	"cloudeng.io/file"
)

type info struct {
	Path string
	Info file.Info
}

func main() {
	save, err := profiling.StartFromSpecs(
		profiling.ProfileSpec{Name: "cpu", Filename: "cpu.out"},
		profiling.ProfileSpec{Name: "mem", Filename: "mem.out"},
	)
	if err != nil {
		panic(err)
	}
	defer save()

	start := time.Now()
	infile := os.ExpandEnv("$HOME/filewalk.gob.gz")
	file, err := os.Open(infile)
	if err != nil {
		panic(err)
	}
	rd, err := gzip.NewReader(file)
	if err != nil {
		panic(err)
	}
	dec := gob.NewDecoder(rd)
	nr := 0
	for {
		var fi info
		if err := dec.Decode(&fi); err != nil {
			fmt.Printf("err: %v\n", err)
			break
		}
		//fmt.Printf("%v\n", fi.Path)
		nr++
	}
	fmt.Printf("%v: read: %v\n", time.Since(start), nr)
}
