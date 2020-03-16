// Copyright 2020 cloudeng LLC. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package cloudpath

import (
	"net/url"
	"strings"
)

func stripStorageURLPrefix(p string) (bool, string) {
	for _, prefix := range []string{
		"/storage/v1/b",
		"/upload/storage/v1/b",
		"/batch/storage/v1/b",
	} {
		if s := strings.TrimPrefix(p, prefix); len(s) < len(p) {
			return true, s
		}
	}
	return false, p
}

// GoogleCloudStorageMatcher implements Matcher for Google Cloud Storage
// object names. It returns GoogleCloudStorage for its scheme result.
func GoogleCloudStorageMatcher(p string) *Match {
	u, err := url.Parse(p)
	if err != nil {
		return nil
	}
	m := &Match{
		Scheme:     GoogleCloudStorage,
		Separator:  '/',
		Parameters: parametersFromQuery(u),
	}
	switch u.Scheme {
	default:
		return nil
	case "gs":
		// gs://[BUCKET_NAME]/[OBJECT_NAME]
		m.Volume = u.Host
		m.Path = pathFromHostAndPath(u)
		return m
	case "http", "https":
	}
	m.Host = u.Host
	m.Path = u.Path
	switch u.Host {
	default:
		return nil
	case "storage.cloud.google.com":
		// https://storage.cloud.google.com/[BUCKET_NAME]/[OBJECT_NAME]
		m.Volume = firstPathComponent(u.Path)
		return m
	case "storage.googleapis.com", "www.googleapis.com":
		// https://storage.googleapis.com/storage/v1/[PATH_TO_RESOURCE]
		// https://storage.googleapis.com/upload/storage/v1/b/[BUCKET_NAME]/o
		// https://storage.googleapis.com/batch/storage/v1/[PATH_TO_RESOURCE]
	}
	if prefix, stripped := stripStorageURLPrefix(u.Path); prefix {
		m.Volume = firstPathComponent(stripped)
		m.Path = stripped
	}
	return m
}
