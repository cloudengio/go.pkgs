// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ociimage

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// ErrInvalidReference is returned by Parse for a reference it cannot make
// sense of. Use errors.Is to test for it; the error itself says which
// reference was rejected and why.
var ErrInvalidReference = errors.New("invalid OCI image reference")

// Reference is a parsed OCI image reference. It is purely syntactic: it
// records what the reference says rather than what a particular tool will do
// with it, and String returns it unchanged. Every field is a string, so a
// Reference is comparable and can be used as a map key.
//
// The zero value is the empty reference, which no valid reference parses to,
// see IsZero.
type Reference struct {
	// Registry is the host the image is pulled from, with its port if it has
	// one, or empty if the reference names none, see Remote.
	Registry string
	// Repository is the path between the registry and the name, with no
	// leading or trailing "/", or empty if there is none: "cirruslabs" for
	// "ghcr.io/cirruslabs/ubuntu".
	Repository string
	// Name is the image name alone, which is what something named after the
	// image uses: a tart VM name and a docker container name may not contain
	// the "/" and ":" of a full reference.
	Name string
	// Tag is the tag, or empty if the reference has none. It is not defaulted
	// to "latest": what the reference says and what it implies are kept
	// apart, so that String round-trips.
	Tag string
	// Digest is the "algorithm:hex" digest, or empty if the reference has
	// none. A reference may carry a tag and a digest both, in which case it
	// is the digest that identifies the image.
	Digest string
}

// Parse parses an OCI image reference.
//
// The leading segment is taken to be a registry if it looks like a host: it
// contains a "." or a ":", or is exactly "localhost". Anything else is part
// of the repository path, so "myorg/linux-ci" names no registry, see Remote.
func Parse(s string) (Reference, error) {
	if s == "" {
		return Reference{}, fmt.Errorf("%w: the reference is empty", ErrInvalidReference)
	}
	if strings.ContainsAny(s, " \t\r\n") {
		return Reference{}, fmt.Errorf("%w: %q: contains whitespace", ErrInvalidReference, s)
	}
	var ref Reference
	rest, digest, hasDigest := strings.Cut(s, "@")
	if hasDigest {
		if err := validateDigest(s, digest); err != nil {
			return Reference{}, err
		}
		ref.Digest = digest
	}
	segments := strings.Split(rest, "/")
	if slices.Contains(segments, "") {
		return Reference{}, fmt.Errorf("%w: %q: contains an empty path segment", ErrInvalidReference, s)
	}
	// A host in the leading position is the registry; anything else there is
	// the first segment of the repository path.
	if len(segments) > 1 && isRegistry(segments[0]) {
		ref.Registry, segments = segments[0], segments[1:]
	}
	last := segments[len(segments)-1]
	ref.Repository = strings.Join(segments[:len(segments)-1], "/")

	// The registry, which is the only other part that may carry a ":", has
	// already been taken off, so a ":" here introduces the tag.
	name, tag, hasTag := strings.Cut(last, ":")
	if hasTag {
		if tag == "" {
			return Reference{}, fmt.Errorf("%w: %q: the tag is empty", ErrInvalidReference, s)
		}
		if strings.Contains(tag, ":") {
			return Reference{}, fmt.Errorf("%w: %q: the tag %q contains a %q", ErrInvalidReference, s, tag, ":")
		}
		ref.Tag = tag
	}
	if name == "" {
		return Reference{}, fmt.Errorf("%w: %q: the image name is empty", ErrInvalidReference, s)
	}
	ref.Name = name
	return ref, nil
}

// isRegistry reports whether a leading segment names a registry host rather
// than the first segment of a repository path. A host is distinguishable by
// carrying a domain or a port, or by being localhost.
func isRegistry(segment string) bool {
	return strings.ContainsAny(segment, ".:") || segment == "localhost"
}

// validateDigest checks that digest is an "algorithm:hex" pair, which is as
// far as this package goes: whether the algorithm is one the tool supports,
// and whether the hex is the right length for it, is for the tool to say.
func validateDigest(s, digest string) error {
	algorithm, hex, ok := strings.Cut(digest, ":")
	switch {
	case !ok:
		return fmt.Errorf("%w: %q: the digest %q is not an algorithm:hex pair", ErrInvalidReference, s, digest)
	case algorithm == "":
		return fmt.Errorf("%w: %q: the digest names no algorithm", ErrInvalidReference, s)
	case hex == "":
		return fmt.Errorf("%w: %q: the digest %q has no value", ErrInvalidReference, s, digest)
	}
	for _, r := range hex {
		if !isHexDigit(r) {
			return fmt.Errorf("%w: %q: the digest %q is not hexadecimal", ErrInvalidReference, s, digest)
		}
	}
	return nil
}

func isHexDigit(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

// String returns the reference, which for a Reference obtained from Parse is
// the string it was parsed from.
func (r Reference) String() string {
	var out strings.Builder
	if r.Registry != "" {
		out.WriteString(r.Registry)
		out.WriteByte('/')
	}
	if r.Repository != "" {
		out.WriteString(r.Repository)
		out.WriteByte('/')
	}
	out.WriteString(r.Name)
	if r.Tag != "" {
		out.WriteByte(':')
		out.WriteString(r.Tag)
	}
	if r.Digest != "" {
		out.WriteByte('@')
		out.WriteString(r.Digest)
	}
	return out.String()
}

// IsZero reports whether the reference is the zero value, which is what a
// Reference that was never parsed holds: no valid reference parses to it.
func (r Reference) IsZero() bool {
	return r == Reference{}
}

// Remote reports whether the reference names a registry explicitly, and so
// has somewhere to be pulled from.
//
// A reference that names none is local to whichever tool resolves it: tart
// looks for it in its own store, whereas a docker-like tool resolves it
// against a default registry of its own, see WithDefaultRegistry. A
// reference carrying a digest is remote whatever else it says, since a digest
// identifies content in a registry rather than anything a local store names.
func (r Reference) Remote() bool {
	return r.Registry != "" || r.Digest != ""
}

// Immutable reports whether the reference can only ever resolve to one
// image, which is true of a digest and of nothing else: any tag may be
// pushed over, ":latest" by habit and the rest by mistake.
func (r Reference) Immutable() bool {
	return r.Digest != ""
}

// Path returns the reference without its registry, tag or digest: the
// repository path and the image name, which is what identifies the image
// within whichever registry holds it.
func (r Reference) Path() string {
	if r.Repository == "" {
		return r.Name
	}
	return r.Repository + "/" + r.Name
}

// WithRegistry returns the reference with its registry replaced, or removed
// if host is empty.
func (r Reference) WithRegistry(host string) Reference {
	r.Registry = host
	return r
}

// WithDefaultRegistry returns the reference with host as its registry if it
// names none, and unchanged otherwise. It is how a tool that resolves a
// reference against a registry of its own, as a docker-like tool resolves a
// bare name against Docker Hub, says so.
func (r Reference) WithDefaultRegistry(host string) Reference {
	if r.Registry == "" {
		r.Registry = host
	}
	return r
}

// WithTag returns the reference with its tag replaced, or removed if tag is
// empty. The digest, if any, is left as it is; it is the digest that
// identifies the image when a reference carries both.
func (r Reference) WithTag(tag string) Reference {
	r.Tag = tag
	return r
}

// WithDigest returns the reference with its digest replaced, or removed if
// digest is empty. The digest is not validated, unlike one that Parse reads.
func (r Reference) WithDigest(digest string) Reference {
	r.Digest = digest
	return r
}
