package filewalk_test

import (
	"context"
	"fmt"
	"testing"

	"cloudeng.io/file/filewalk"
)

type logger struct {
	dirs  []filewalk.Info
	files []filewalk.Info
}

func (l *logger) filesFunc(ctx context.Context, prefix string, ch <-chan filewalk.ListResults) error {
	for results := range ch {
		fmt.Printf("FILES: %v: # %v / %v\n", prefix, len(results.Files), len(results.Children))
		l.files = append(l.files, results.Files...)
	}
	return nil
}

func (l *logger) dirsFunc(ctx context.Context, prefix string, info filewalk.Info, err error) (bool, error) {
	if err != nil {
		fmt.Printf("ERR: %v .. %v\n", prefix, err)
		return true, err
	}
	fmt.Printf("DIR: %v .. %v\n", prefix, info.IsPrefix())
	l.dirs = append(l.dirs, info)
	return false, nil
}

func TestSimple(t *testing.T) {
	ctx := context.Background()
	sc := filewalk.LocalScanner()
	wk := filewalk.New(sc)
	lg := &logger{}
	err := wk.Walk(ctx, lg.dirsFunc, lg.filesFunc, "/")
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}

	for i, info := range lg.dirs {
		fmt.Printf("%v: dir %v\n", i, info.Name())
	}

	for i, info := range lg.files {
		fmt.Printf("%v: file %v\n", i, info.Name())
	}

}
