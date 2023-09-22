// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalk

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync/atomic"

	"cloudeng.io/file"
)

type local struct {
	scanSize    int
	numList     int64
	numStat     int64
	numPrefixes int64
	numFiles    int64
}

func createInfo(path string, fi fs.FileInfo, symlinkSize int64) file.Info {
	userID, groupID := getUserAndGroupID(path, fi)
	size := fi.Size()
	if size == 0 && symlinkSize > 0 {
		size = symlinkSize
	}
	return *file.NewInfo(
		fi.Name(),
		size,
		fi.Mode(),
		fi.ModTime(),
		file.InfoOption{
			User:    userID,
			Group:   groupID,
			IsDir:   fi.IsDir(),
			IsLink:  fi.Mode()&os.ModeSymlink == os.ModeSymlink,
			SysInfo: fi,
		},
	)
}

func (l *local) Stats() FilesystemStats {
	return FilesystemStats{
		NumList:     atomic.LoadInt64(&l.numList),
		NumStat:     atomic.LoadInt64(&l.numStat),
		NumFiles:    atomic.LoadInt64(&l.numFiles),
		NumPrefixes: atomic.LoadInt64(&l.numPrefixes),
	}
}

func (l *local) List(ctx context.Context, path string, dirsOnly bool, ch chan<- Contents) {
	f, err := os.Open(path)
	if err != nil {
		ch <- Contents{Path: path, Err: err}
		return
	}
	defer f.Close()
	atomic.AddInt64(&l.numList, 1)
	for {
		select {
		case <-ctx.Done():
			ch <- Contents{Path: path, Err: ctx.Err()}
		default:
		}
		dirEntries, err := f.ReadDir(l.scanSize)
		if len(dirEntries) > 0 {
			var files []file.Info
			if !dirsOnly {
				files = make([]file.Info, 0, len(dirEntries))
			}
			dirs := make([]file.Info, 0, 10)
			fmt.Printf("%s: # dir entries %v\n", path, len(dirEntries))
			for _, de := range dirEntries {
				if de.IsDir() {
					info, err := de.Info()
					if err != nil {
						break
					}
					dirs = append(dirs, createInfo(path, info, -1))
					continue
				}
				if dirsOnly {
					continue
				}
				info, err := de.Info()
				if err != nil {
					break
				}
				if (info.Mode()&os.ModeSymlink) == os.ModeSymlink && info.Size() == 0 {
					s, err := os.Readlink(filepath.Join(path, info.Name()))
					if err == nil {
						ni := createInfo(path, info, int64(len(s)))
						files = append(files, ni)
						continue
					}
				}
				size := int64(-1)
				if (info.Mode()&os.ModeSymlink) == os.ModeSymlink && info.Size() == 0 {
					size = symlinkSize(path, info)
				}
				files = append(files, createInfo(path, info, size))
			}
			atomic.AddInt64(&l.numFiles, int64(len(files)))
			atomic.AddInt64(&l.numPrefixes, int64(len(dirs)))
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

func (l *local) Stat(_ context.Context, path string) (file.Info, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return file.Info{}, err
	}
	atomic.AddInt64(&l.numStat, 1)
	return createInfo(path, info, -1), nil
}

func (l *local) Join(components ...string) string {
	return filepath.Join(components...)
}

func (l *local) IsPermissionError(err error) bool {
	return os.IsPermission(err)
}

func (l *local) IsNotExist(err error) bool {
	return os.IsNotExist(err)
}

func LocalFilesystem(scanSize int) Filesystem {
	return &local{scanSize: scanSize}
}
