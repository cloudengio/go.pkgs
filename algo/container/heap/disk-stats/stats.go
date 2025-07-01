//go:build ignore

package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/btree"
)

type info struct {
	path string
	fs.FileInfo
}

func lessFunc(a, b *info) bool {
	return a.path < b.path
}

func main() {
	bt := btree.NewG(2, lessFunc)
	dirs := make(map[string]int64, 2286366)
	nfiles, nerrors := 0, 0
	sc := bufio.NewScanner(os.Stdin)
	for sc.Scan() {
		nfiles++
		l := sc.Text()
		fi, err := os.Stat(l)
		if err != nil {
			nerrors++
			continue
		}
		nfiles++
		dup, ok := bt.ReplaceOrInsert(&info{l, fi})
		if ok {
			panic(fmt.Sprintf("duplicate: %v: %v\n", l, dup.Name()))
		}
		d := filepath.Dir(l)
		dirs[d]++
		if nfiles == 100000 {
			break
		}
	}

	fmt.Printf("bt.Len() = %v (errs: %v)\n", bt.Len(), nerrors)

	maxDir, minDir := "", ""
	max, min := int64(0), int64(1<<63-1)
	buckets := [5]int64{}
	bucketFiles := [5]int64{}

	for k, v := range dirs {
		if v > max {
			max = v
			maxDir = k
		}
		if v < min {
			min = v
			minDir = k
		}
		switch {
		case v < 4:
			buckets[0]++
			bucketFiles[0] += v
		case v < 16:
			buckets[1]++
			bucketFiles[1] += v
		case v < 64:
			buckets[2]++
			bucketFiles[2] += v
		case v < 256:
			buckets[3]++
			bucketFiles[3] += v
		default:
			buckets[4]++
			bucketFiles[4] += v
		}
	}

	fmt.Printf("# dirs: %v, # files: %v\n", len(dirs), nfiles)
	fmt.Printf("# max: %v (%v), min: %v (%v)\n", max, maxDir, min, minDir)
	fmt.Printf("buckets % 6v\n", buckets)
	fmt.Printf("buckets % 6v\n", bucketFiles)

}
