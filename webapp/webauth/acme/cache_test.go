package acme_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloudeng.io/webapp/webauth/acme"
)

func TestCacheFactory(t *testing.T) {
	ctx := context.Background()
	td := filepath.Join(t.TempDir(), "cache")
	if got, want := acme.AutoCertDiskStore.Type(), "autocert-dir-cache"; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	contents := []byte{0x11, 0x22}
	dc := acme.NewDirCache(td, false)
	if err := dc.Put(ctx, "my.domain", contents); err != nil {
		t.Fatal(err)
	}

	buf, err := os.ReadFile(filepath.Join(td, "my.domain"))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := buf, contents; !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)

	}

	store, err := acme.AutoCertDiskStore.New(ctx, td)
	if err != nil {
		t.Fatal(err)
	}

	buf, err = store.Get(ctx, "my.domain")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := buf, contents; !bytes.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	_, err = store.Get(ctx, "my.domain-not-there")
	if err == nil || !strings.Contains(err.Error(), "cache miss") {
		t.Fatalf("missing or unexpected error: %v", err)
	}

	dc.Delete(ctx, "my.domain")
	_, err = store.Get(ctx, "my.domain")
	if err == nil || !strings.Contains(err.Error(), "cache miss") {
		t.Fatalf("missing or unexpected error: %v", err)
	}
}
