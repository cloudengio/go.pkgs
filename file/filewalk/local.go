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
	//scanSize int
	// numList  int64
	// numStat  int64
}

/*
func (l *local) Stats() FilesystemStats {
	return FilesystemStats{
		NumList: atomic.LoadInt64(&l.numList),
		NumStat: atomic.LoadInt64(&l.numStat),
	}
}*/

type scanner struct {
	err     error
	file    *os.File
	entries []fs.DirEntry
}

func (s *scanner) ReadDir() []fs.DirEntry {
	return s.entries
}

func (s *scanner) Scan(_ context.Context, n int) bool {
	dirEntries, err := s.file.ReadDir(n)
	if err != nil {
		s.file.Close()
		if err = io.EOF; err != nil {
			return false
		}
		s.err = err
		return false
	}
	s.entries = dirEntries
	return true
}

func (s *scanner) Err() error {
	return s.err
}

func NewScanner(path string) Scanner {
	f, err := os.Open(path)
	if err != nil {
		return &scanner{err: err}
	}
	return &scanner{file: f}
}

/*
func (l *local) List(ctx context.Context, path string, dirsOnly bool, ch chan<- Contents) {
	f, err := os.Open(path)
	if err != nil {
		ch <- Contents{Path: path, Err: err}
		return
	}
	defer f.Close()
	//	atomic.AddInt64(&l.numList, 1)
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
			for _, de := range dirEntries {
				if de.IsDir() {
					info, err := de.Info()
					//					atomic.AddInt64(&l.numStat, 1)
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
				//				atomic.AddInt64(&l.numStat, 1)
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
					size, _ = symlinkSize(path, info)
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
}*/

func (l *local) DirScanner(path string) DirScanner {
	return NewDirScanner(path)
}

/*
func createInfo(path string, fi fs.FileInfo, symlinkSize int64) file.Info {
	size := fi.Size()
	if size == 0 && symlinkSize > 0 {
		size = symlinkSize
	}
	return file.NewInfo(
		fi.Name(),
		size,
		fi.Mode(),
		fi.ModTime(),
		fi)
}*/

func (l *local) Stat(ctx context.Context, path string) (file.Info, error) {
	info, err := os.Stat(path)
	if err != nil {
		return file.Info{}, err
	}
	//	atomic.AddInt64(&l.numStat, 1)
	return file.NewInfoFromFileInfo(info), nil
}

func (l *local) LStat(ctx context.Context, path string) (file.Info, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return file.Info{}, err
	}
	//	atomic.AddInt64(&l.numStat, 1)
	return file.NewInfoFromFileInfo(info), nil
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

func LocalFilesystem() FS {
	return &local{}
}
