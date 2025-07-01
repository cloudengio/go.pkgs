//go:build ignore

package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloudeng.io/file"
	"cloudeng.io/file/filewalk"
)

type info struct {
	Path string
	Info file.Info
}

type saver struct {
	mu sync.Mutex
	//cmp   *gzip.Writer
	enc   *gob.Encoder
	total int
}

func (s *saver) save(path string, fi file.Info) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.enc.Encode(&info{Path: path, Info: fi}); err != nil {
		return err
	}
	s.total++
	if s.total%10000 == 0 {
		fmt.Printf("%v: written: % 10v\n", time.Now().Format(time.TimeOnly), s.total)
		//		s.cmp.Flush()
	}
	return nil
}

func (s *saver) filesFunc(ctx context.Context, prefix string, parent file.Info, ch <-chan filewalk.Contents) (file.InfoList, error) {
	if err := s.save(prefix, parent); err != nil {
		return nil, err
	}
	children := make(file.InfoList, 0, 10)
	for results := range ch {
		for _, fi := range results.Files {
			if err := s.save(filepath.Join(prefix, fi.Name()), fi); err != nil {
				return nil, err
			}
		}
		children = append(children, results.Children...)
	}
	return children, nil
}

func (s *saver) dirsFunc(ctx context.Context, prefix string, info file.Info, err error) (bool, file.InfoList, error) {
	if err != nil {
		return true, nil, nil
	}
	return strings.HasPrefix(prefix, "/Volumes/") || strings.HasPrefix(prefix, "/System/"), nil, nil
}

func main() {
	ctx := context.Background()
	sc := filewalk.LocalFilesystem(1000)
	wk := filewalk.New(sc)

	outfile := os.ExpandEnv("$HOME/filewalk.gob.gz")
	file, err := os.OpenFile(outfile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	//	cmp := gzip.NewWriter(file)
	saver := &saver{ /*cmp: cmp,*/ enc: gob.NewEncoder(file)}

	if err := wk.Walk(ctx, saver.dirsFunc, saver.filesFunc, os.Args[1]); err != nil {
		panic(err)
	}
	//	if err := cmp.Flush(); err != nil {
	//		panic(err)
	//	}
	fmt.Printf("%v: wrote: %v\n", time.Now().Format(time.TimeOnly), saver.total)
}
