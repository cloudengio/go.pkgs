// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ociimage_test

import (
	"errors"
	"testing"

	"cloudeng.io/vms/ociimage"
)

// corpus is the set of references the package is expected to handle, with
// what each one parses to. It is used by the parse, round-trip and predicate
// tests, so that every reference is checked by all of them.
var corpus = []struct {
	ref  string
	want ociimage.Reference
}{
	// A bare local name, which is what a locally built image is referred to by.
	{"linux-ci", ociimage.Reference{Name: "linux-ci"}},
	{"macos-ci", ociimage.Reference{Name: "macos-ci"}},
	// A tag on a local name.
	{"linux-ci:latest", ociimage.Reference{Name: "linux-ci", Tag: "latest"}},
	// A leading segment that is not a host is part of the repository path, so
	// this names no registry.
	{"myorg/linux-ci", ociimage.Reference{Repository: "myorg", Name: "linux-ci"}},
	// A host with a port, with and without a tag.
	{"mm-1:5001/linux-ci:latest", ociimage.Reference{Registry: "mm-1:5001", Name: "linux-ci", Tag: "latest"}},
	{"mm-1:5001/linux-ci", ociimage.Reference{Registry: "mm-1:5001", Name: "linux-ci"}},
	{"localhost:5000/macos-ci:1.2.3", ociimage.Reference{Registry: "localhost:5000", Name: "macos-ci", Tag: "1.2.3"}},
	// localhost is a host even without a port.
	{"localhost/macos-ci", ociimage.Reference{Registry: "localhost", Name: "macos-ci"}},
	// A domain, with and without a repository path.
	{"ghcr.io/cirruslabs/macos-sequoia-base:latest", ociimage.Reference{
		Registry: "ghcr.io", Repository: "cirruslabs", Name: "macos-sequoia-base", Tag: "latest"}},
	{"ghcr.io/cirruslabs/ubuntu", ociimage.Reference{
		Registry: "ghcr.io", Repository: "cirruslabs", Name: "ubuntu"}},
	{"registry.example.com/team/img", ociimage.Reference{
		Registry: "registry.example.com", Repository: "team", Name: "img"}},
	// A repository path of more than one segment.
	{"ghcr.io/org/team/img:v1", ociimage.Reference{
		Registry: "ghcr.io", Repository: "org/team", Name: "img", Tag: "v1"}},
	// A digest, alone and with a registry.
	{"linux-ci@sha256:0123456789abcdef", ociimage.Reference{
		Name: "linux-ci", Digest: "sha256:0123456789abcdef"}},
	{"mm-1:5001/linux-ci@sha256:0123456789abcdef", ociimage.Reference{
		Registry: "mm-1:5001", Name: "linux-ci", Digest: "sha256:0123456789abcdef"}},
	// A tag and a digest together, which the specification allows: the digest
	// is what identifies the image and the tag is a hint at where it came
	// from.
	{"ghcr.io/cirruslabs/ubuntu:latest@sha256:0123456789ABCDEF", ociimage.Reference{
		Registry: "ghcr.io", Repository: "cirruslabs", Name: "ubuntu", Tag: "latest",
		Digest: "sha256:0123456789ABCDEF"}},
	// A lone host-looking segment is an image name and a tag, not a registry:
	// a registry with no image after it names nothing.
	{"mm-1:5001", ociimage.Reference{Name: "mm-1", Tag: "5001"}},
}

func TestParse(t *testing.T) {
	for _, tc := range corpus {
		got, err := ociimage.Parse(tc.ref)
		if err != nil {
			t.Errorf("Parse(%q): %v", tc.ref, err)
			continue
		}
		if got != tc.want {
			t.Errorf("Parse(%q):\n got %+v\nwant %+v", tc.ref, got, tc.want)
		}
	}
}

// TestParseRoundTrip covers the reference surviving a parse unchanged, which
// is what lets a parsed reference be handed to a tool: the tool must see what
// its user wrote, not an interpretation of it.
func TestParseRoundTrip(t *testing.T) {
	for _, tc := range corpus {
		ref, err := ociimage.Parse(tc.ref)
		if err != nil {
			t.Errorf("Parse(%q): %v", tc.ref, err)
			continue
		}
		if got := ref.String(); got != tc.ref {
			t.Errorf("String: got %q, want %q", got, tc.ref)
		}
	}
}

// TestParseErrors covers the references that are rejected: those that are
// structurally not references at all. Anything beyond that — whether the
// image exists, whether its name obeys the whole of the specification — is
// left to the tool that resolves it.
func TestParseErrors(t *testing.T) {
	for _, ref := range []string{
		"",                                 // nothing at all
		"/",                                // no segments
		"/linux-ci",                        // leading separator
		"linux-ci/",                        // trailing separator
		"ghcr.io//ubuntu",                  // empty path segment
		"linux-ci:",                        // a tag introduced and not given
		"linux-ci:a:b",                     // two tags
		"ghcr.io/ubuntu:",                  // the same, with a registry
		":latest",                          // a tag with no image to put it on
		"ghcr.io/:latest",                  // the same, with a registry
		":",                                // neither one nor the other
		"linux-ci@",                        // a digest introduced and not given
		"linux-ci@sha256:",                 // a digest with no value
		"linux-ci@:0123",                   // a digest with no algorithm
		"linux-ci@latest",                  // a tag where a digest belongs
		"linux-ci@sha256:nothexatall",      // a digest that is not hexadecimal
		"linux-ci@sha256:0123@sha256:4567", // two digests
		"linux ci",                         // whitespace
		"ghcr.io/cirruslabs/ubuntu:la test",
	} {
		got, err := ociimage.Parse(ref)
		if err == nil {
			t.Errorf("Parse(%q): got %+v, want an error", ref, got)
			continue
		}
		if !errors.Is(err, ociimage.ErrInvalidReference) {
			t.Errorf("Parse(%q): got %v, want ErrInvalidReference", ref, err)
		}
		if !got.IsZero() {
			t.Errorf("Parse(%q): got %+v alongside the error, want the zero reference", ref, got)
		}
	}
}

// TestReferenceRemote covers which references name a registry and so can be
// pulled. A bare or namespaced name cannot: what it means is for the tool
// that resolves it to say.
func TestReferenceRemote(t *testing.T) {
	for _, tc := range []struct {
		ref  string
		want bool
	}{
		// Local to whichever tool resolves them.
		{"linux-ci", false},
		{"linux-ci:latest", false},
		{"myorg/linux-ci", false},
		// A registry, named by a domain, a port or localhost.
		{"mm-1:5001/linux-ci:latest", true},
		{"localhost:5000/macos-ci", true},
		{"localhost/macos-ci", true},
		{"ghcr.io/cirruslabs/ubuntu:latest", true},
		{"registry.example.com/team/img", true},
		// A digest identifies content in a registry, whatever else the
		// reference says.
		{"linux-ci@sha256:0123456789abcdef", true},
	} {
		ref, err := ociimage.Parse(tc.ref)
		if err != nil {
			t.Errorf("Parse(%q): %v", tc.ref, err)
			continue
		}
		if got := ref.Remote(); got != tc.want {
			t.Errorf("%q: Remote() = %v, want %v", tc.ref, got, tc.want)
		}
	}
}

// TestReferenceName covers the image name alone, which is what something
// named after the image uses: a VM or container name may contain neither the
// "/" of a repository path nor the ":" of a tag.
func TestReferenceName(t *testing.T) {
	for _, tc := range []struct{ ref, want string }{
		{"linux-ci", "linux-ci"},
		{"linux-ci:latest", "linux-ci"},
		{"myorg/linux-ci", "linux-ci"},
		{"mm-1:5001/linux-ci:latest", "linux-ci"},
		{"mm-1:5001/linux-ci", "linux-ci"},
		{"localhost:5000/macos-ci:1.2.3", "macos-ci"},
		{"ghcr.io/cirruslabs/macos-sequoia-base:latest", "macos-sequoia-base"},
		{"ghcr.io/cirruslabs/ubuntu", "ubuntu"},
		{"linux-ci@sha256:0123456789abcdef", "linux-ci"},
		{"mm-1:5001/linux-ci@sha256:0123456789abcdef", "linux-ci"},
	} {
		ref, err := ociimage.Parse(tc.ref)
		if err != nil {
			t.Errorf("Parse(%q): %v", tc.ref, err)
			continue
		}
		if got := ref.Name; got != tc.want {
			t.Errorf("%q: Name = %q, want %q", tc.ref, got, tc.want)
		}
	}
}

// TestReferenceImmutable covers the one reference that can only ever resolve
// to a single image.
func TestReferenceImmutable(t *testing.T) {
	for _, tc := range []struct {
		ref  string
		want bool
	}{
		{"linux-ci", false},
		{"linux-ci:latest", false},
		{"ghcr.io/cirruslabs/ubuntu:v1.2.3", false}, // a version tag may still be pushed over
		{"linux-ci@sha256:0123456789abcdef", true},
		{"ghcr.io/cirruslabs/ubuntu:latest@sha256:0123456789abcdef", true},
	} {
		ref, err := ociimage.Parse(tc.ref)
		if err != nil {
			t.Errorf("Parse(%q): %v", tc.ref, err)
			continue
		}
		if got := ref.Immutable(); got != tc.want {
			t.Errorf("%q: Immutable() = %v, want %v", tc.ref, got, tc.want)
		}
	}
}

func TestReferencePath(t *testing.T) {
	for _, tc := range []struct{ ref, want string }{
		{"linux-ci", "linux-ci"},
		{"linux-ci:latest", "linux-ci"},
		{"myorg/linux-ci", "myorg/linux-ci"},
		{"mm-1:5001/linux-ci:latest", "linux-ci"},
		{"ghcr.io/cirruslabs/ubuntu:latest", "cirruslabs/ubuntu"},
		{"ghcr.io/org/team/img:v1", "org/team/img"},
		{"linux-ci@sha256:0123456789abcdef", "linux-ci"},
	} {
		ref, err := ociimage.Parse(tc.ref)
		if err != nil {
			t.Errorf("Parse(%q): %v", tc.ref, err)
			continue
		}
		if got := ref.Path(); got != tc.want {
			t.Errorf("%q: Path() = %q, want %q", tc.ref, got, tc.want)
		}
	}
}

// TestReferenceBuilders covers deriving one reference from another, which is
// how a tool applies a registry of its own and how a caller pins or moves a
// version. Each returns a copy, leaving the reference it was called on as it
// was.
func TestReferenceBuilders(t *testing.T) {
	ref, err := ociimage.Parse("myorg/linux-ci:latest")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// A tool that resolves a bare reference against a registry of its own
	// says so with WithDefaultRegistry, which is the docker reading of a
	// reference that tart would treat as local.
	if got, want := ref.WithDefaultRegistry("docker.io").String(), "docker.io/myorg/linux-ci:latest"; got != want {
		t.Errorf("WithDefaultRegistry: got %q, want %q", got, want)
	}
	// An explicit registry is left alone by it, and replaced by
	// WithRegistry.
	pinned := ref.WithRegistry("ghcr.io")
	if got, want := pinned.WithDefaultRegistry("docker.io").String(), "ghcr.io/myorg/linux-ci:latest"; got != want {
		t.Errorf("WithDefaultRegistry over an explicit registry: got %q, want %q", got, want)
	}
	if got, want := pinned.WithRegistry("").String(), "myorg/linux-ci:latest"; got != want {
		t.Errorf("WithRegistry(\"\"): got %q, want %q", got, want)
	}

	if got, want := ref.WithTag("v2").String(), "myorg/linux-ci:v2"; got != want {
		t.Errorf("WithTag: got %q, want %q", got, want)
	}
	if got, want := ref.WithTag("").String(), "myorg/linux-ci"; got != want {
		t.Errorf("WithTag(\"\"): got %q, want %q", got, want)
	}
	// Pinning to a digest keeps the tag, which the digest overrides, and
	// makes the reference immutable.
	digested := ref.WithDigest("sha256:0123456789abcdef")
	if got, want := digested.String(), "myorg/linux-ci:latest@sha256:0123456789abcdef"; got != want {
		t.Errorf("WithDigest: got %q, want %q", got, want)
	}
	if !digested.Immutable() || !digested.Remote() {
		t.Errorf("a digested reference is immutable and remote: %+v", digested)
	}
	if got, want := digested.WithDigest("").String(), "myorg/linux-ci:latest"; got != want {
		t.Errorf("WithDigest(\"\"): got %q, want %q", got, want)
	}

	// None of them altered the reference they were called on.
	if got, want := ref.String(), "myorg/linux-ci:latest"; got != want {
		t.Errorf("the original reference was modified: got %q, want %q", got, want)
	}
}

// TestReferenceComparable covers a Reference being usable as a map key, which
// is what lets something keyed by image avoid holding the string as well.
func TestReferenceComparable(t *testing.T) {
	seen := map[ociimage.Reference]int{}
	for _, tc := range corpus {
		ref, err := ociimage.Parse(tc.ref)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.ref, err)
		}
		seen[ref]++
		again, err := ociimage.Parse(tc.ref)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.ref, err)
		}
		if ref != again {
			t.Errorf("%q: two parses of one reference differ", tc.ref)
		}
	}
	if got, want := len(seen), len(corpus); got != want {
		t.Errorf("got %v distinct references, want %v: the corpus has a duplicate or two differ only in their string", got, want)
	}
	for ref, n := range seen {
		if n != 1 {
			t.Errorf("%v was counted %v times", ref, n)
		}
	}
}

// TestReferenceZero covers the zero value, which no reference parses to, so
// that it can stand for "no image".
func TestReferenceZero(t *testing.T) {
	var zero ociimage.Reference
	if !zero.IsZero() {
		t.Error("the zero value does not report itself as zero")
	}
	if got := zero.String(); got != "" {
		t.Errorf("the zero value renders as %q, want %q", got, "")
	}
	if zero.Remote() || zero.Immutable() {
		t.Error("the zero value is neither remote nor immutable")
	}
	for _, tc := range corpus {
		ref, err := ociimage.Parse(tc.ref)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.ref, err)
		}
		if ref.IsZero() {
			t.Errorf("%q parsed to the zero reference", tc.ref)
		}
	}
}
