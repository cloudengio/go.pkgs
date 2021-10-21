// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalk

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

type local struct {
	scanSize int
}

func createInfo(path string, i os.FileInfo) Info {
	info := Info{
		Name:    i.Name(),
		Size:    i.Size(),
		ModTime: i.ModTime(),
		sys:     i,
	}
	info.UserID, info.GroupID = getUserAndGroupID(path, i)
	m := i.Mode()
	info.Mode = FileMode(m&os.ModePerm | m&os.ModeSymlink | m&os.ModeDir)
	return info
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
			files := make([]Info, 0, len(infos))
			dirs := make([]Info, 0, 10)
			for _, info := range infos {
				if info.IsDir() {
					dirs = append(dirs, createInfo(path, info))
					continue
				}
				if (info.Mode()&os.ModeSymlink) == os.ModeSymlink && info.Size() == 0 {
					s, err := os.Readlink(filepath.Join(path, info.Name()))
					if err == nil {
						ni := createInfo(path, info)
						ni.Size = int64(len(s))
						files = append(files, ni)
						continue
					}
				}
				ni := createInfo(path, info)
				if (info.Mode()&os.ModeSymlink) == os.ModeSymlink && info.Size() == 0 {
					ni.Size = symlinkSize(path, info)
				}
				files = append(files, ni)
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
		return Info{}, err
	}
	return createInfo(path, info), nil
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
