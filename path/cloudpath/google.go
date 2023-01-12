// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath

import (
	"net/url"
	"strings"
)

// GoogleCloudStorageMatcher implements Matcher for Google Cloud Storage
// object names. It returns GoogleCloudStorage for its scheme result.
func GoogleCloudStorageMatcher(p string) *Match {
	m := &Match{
		Scheme:    GoogleCloudStorage,
		Separator: '/',
	}
	if len(p) >= 5 && p[0:5] == "gs://" {
		m.Path = p[5:]
		m.Volume, m.Key = bucketAndKey(m.Path)
		return m
	}
	u, err := url.Parse(p)
	if err != nil {
		return nil
	}
	m.Parameters = parametersFromQuery(u)
	switch u.Scheme {
	case "http", "https":
	default:
		return nil
	}
	m.Host = u.Host
	m.Path = u.Path
	switch u.Host {
	default:
		return nil
	case "storage.cloud.google.com":
		// https://storage.cloud.google.com/[BUCKET_NAME]/[OBJECT_NAME]
		m.Volume, m.Key = bucketAndKey(u.Path)
		return m
	case "storage.googleapis.com":
		/*
			https://storage.googleapis.com/storage/v1/PATH_TO_RESOURCE
			https://storage.googleapis.com/batch/storage/v1/PATH_TO_RESOURCE
			https://storage.googleapis.com/download/storage/v1/b/BUCKET_NAME/o/OBJECT_NAME?alt=media
			https://storage.googleapis.com/upload/storage/v1/b/BUCKET_NAME/o?name=OBJECT_NAME
		*/
		endpoint := "storage/v1/b"
		idx := strings.Index(u.Path, endpoint)
		if idx < 0 {
			return nil
		}
		m.Path = u.Path[idx+len(endpoint):]
		op := u.Path[:idx]
		switch op {
		default:
			return nil
		case "/":
			m.Volume, m.Key = bucketAndKey(m.Path)
		case "/download/":
			oidx := strings.Index(m.Path, "/o/")
			if oidx < 0 {
				return nil
			}
			m.Volume = m.Path[1:oidx]
			m.Key = m.Path[oidx+2:]
		case "/upload/":
			oidx := strings.Index(m.Path, "/o")
			if oidx < 0 {
				return nil
			}
			m.Volume = m.Path[1:oidx]
			m.Key = u.Query().Get("name")
		case "/batch/":
			return m
		}
		return m
	}
}
