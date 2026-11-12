// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dockervm_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloudeng.io/vms"
	"cloudeng.io/vms/dockervm"
	"cloudeng.io/vms/ociimage"
)

const verifiedDigest = "sha256:0123456789abcdef"

var verifyPolicy = ociimage.Policy{Signers: []ociimage.Signer{{Key: "cosign.pub"}}}

type stubVerifier struct{ err error }

func (s stubVerifier) Verify(_ context.Context, ref ociimage.Reference, _ ociimage.Policy) (ociimage.Verified, error) {
	if s.err != nil {
		return ociimage.Verified{}, s.err
	}
	return ociimage.Verified{Reference: ref.WithDigest(verifiedDigest), Signatures: []ociimage.Signature{{Key: "cosign.pub"}}}, nil
}

// fakeDocker returns the path of a docker stub that records its arguments, and
// a function to read them.
func fakeDocker(t *testing.T) (string, func() string) {
	t.Helper()
	dir := t.TempDir()
	calls := filepath.Join(dir, "calls")
	bin := filepath.Join(dir, "docker")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\necho \"$@\" >> "+calls+"\n"), 0700); err != nil { //nolint:gosec // test stub must be executable
		t.Fatal(err)
	}
	return bin, func() string {
		b, _ := os.ReadFile(calls)
		return strings.TrimSpace(string(b))
	}
}

func TestCloneVerifies(t *testing.T) {
	bin, calls := fakeDocker(t)
	inst := dockervm.New(t.Context(), "ghcr.io/cloudengio/ci:latest", "c",
		dockervm.WithDockerBinary(bin), dockervm.WithVerification(stubVerifier{}, verifyPolicy))
	if err := inst.Clone(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := calls(); !strings.Contains(got, " ghcr.io/cloudengio/ci@"+verifiedDigest) || strings.Contains(got, ":latest") {
		t.Errorf("the verified digest was not used: %q", got)
	}

	bin, calls = fakeDocker(t)
	inst = dockervm.New(t.Context(), "ghcr.io/cloudengio/ci:latest", "c", dockervm.WithDockerBinary(bin))
	if err := inst.Clone(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := calls(); !strings.Contains(got, "ghcr.io/cloudengio/ci:latest") {
		t.Errorf("unverified image was altered: %q", got)
	}
}

func TestCloneVerificationFails(t *testing.T) {
	for name, tc := range map[string]struct {
		image string
		err   error
	}{
		"unsigned":    {"ghcr.io/cloudengio/ci:latest", errors.Join(ociimage.ErrNotVerified, ociimage.ErrUnsigned)},
		"local image": {"ci", nil},
	} {
		bin, calls := fakeDocker(t)
		inst := dockervm.New(t.Context(), tc.image, "c",
			dockervm.WithDockerBinary(bin), dockervm.WithVerification(stubVerifier{err: tc.err}, verifyPolicy))
		if err := inst.Clone(t.Context()); !errors.Is(err, ociimage.ErrNotVerified) {
			t.Errorf("%v: got %v, want ErrNotVerified", name, err)
		}
		if got := calls(); got != "" {
			t.Errorf("%v: docker was run: %q", name, got)
		}
		if got := inst.State(t.Context()); got != vms.StateInitial {
			t.Errorf("%v: state %v", name, got)
		}
	}
}
