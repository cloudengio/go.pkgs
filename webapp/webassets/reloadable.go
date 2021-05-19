package webassets

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"sync"
	"time"
)

type state struct {
	modtime time.Time
	file    fs.File
}

type reloadable struct {
	sync.Mutex
	embedded fs.FS
	dynroot  string
	files    map[string]fs.File
	stat     map[string]fs.FileInfo
}

func (r *reloadable) fullpath(p string) string {
	return path.Join(r.dynroot, p)
}

func differs(a, b fs.FileInfo) bool {
	return a.ModTime() != b.ModTime() && a.Size() != b.Size()
}

func (r *reloadable) statEmbedded(name string) (fs.FileInfo, error) {
	r.Lock()
	defer r.Unlock()
	if fi, ok := r.stat[name]; ok {
		return fi, nil
	}
	f, err := r.embedded.Open(name)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	r.stat[name] = fi
	return fi, nil
}

func (r *reloadable) reload(name string) (bool, error) {
	if r.embedded == nil {
		return true, nil
	}
	fp := r.fullpath(name)
	ondisk, err := os.Stat(fp)
	if err == nil {
		inram, err := r.statEmbedded(name)
		if err != nil {
			return true, fmt.Errorf("failed to stat embedded file: %v: %v", name, err)
		}
		return differs(ondisk, inram), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to stat on disk file: %v: %v", fp, err)
}

func (r *reloadable) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}
	shouldReload, err := r.reload(name)
	if err != nil {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  err,
		}
	}
	if !shouldReload {
		return r.embedded.Open(name)
	}
	fp := r.fullpath(name)
	return os.Open(fp)
}

func Reloadable(embedded fs.FS, dynamic string) fs.FS {
	return &reloadable{
		embedded: embedded,
		dynroot:  dynamic,
		files:    make(map[string]fs.File),
		stat:     make(map[string]fs.FileInfo),
	}
}
