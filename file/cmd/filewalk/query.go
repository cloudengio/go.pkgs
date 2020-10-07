package main

import (
	"container/heap"
	"context"
	"fmt"
	"strings"

	"cloudeng.io/errors"
	"cloudeng.io/file/filewalk/walkdb"
	"cloudeng.io/sync/errgroup"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type lsFlags struct {
	CommonFlags
	TopN int `subcmd:"top,20,show the top prefixes by file count and disk usage"`
}

type sizeHeap struct {
	d     []walkdb.Metric
	total int64
}

func (h sizeHeap) Len() int           { return len(h.d) }
func (h sizeHeap) Less(i, j int) bool { return h.d[i].Size >= h.d[j].Size }
func (h sizeHeap) Swap(i, j int)      { h.d[i], h.d[j] = h.d[j], h.d[i] }

func (h *sizeHeap) Push(x interface{}) {
	m := x.(walkdb.Metric)
	h.d = append(h.d, m)
	h.total += m.Size
}

func (h *sizeHeap) Pop() interface{} {
	old := h.d
	n := len(old)
	x := old[n-1]
	h.d = old[0 : n-1]
	return x
}

func (h *sizeHeap) topn(n int) []walkdb.Metric {
	if n >= len(h.d) {
		n = len(h.d) - 1
	}
	top := make([]walkdb.Metric, n)
	for i := 0; i < n; i++ {
		top[i] = heap.Pop(h).(walkdb.Metric)
	}
	return top
}

func lsTree(ctx context.Context, db *walkdb.Database, root string) (files, children, disk *sizeHeap, err error) {
	intPrinter := message.NewPrinter(language.English)
	files, children, disk = &sizeHeap{}, &sizeHeap{}, &sizeHeap{}
	sc := db.NewScanner(root, true)
	for sc.Scan() {
		prefix, pi := sc.Item()
		if err := pi.Err; len(err) > 0 {
			fmt.Printf("%s: %s\n", prefix, pi.Err)
			continue
		}
		heap.Push(files, walkdb.Metric{Prefix: prefix, Size: int64(len(pi.Files))})
		heap.Push(children, walkdb.Metric{Prefix: prefix, Size: int64(len(pi.Children))})
		heap.Push(disk, walkdb.Metric{Prefix: prefix, Size: pi.DiskUsage})
		intPrinter.Printf("% 15v (% 8v) - % 6v : %s\n", pi.DiskUsage, len(pi.Files), len(pi.Children), prefix)
	}
	err = sc.Err()
	return
}

func ls(ctx context.Context, values interface{}, args []string) error {
	flagValues := values.(*lsFlags)
	ctx = flagValues.withVerbosity(ctx)
	db, err := walkdb.Open(flagValues.DatabaseDir, walkdb.ReadOnly())
	if err != nil {
		return err
	}
	roots := args
	if len(roots) == 0 {
		roots = []string{""}
	}

	type results struct {
		root                  string
		files, children, disk *sizeHeap
		err                   error
	}

	listers := &errgroup.T{}
	listers = errgroup.WithConcurrency(listers, len(roots))
	resultsCh := make(chan results)
	for _, root := range roots {
		root := root
		listers.Go(func() error {
			files, children, disk, err := lsTree(ctx, db, root)
			resultsCh <- results{
				root:     root,
				files:    files,
				children: children,
				disk:     disk,
			}
			return err
		})
	}

	errs := errors.M{}
	go func() {
		errs.Append(listers.Wait())
		close(resultsCh)
	}()

	for result := range resultsCh {
		files, children, disk := result.files, result.children, result.disk
		heading := fmt.Sprintf("\n\nResults for %v", result.root)
		fmt.Println(heading)
		fmt.Println(strings.Repeat("=", len(heading)))
		fmt.Printf("\nTop %v prefixes by disk usage\n", flagValues.TopN)
		printMetric(disk.topn(flagValues.TopN))
		fmt.Printf("\nTop %v prefixes by file count\n", flagValues.TopN)
		printMetric(files.topn(flagValues.TopN))
		fmt.Printf("\nTop %v prefixes by child count\n", flagValues.TopN)
		printMetric(children.topn(flagValues.TopN))
	}
	return errs.Err()
}

type queryFlags struct {
	CommonFlags
}

func query(ctx context.Context, values interface{}, args []string) error {
	flagValues := values.(*queryFlags)
	_ = flagValues
	return nil
}
