package filewalk

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type local struct{}

func createInfo(i os.FileInfo) *Info {
	return &Info{
		Name:     i.Name(),
		Size:     i.Size(),
		ModTime:  i.ModTime(),
		IsPrefix: i.IsDir(),
		IsLink:   (i.Mode() & os.ModeSymlink) == os.ModeSymlink,
		Sys:      i,
	}
}

func (l *local) List(ctx context.Context, path string, n int, ch chan<- Contents) {
	f, err := os.Open(path)
	if err != nil {
		ch <- Contents{Path: path, Err: err}
		return
	}
	defer f.Close()
	for {
		select {
		case <-ctx.Done():
			ch <- Contents{Path: path, Err: ctx.Err()}
		default:
		}
		infos, err := f.Readdir(n)
		if len(infos) > 0 {
			files := make([]*Info, 0, len(infos))
			dirs := make([]*Info, 0, 10)
			for _, info := range infos {
				if info.IsDir() {
					dirs = append(dirs, createInfo(info))
					continue
				}
				files = append(files, createInfo(info))
			}
			ch <- Contents{
				Path:     path,
				Children: dirs,
				Files:    files,
				Err:      err,
			}
		}
		if err != nil {
			if err == io.EOF {
				return
			}
			ch <- Contents{Path: path, Err: err}
			return
		}
	}
}

func (l *local) Stat(ctx context.Context, path string) (*Info, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	return createInfo(info), nil
}

func (l *local) Join(components ...string) string {
	return filepath.Join(components...)
}

func LocalScanner() Scanner {
	return &local{}
}
