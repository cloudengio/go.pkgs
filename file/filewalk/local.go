package filewalk

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type local struct{}

type osinfo struct {
	os.FileInfo
}

func (i osinfo) IsPrefix() bool {
	return i.IsDir()
}

func (i osinfo) IsLink() bool {
	return (i.Mode() & os.ModeSymlink) == os.ModeSymlink
}

func (i osinfo) Sys() interface{} {
	return i.FileInfo
}

func (l *local) List(ctx context.Context, path string, n int, ch chan<- ListResults) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer close(ch)
	for {
		infos, err := f.Readdir(n)
		files := make([]Info, 0, len(infos))
		dirs := make([]Info, 0, 10)
		for _, info := range infos {
			if info.IsDir() {
				dirs = append(dirs, osinfo{info})
				continue
			}
			files = append(files, osinfo{info})
		}
		fmt.Printf("RES: %v: %v %v: %v\n", path, len(dirs), len(files), err)
		ch <- ListResults{
			Parent:   path,
			Children: dirs,
			Files:    files,
			Err:      err,
		}
		if err == io.EOF {
			return nil
		}
	}
}

func (l *local) Stat(ctx context.Context, path string) (Info, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	return &osinfo{info}, nil
}

func (l *local) Join(components ...string) string {
	return filepath.Join(components...)
}

func LocalScanner() Scanner {
	return &local{}
}
