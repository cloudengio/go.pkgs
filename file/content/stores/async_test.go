// Copyright 2024 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package stores_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"sync"
	"testing"
	"time"

	"cloudeng.io/file/content"
	"cloudeng.io/file/content/stores"
	"cloudeng.io/file/localfs"
	"cloudeng.io/sync/synctestutil"
)

func writeObject(ctx context.Context, t *testing.T, store content.ObjectStore, prefix string, idx int) string {
	obj := content.Object[string, int]{
		Value:    fmt.Sprintf("test-0%3v", idx),
		Response: idx,
		Type:     content.Type("test"),
	}
	name := fmt.Sprintf("c-%03v", idx)
	err := obj.Store(ctx, store, prefix, name, content.JSONObjectEncoding, content.JSONObjectEncoding)
	if err != nil {
		t.Fatal(err)
	}
	return name
}

func TestAsyncWrite(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx := context.Background()
	tmpDir := t.TempDir()
	fs := localfs.New()

	root := fs.Join(tmpDir, "store")
	mkdirall(t, root)
	mkdirall(t, root, "l1", "l2")
	if _, err := fs.Stat(ctx, root); err != nil {
		t.Fatal(err)
	}

	for _, concurrency := range []int{0, 5, 10} {
		store := stores.New(fs, concurrency)

		if err := store.EraseExisting(ctx, root); err != nil {
			t.Fatal(err)
		}

		_, err := fs.Stat(ctx, root)
		if err == nil || !fs.IsNotExist(err) {
			t.Errorf("expected not to exist: %v", err)
		}

		prefix := fs.Join(root, "l1", "l2")
		for i := 0; i < 100; i++ {
			writeObject(ctx, t, store, prefix, i)
		}
		if err := store.Finish(ctx); err != nil {
			t.Fatal(err)
		}

		// It's safe to call Finish multiple times.
		if err := store.Finish(ctx); err != nil {
			t.Fatal(err)
		}

		for i := 0; i < 100; i++ {
			var obj content.Object[string, int]
			ctype, err := obj.Load(ctx, store, prefix, fmt.Sprintf("c-%03v", i))
			if err != nil {
				t.Fatal(err)
			}
			if got, want := ctype, content.Type("test"); got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := obj.Value, fmt.Sprintf("test-0%3v", i); got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := obj.Response, i; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		}
	}
}

type slowFS struct {
	content.FS
	delay time.Duration
}

func (sfs *slowFS) Put(ctx context.Context, path string, mode os.FileMode, data []byte) error {
	time.Sleep(sfs.delay)
	return sfs.FS.Put(ctx, path, mode, data)
}

func (sfs *slowFS) Get(ctx context.Context, path string) ([]byte, error) {
	time.Sleep(sfs.delay)
	return sfs.FS.Get(ctx, path)
}

func TestAsyncWriteCancel(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	tmpDir := t.TempDir()

	sfs := &slowFS{FS: localfs.New(), delay: 100 * time.Millisecond}
	prefix := sfs.Join(tmpDir, "store")
	store := stores.NewAsync(sfs, 2)

	var errCh = make(chan error, 1)
	go func() {
		for i := 0; i < 1000; i++ {
			obj := content.Object[string, int]{
				Value:    fmt.Sprintf("test-0%3v", i),
				Response: i,
				Type:     content.Type("test"),
			}
			err := obj.Store(ctx, store, prefix, fmt.Sprintf("c-%03v", i), content.JSONObjectEncoding, content.JSONObjectEncoding)
			if err != nil {
				errCh <- err
				return
			}
		}
		errCh <- store.Finish(ctx)
	}()

	time.Sleep(time.Second)
	cancel()
	err := <-errCh
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("wrong or missing error: %v", err)
	}
}

func TestAsyncWriteFinish(t *testing.T) {
	ctx := context.Background()
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	fs := localfs.New()
	store := stores.NewAsync(fs, 5)
	if err := store.Finish(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestAsyncRead(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx := context.Background()
	tmpDir := t.TempDir()
	fs := localfs.New()

	root := fs.Join(tmpDir, "store")
	syncStore := stores.NewSync(fs)
	mkdirall(t, root)
	names := []string{}
	for i := 0; i < 10; i++ {
		name := writeObject(ctx, t, syncStore, root, i)
		names = append(names, name)
	}

	for _, concurrency := range []int{0, 5, 10} {
		store := stores.New(fs, concurrency)

		var (
			objs     []content.Object[string, int]
			prefixes = map[string]struct{}{}
			found    []string
			mu       sync.Mutex
		)

		err := store.ReadV(ctx, root, names, func(_ context.Context, prefix, name string, typ content.Type, data []byte, err error) error {
			if err != nil {
				return err
			}
			if typ != content.Type("test") {
				return fmt.Errorf("unexpected type: %v", typ)
			}
			var obj content.Object[string, int]
			if err := obj.Decode(data); err != nil {
				return err
			}
			mu.Lock()
			prefixes[prefix] = struct{}{}
			objs = append(objs, obj)
			found = append(found, name)
			mu.Unlock()
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		sort.Slice(objs, func(i, j int) bool {
			return objs[i].Response < objs[j].Response
		})

		sort.Strings(found)
		if got, want := found, names; !reflect.DeepEqual(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}

		if got, want := len(prefixes), 1; got != want {
			t.Errorf("got %v, want %v", got, want)
		}
		if _, ok := prefixes[root]; !ok {
			t.Errorf("missing prefix: %v", root)
		}

		for i := 0; i < len(names); i++ {
			o := objs[i]
			if got, want := o.Type, content.Type("test"); got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := o.Value, fmt.Sprintf("test-0%3v", i); got != want {
				t.Errorf("got %v, want %v", got, want)
			}
			if got, want := o.Response, i; got != want {
				t.Errorf("got %v, want %v", got, want)
			}
		}

	}
}

func TestAsyncReadError(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx := context.Background()
	tmpDir := t.TempDir()
	fs := localfs.New()
	store := stores.NewAsync(fs, 5)

	root := fs.Join(tmpDir, "store")
	err := store.ReadV(ctx, root, []string{"a", "b", "c"}, func(_ context.Context, _, _ string, _ content.Type, _ []byte, err error) error {
		time.Sleep(100 * time.Millisecond)
		return err
	})
	if !fs.IsNotExist(err) {
		t.Fatalf("unexpected or missing error: %v %T", err, err)
	}
}

func TestAsyncReadCancel(t *testing.T) {
	defer synctestutil.AssertNoGoroutinesRacy(t, time.Second)()
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	tmpDir := t.TempDir()
	fs := localfs.New()
	store := stores.NewAsync(fs, 5)

	errCh := make(chan error)
	go func() {
		errCh <- store.ReadV(ctx, tmpDir, []string{"a", "b", "c"}, func(_ context.Context, _, _ string, _ content.Type, _ []byte, err error) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
			}
			return err
		})
	}()

	go cancel()
	if err := <-errCh; !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected or missing error: %v %T", err, err)
	}

}
