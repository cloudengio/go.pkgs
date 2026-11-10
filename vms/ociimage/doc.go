// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package ociimage parses and carries references to images in the Open
// Container Initiative format, which is the format every VM and container
// technology supported here uses: tart, docker and anything else that pulls
// an image from a registry.
//
// A Reference is the parse of a reference such as "linux-ci",
// "linux-ci:latest", "ghcr.io/cirruslabs/ubuntu:latest" or
// "mm-1:5001/linux-ci@sha256:...", split into the registry, the repository
// path, the image name and the tag or digest that identifies the version.
// Parsing is purely syntactic: a Reference records what the reference says,
// and String returns it unchanged, so that the reference handed to a tool is
// the one its user wrote.
//
// Whether a reference is local is the one rule worth stating. A registry is
// recognised by its host looking like one — it contains a "." or a ":", or is
// exactly "localhost" — and anything else in the leading position is part of
// the repository path. So "myorg/linux-ci" names no registry, and what that
// means is the using tool's business, not this package's: tart resolves such
// a reference in its own local store, whereas a docker-like tool resolves it
// against a default registry of its own, see Reference.WithDefaultRegistry.
// Reference.Remote says only that a registry was named explicitly, which is
// the question "is there anywhere to pull this from".
//
// Config is the configuration of an image and how it is obtained, for a
// backend to embed in its own configuration so that every backend spells
// these the same way. Pulling itself is left to the backend, which is the
// only thing that knows how: a backend that can pull implements Puller.
//
// Nothing here validates a reference beyond its structure. An empty
// reference, an empty path segment, a tag or digest that is not one at all:
// these are rejected, since they are mistakes worth catching when the
// configuration naming them is read. Whether the image exists, and whether
// its name obeys the full grammar of the specification, is left to the tool
// that resolves it.
package ociimage
