package filewalk

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"time"
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

type jsonInfo struct {
	Tag      string
	Name     string
	Size     int64
	ModTime  time.Time
	IsPrefix bool `json:,omitempty`
	IsLink   bool `json:,omitempty`
}

func (i osinfo) MarshalJSON() ([]byte, error) {
	ji := jsonInfo{
		Name:     i.Name(),
		Size:     i.Size(),
		ModTime:  i.ModTime(),
		IsPrefix: i.IsPrefix(),
		IsLink:   i.IsLink(),
	}
	return json.Marshal(&ji)
}

func (i *osinfo) UnmarshalJSON(buf []byte) error {
	ji := &jsonInfo{}
	if err := json.Unmarshal(buf, ji); err != nil {
		return err
	}
	return nil
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
			files := make([]Info, 0, len(infos))
			dirs := make([]Info, 0, 10)
			for _, info := range infos {
				if info.IsDir() {
					dirs = append(dirs, &osinfo{info})
					continue
				}
				files = append(files, &osinfo{info})
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
