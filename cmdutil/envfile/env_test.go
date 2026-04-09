// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package envfile_test

import (
	"os"
	"path/filepath"
	"testing"

	"cloudeng.io/cmdutil/envfile"
)

func TestExpandEnv(t *testing.T) {
	t.Setenv("TEST_HOST", "db.example.com")
	t.Setenv("TEST_PORT", "5432")

	type cfg struct {
		Host    string `use_env:""`
		Port    string `use_env:""`
		Literal string // no tag — must not be changed
	}
	s := cfg{
		Host:    "${TEST_HOST}",
		Port:    "$TEST_PORT",
		Literal: "$TEST_HOST",
	}

	var se envfile.StructEnv
	if err := se.Expand(&s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := s.Host, "db.example.com"; got != want {
		t.Errorf("Host: got %q, want %q", got, want)
	}
	if got, want := s.Port, "5432"; got != want {
		t.Errorf("Port: got %q, want %q", got, want)
	}
	if got, want := s.Literal, "$TEST_HOST"; got != want {
		t.Errorf("Literal: got %q, want %q (must not be expanded)", got, want)
	}
}

func TestExpandEnvMissing(t *testing.T) {
	os.Unsetenv("DEFINITELY_NOT_SET_XYZ")

	type cfg struct {
		Val string `use_env:""`
	}
	s := cfg{Val: "${DEFINITELY_NOT_SET_XYZ}"}
	var se envfile.StructEnv
	if err := se.Expand(&s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := s.Val, ""; got != want {
		t.Errorf("Val: got %q, want %q", got, want)
	}
}

func TestExpandEnvMixed(t *testing.T) {
	t.Setenv("APP_NAME", "myapp")
	t.Setenv("APP_ENV", "prod")

	type cfg struct {
		Label string `use_env:""`
	}
	s := cfg{Label: "${APP_NAME}-${APP_ENV}"}
	var se envfile.StructEnv
	if err := se.Expand(&s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := s.Label, "myapp-prod"; got != want {
		t.Errorf("Label: got %q, want %q", got, want)
	}
}

func TestExpandEnvFileTag(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "secrets.env"),
		[]byte("SECRET_KEY=hunter2\nDB_URL=postgres://localhost/mydb\n"), 0600); err != nil {
		t.Fatal(err)
	}

	type credentials struct {
		Key string `use_env_file:""`
		URL string `use_env_file:""`
	}
	s := credentials{
		Key: "secrets.env:${SECRET_KEY}",
		URL: "secrets.env:$DB_URL",
	}

	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	var se envfile.StructEnv
	if err := se.Expand(&s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := s.Key, "hunter2"; got != want {
		t.Errorf("Key: got %q, want %q", got, want)
	}
	if got, want := s.URL, "postgres://localhost/mydb"; got != want {
		t.Errorf("URL: got %q, want %q", got, want)
	}
}

func TestExpandEnvFileGreedyFilename(t *testing.T) {
	// Filename contains ':' — greedy parsing should take the whole name.
	dir := t.TempDir()
	subdir := filepath.Join(dir, "a:b")
	if err := os.Mkdir(subdir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "cfg.env"),
		[]byte("TOKEN=secret\n"), 0600); err != nil {
		t.Fatal(err)
	}

	type cfg struct {
		Tok string `use_env_file:""`
	}
	s := cfg{Tok: "a:b/cfg.env:${TOKEN}"}

	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	var se envfile.StructEnv
	if err := se.Expand(&s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := s.Tok, "secret"; got != want {
		t.Errorf("Tok: got %q, want %q", got, want)
	}
}

func TestExpandEnvFileNoPattern(t *testing.T) {
	// Field value without the filename:$VAR pattern is left unchanged.
	type cfg struct {
		Val string `use_env_file:""`
	}
	s := cfg{Val: "just-a-literal"}
	var se envfile.StructEnv
	if err := se.Expand(&s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := s.Val, "just-a-literal"; got != want {
		t.Errorf("Val: got %q, want %q", got, want)
	}
}

func TestExpandEnvFileSharedCache(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "shared.env"),
		[]byte("A=alpha\nB=beta\n"), 0600); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	type first struct {
		Val string `use_env_file:""`
	}
	type second struct {
		Val string `use_env_file:""`
	}

	f := first{Val: "shared.env:$A"}
	s := second{Val: "shared.env:${B}"}

	var se envfile.StructEnv
	if err := se.Expand(&f); err != nil {
		t.Fatalf("first Expand: %v", err)
	}
	if err := se.Expand(&s); err != nil {
		t.Fatalf("second Expand: %v", err)
	}
	if got, want := f.Val, "alpha"; got != want {
		t.Errorf("first.Val: got %q, want %q", got, want)
	}
	if got, want := s.Val, "beta"; got != want {
		t.Errorf("second.Val: got %q, want %q", got, want)
	}
}

func TestExpandErrors(t *testing.T) {
	var se envfile.StructEnv
	if err := se.Expand(nil); err == nil {
		t.Error("expected error for nil")
	}
	if err := se.Expand("not a pointer"); err == nil {
		t.Error("expected error for non-pointer")
	}
	i := 42
	if err := se.Expand(&i); err == nil {
		t.Error("expected error for pointer to non-struct")
	}

	type cfg struct {
		Val string `use_env_file:""`
	}
	s := cfg{Val: "nonexistent_file_xyz.env:$X"}
	if err := se.Expand(&s); err == nil {
		t.Error("expected error for missing envfile")
	}
}

func TestExpandEmbeddedPtrStruct(t *testing.T) {
	t.Setenv("PTR_HOST", "ptr.internal")
	t.Setenv("PTR_PORT", "9999")

	type base struct {
		Host string `use_env:""`
	}
	type derived struct {
		*base
		Port string `use_env:""`
	}

	// Non-nil embedded pointer: fields must be expanded.
	s := derived{
		base: &base{Host: "$PTR_HOST"},
		Port: "$PTR_PORT",
	}
	var se envfile.StructEnv
	if err := se.Expand(&s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := s.Host, "ptr.internal"; got != want {
		t.Errorf("embedded *base.Host: got %q, want %q", got, want)
	}
	if got, want := s.Port, "9999"; got != want {
		t.Errorf("Port: got %q, want %q", got, want)
	}

	// Nil embedded pointer: must not panic.
	s2 := derived{base: nil, Port: "$PTR_PORT"}
	if err := se.Expand(&s2); err != nil {
		t.Fatalf("nil embedded pointer: unexpected error: %v", err)
	}
	if got, want := s2.Port, "9999"; got != want {
		t.Errorf("nil case Port: got %q, want %q", got, want)
	}
}

func TestExpandEmbedded(t *testing.T) {
	t.Setenv("EMBED_HOST", "db.internal")
	t.Setenv("EMBED_PORT", "5432")

	type base struct {
		Host string `use_env:""`
	}
	type derived struct {
		base
		Port string `use_env:""`
	}
	s := derived{
		base: base{Host: "$EMBED_HOST"},
		Port: "$EMBED_PORT",
	}
	var se envfile.StructEnv
	if err := se.Expand(&s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := s.Host, "db.internal"; got != want {
		t.Errorf("embedded Host: got %q, want %q", got, want)
	}
	if got, want := s.Port, "5432"; got != want {
		t.Errorf("Port: got %q, want %q", got, want)
	}
}

func TestExpandNonStringFieldsIgnored(t *testing.T) {
	t.Setenv("SOME_INT", "99")

	type cfg struct {
		Count int    `use_env:""` // int with env tag: must be silently ignored
		Name  string `use_env:""`
	}
	s := cfg{Count: 7, Name: "$SOME_INT"}
	var se envfile.StructEnv
	if err := se.Expand(&s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := s.Count, 7; got != want {
		t.Errorf("Count: got %d, want %d (must not be touched)", got, want)
	}
	if got, want := s.Name, "99"; got != want {
		t.Errorf("Name: got %q, want %q", got, want)
	}
}
