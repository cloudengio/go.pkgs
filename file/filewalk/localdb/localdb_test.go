// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package localdb_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"cloudeng.io/file/filewalk"
	"cloudeng.io/file/filewalk/localdb"
)

func TestDBSimple(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbDir := filepath.Join(tmpDir, "first")
	var err error
	assert := func() {
		if err != nil {
			_, _, line, _ := runtime.Caller(1)
			t.Fatalf("line %v: %v", line, err)
		}
	}
	var users []string
	assertUsers := func(u int) {
		if err != nil {
			_, _, line, _ := runtime.Caller(1)
			t.Fatalf("line %v: %v", line, err)
		}
		if got, want := len(users), u; got != want {
			_, _, line, _ := runtime.Caller(1)
			t.Errorf("line: %v: got %v, want %v", line, got, want)
		}
	}
	db, err := localdb.Open(ctx, dbDir, nil)
	assert()
	if got, want := len(db.Metrics()), 4; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	users, err = db.UserIDs(ctx)
	assertUsers(0)
	if got, want := len(users), 0; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	now := time.Now()
	pi := filewalk.PrefixInfo{
		ModTime:   now,
		UserID:    "500",
		DiskUsage: 33,
	}
	err = db.Set(ctx, "/a/b", &pi)
	assert()
	users, err = db.UserIDs(ctx)
	assertUsers(1)
	pi.DiskUsage = 40
	pi.Files = []filewalk.Info{
		{},
		{},
	}
	pi.Children = []filewalk.Info{
		{},
	}
	err = db.Set(ctx, "/a/b", &pi)
	assert()
	err = db.Set(ctx, "/a/b/c", &pi)
	assert()
	pi.UserID = "501"
	err = db.Set(ctx, "/a/b/c", &pi)
	assert()

	users, err = db.UserIDs(ctx)
	assertUsers(2)

	total := func(name filewalk.MetricName, opts ...filewalk.MetricOption) int64 {
		v, err := db.Total(ctx, name, opts...)
		if err != nil {
			_, _, line, _ := runtime.Caller(1)
			t.Fatalf("line %v: %v", line, err)
		}
		return v
	}

	for _, tc := range []struct {
		name filewalk.MetricName
		val  int64
	}{
		{filewalk.TotalFileCount, 4},
		{filewalk.TotalPrefixCount, 2},
		{filewalk.TotalDiskUsage, 80},
	} {
		if got, want := total(tc.name, filewalk.Global()), tc.val; got != want {
			t.Errorf("%v: got %v, want %v", tc.name, got, want)
		}
	}

	err = db.Close(ctx)
	assert()

	db, err = localdb.Open(ctx, dbDir, nil)
	assert()
	defer db.Close(ctx)

	if got, want := total(filewalk.TotalDiskUsage, filewalk.Global()), int64(80); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	users, err = db.UserIDs(ctx)
	assertUsers(2)

	keys := []string{}
	sc := db.NewScanner("", 0)
	for sc.Scan(ctx) {
		k, _ := sc.PrefixInfo()
		keys = append(keys, k)
	}
	err = sc.Err()
	assert()
	if got, want := keys, []string{"/a/b", "/a/b/c"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDBLocking(t *testing.T) {
	ctx := context.Background()

	tmpDir := t.TempDir()
	dbDir := filepath.Join(tmpDir, "first")
	db, err := localdb.Open(ctx, dbDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	db.Close(ctx)

	db, err = localdb.Open(ctx, dbDir, nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = localdb.Open(ctx, dbDir, nil,
		localdb.LockStatusDelay(time.Millisecond*100),
		localdb.TryLock(),
	)
	if err == nil || !strings.Contains(err.Error(), "failed to lock") {
		t.Fatalf("missing or unexpected error: %v", err)
	}

	_, err = localdb.Open(ctx, dbDir,
		[]filewalk.DatabaseOption{filewalk.ReadOnly()},
		localdb.LockStatusDelay(time.Millisecond*100),
		localdb.TryLock(),
	)
	if err == nil || !strings.Contains(err.Error(), "failed to lock") {
		t.Fatalf("missing or unexpected error: %v", err)
	}

	fmt.Printf("should unblock the previous lock....\n")
	if err := db.Close(ctx); err != nil {
		t.Fatal(err)
	}

	time.Sleep(time.Second)

	dbDir = filepath.Join(tmpDir, "second")
	if err := os.MkdirAll(dbDir, 0766); err != nil {
		t.Fatal(err)
	}

	ndbd, err := localdb.Open(ctx, dbDir,
		[]filewalk.DatabaseOption{},
		localdb.LockStatusDelay(time.Millisecond*100),
		localdb.TryLock(),
	)
	if err != nil {
		t.Fatal(err)
	}
	ndbd.Close(ctx)

	dbr1, err := localdb.Open(ctx, dbDir,
		[]filewalk.DatabaseOption{filewalk.ReadOnly()},
		localdb.LockStatusDelay(time.Millisecond*100),
		localdb.TryLock(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer dbr1.Close(ctx)

	dbr2, err := localdb.Open(ctx, dbDir,
		[]filewalk.DatabaseOption{filewalk.ReadOnly()},
		localdb.LockStatusDelay(time.Millisecond*100),
		localdb.TryLock(),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer dbr2.Close(ctx)
	buf := make([]byte, 1024*1024)
	n := runtime.Stack(buf, true)
	fmt.Printf("%s\n", string(buf[:n]))
}

func fill(ctx context.Context, db filewalk.Database, when time.Time, n int) error {
	for i := 0; i < n; i++ {
		p := fmt.Sprintf("/a/%05v", i)
		pi := &filewalk.PrefixInfo{
			ModTime: when,
		}
		if err := db.Set(ctx, p, pi); err != nil {
			return err
		}
	}
	return nil
}

func TestScanning(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()
	dbDir := filepath.Join(tmpDir, "first")
	db, err := localdb.Open(ctx, dbDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	nitems := 3333
	when := time.Now()
	if err := fill(ctx, db, when, nitems); err != nil {
		t.Fatal(err)
	}
	db.Close(ctx)

	db, err = localdb.Open(ctx, dbDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close(ctx)

	keys := []string{}
	limit := 10
	sc := db.NewScanner("", limit, filewalk.ScanLimit(6))
	for sc.Scan(ctx) {
		k, v := sc.PrefixInfo()
		keys = append(keys, k)
		if got, want := v.ModTime, when; !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < limit; i++ {
		if got, want := keys[i], fmt.Sprintf("/a/%05v", i); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	sc = db.NewScanner("", 3333, filewalk.ScanLimit(6))
	keys = []string{}
	for sc.Scan(ctx) {
		k, v := sc.PrefixInfo()
		keys = append(keys, k)
		if got, want := v.ModTime, when; !got.Equal(want) {
			t.Errorf("got %v, want %v", got, want)
		}
	}

	if got, want := len(keys), nitems; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	for i := 0; i < nitems; i++ {
		if got, want := keys[i], fmt.Sprintf("/a/%05v", i); got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	}
}
