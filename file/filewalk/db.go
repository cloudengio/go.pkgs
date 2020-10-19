// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package filewalk

import (
	"bytes"
	"context"
	"encoding/gob"
	"sync"
	"time"

	"cloudeng.io/errors"
)

// PrefixInfo represents information on a given prefix.
type PrefixInfo struct {
	ModTime   time.Time
	Size      int64
	UserID    string
	Mode      FileMode
	Children  []Info
	Files     []Info
	DiskUsage int64 // DiskUsage is the total amount of storage required for the files under this prefix taking the filesystem's layout/block size into account.
	Err       string
}

var bufPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func gobEncodeInfo(enc *gob.Encoder, info []Info) error {
	errs := errors.M{}
	errs.Append(enc.Encode(len(info)))
	for _, i := range info {
		errs.Append(enc.Encode(i.Name))
		errs.Append(enc.Encode(i.UserID))
		errs.Append(enc.Encode(i.Size))
		errs.Append(enc.Encode(i.ModTime))
		errs.Append(enc.Encode(i.Mode))
	}
	return errs.Err()
}

// GobEncode implements gob.Encoder.
func (pi PrefixInfo) GobEncode() ([]byte, error) {
	b := bufPool.Get().(*bytes.Buffer)
	b.Reset()
	errs := errors.M{}
	enc := gob.NewEncoder(b)
	errs.Append(enc.Encode(pi.ModTime))
	errs.Append(enc.Encode(pi.Size))
	errs.Append(enc.Encode(pi.UserID))
	errs.Append(enc.Encode(pi.Mode))
	errs.Append(enc.Encode(pi.DiskUsage))
	errs.Append(enc.Encode(pi.Err))
	errs.Append(gobEncodeInfo(enc, pi.Children))
	errs.Append(gobEncodeInfo(enc, pi.Files))
	buf := make([]byte, len(b.Bytes()))
	copy(buf, b.Bytes())
	bufPool.Put(b)
	return buf, errs.Err()
}

func gobDecodeInfo(dec *gob.Decoder) ([]Info, error) {
	errs := errors.M{}
	var size int
	err := dec.Decode(&size)
	if err != nil {
		return nil, err
	}
	info := make([]Info, size)
	for i := 0; i < size; i++ {
		errs.Append(dec.Decode(&info[i].Name))
		errs.Append(dec.Decode(&info[i].UserID))
		errs.Append(dec.Decode(&info[i].Size))
		errs.Append(dec.Decode(&info[i].ModTime))
		errs.Append(dec.Decode(&info[i].Mode))
	}
	return info, errs.Err()
}

// GobDecode implements gob.Decoder.
func (pi *PrefixInfo) GobDecode(buf []byte) error {
	dec := gob.NewDecoder(bytes.NewBuffer(buf))
	errs := errors.M{}
	errs.Append(dec.Decode(&pi.ModTime))
	errs.Append(dec.Decode(&pi.Size))
	errs.Append(dec.Decode(&pi.UserID))
	errs.Append(dec.Decode(&pi.Mode))
	errs.Append(dec.Decode(&pi.DiskUsage))
	errs.Append(dec.Decode(&pi.Err))
	var err error
	pi.Children, err = gobDecodeInfo(dec)
	errs.Append(err)
	pi.Files, err = gobDecodeInfo(dec)
	errs.Append(err)
	return errs.Err()
}

// Metric represents a value associated with a prefix.
type Metric struct {
	Prefix string
	Value  int64
}

// MetricOptions is configured by instances of MetricOption.
type MetricOptions struct {
	Global bool
	UserID string
}

// MetricOption is used to request particular metrics, either per-user
// or global to the entire database.
type MetricOption func(o *MetricOptions)

func Global() MetricOption {
	return func(o *MetricOptions) {
		o.Global = true
		o.UserID = ""
	}
}

func UserID(userID string) MetricOption {
	return func(o *MetricOptions) {
		o.Global = false
		o.UserID = userID
	}
}

// MetricName names a particular metric supported by instances of Database.
type MetricName string

const (
	// TotalFileCount refers to the total # of files in the database.
	TotalFileCount MetricName = "totalFileCount"
	// TotalPrefixCount refers to the total # of prefixes/directories in
	// the database. For cloud based filesystems the prefixes are likely
	// purely naming conventions as opposed to local filesystem directories.
	TotalPrefixCount MetricName = "totalPrefixCount"
	// TotalDiskUsage refers to the total disk usage of the files and prefixes
	// in the database taking the filesystems block size into account.
	TotalDiskUsage MetricName = "totalDiskUsage"
)

// DatabaseOptions represents options common to all database implementations.
type DatabaseOptions struct {
	ResetStats bool
	ReadOnly   bool
}

// DatabaseOption represent a specific option common to all databases.
type DatabaseOption func(o *DatabaseOptions)

// ResetStats requests that the database reset its statistics when opened.
func ResetStats() DatabaseOption {
	return func(o *DatabaseOptions) {
		o.ResetStats = true
	}
}

// ReadOnly requests that the database be opened in read only mode.
func ReadOnly() DatabaseOption {
	return func(o *DatabaseOptions) {
		o.ReadOnly = true
	}
}

// ScannerOptions represents the options common to all scanner implementations.
type ScannerOptions struct {
	Descending bool
	RangeScan  bool
	KeysOnly   bool
	ScanErrors bool
	ScanLimit  int
}

// ScannerOption represent a specific option common to all scanners.
type ScannerOption func(so *ScannerOptions)

// ScanDescending requests a descending scan, the default is ascending.
func ScanDescending() ScannerOption {
	return func(so *ScannerOptions) {
		so.Descending = true
	}
}

func RangeScan() ScannerOption {
	return func(so *ScannerOptions) {
		so.RangeScan = true
	}
}

func KeysOnly() ScannerOption {
	return func(so *ScannerOptions) {
		so.KeysOnly = true
	}
}

func ScanErrors() ScannerOption {
	return func(so *ScannerOptions) {
		so.ScanErrors = true
	}
}

func ScanLimit(l int) ScannerOption {
	return func(so *ScannerOptions) {
		so.ScanLimit = l
	}
}

// Database is the interface to be implemented by a database suitable for
// use with filewalk.
type Database interface {
	// Set stores the specified information in the database taking care to
	// update all metrics. If PrefixInfo specifies a UserID then the metrics
	// associated with that user will be updated in addition to global ones.
	// Metrics are updated approriately for
	Set(ctx context.Context, prefix string, info *PrefixInfo) error
	// Get returns the information stored for the specified prefix. It will
	// return false if the entry does not exist in the database but with
	// a nil error.
	Get(ctx context.Context, prefix string, info *PrefixInfo) (bool, error)
	// Save saves the database to persistent storage.
	Save(ctx context.Context) error
	// Close will first Save and then release resources associated with the database.
	Close(ctx context.Context) error
	// UserIDs returns the current set of userIDs known to the database.
	UserIDs(ctx context.Context) ([]string, error)
	// Metrics returns the names of the supported metrics.
	Metrics() []MetricName
	// Total returns the total (ie. sum) for the requested metric.
	Total(ctx context.Context, name MetricName, opts ...MetricOption) (int64, error)
	// TopN returns the top-n values for the requested metric.
	TopN(ctx context.Context, name MetricName, n int, opts ...MetricOption) ([]Metric, error)
	// NewScanner creates a scanner that will start at the specified prefix
	// and scan at most limit items; a limit of 0 will scan all available
	// items.
	NewScanner(prefix string, limit int, opts ...ScannerOption) DatabaseScanner
}

// DatabaseScanner implements an idiomatic go scanner.
type DatabaseScanner interface {
	Scan(ctx context.Context) bool
	PrefixInfo() (string, *PrefixInfo)
	Err() error
}
