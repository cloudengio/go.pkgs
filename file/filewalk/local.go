// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalk

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"cloudeng.io/file"
)

type local struct {
	scanSize int
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

func (l *local) List(ctx context.Context, path string, ch chan<- Contents) {
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
		infos, err := f.Readdir(l.scanSize)
		if len(infos) > 0 {
			files := make([]file.Info, 0, len(infos))
			dirs := make([]file.Info, 0, 10)
			for _, info := range infos {
				if info.IsDir() {
					dirs = append(dirs, createInfo(path, info, -1))
					continue
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
