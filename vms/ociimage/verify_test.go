// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ociimage_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"cloudeng.io/vms/ociimage"
	"gopkg.in/yaml.v3"
)

const sum = "sha256:0123456789abcdef"

var keyless = ociimage.Signer{Issuer: "https://issuer", SubjectRegexp: "^https://github.com/cloudengio/.*"}

type fakeVerifier struct {
	res   ociimage.Verified
	err   error
	calls int
}

func (f *fakeVerifier) Verify(_ context.Context, _ ociimage.Reference, _ ociimage.Policy) (ociimage.Verified, error) {
	f.calls++
	return f.res, f.err
}

type verifyPuller struct{ pulled []ociimage.Reference }

func (r *verifyPuller) Pull(_ context.Context, ref ociimage.Reference, _ ...ociimage.PullOption) error {
	r.pulled = append(r.pulled, ref)
	return nil
}

func TestPolicyValidate(t *testing.T) {
	for _, tc := range []struct {
		name string
		p    ociimage.Policy
		want string // empty if valid
	}{
		{"key", ociimage.Policy{Signers: []ociimage.Signer{{Key: "cosign.pub"}}}, ""},
		{"identity", ociimage.Policy{Signers: []ociimage.Signer{keyless}}, ""},
		{"no signers", ociimage.Policy{}, "at least one signer"},
		{"empty signer", ociimage.Policy{Signers: []ociimage.Signer{{}}}, "signers[0]"},
		{"key and identity", ociimage.Policy{Signers: []ociimage.Signer{{Key: "k", Issuer: "i", Subject: "s"}}}, "not both"},
		{"issuer only", ociimage.Policy{Signers: []ociimage.Signer{{Issuer: "i"}}}, "subject"},
		{"subject only", ociimage.Policy{Signers: []ociimage.Signer{{Subject: "s"}}}, "issuer"},
		{"exact and regexp", ociimage.Policy{Signers: []ociimage.Signer{{Issuer: "i", IssuerRegexp: ".*", Subject: "s"}}}, "exactly one"},
		{"bad regexp", ociimage.Policy{Signers: []ociimage.Signer{{Issuer: "i", SubjectRegexp: "("}}}, "subject_regexp"},
	} {
		err := tc.p.Validate()
		switch {
		case tc.want == "" && err != nil:
			t.Errorf("%v: unexpected error: %v", tc.name, err)
		case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
			t.Errorf("%v: got %v, want an error containing %q", tc.name, err, tc.want)
		}
	}
}

func TestVerify(t *testing.T) {
	ctx := t.Context()
	policy := ociimage.Policy{Signers: []ociimage.Signer{keyless}}
	ref := mustParse(t, "ghcr.io/cloudengio/ci:latest")
	good := ociimage.Verified{
		Reference:  ref.WithDigest(sum),
		Signatures: []ociimage.Signature{{Issuer: "https://issuer", Subject: "s"}},
	}

	res, err := ociimage.Verify(ctx, &fakeVerifier{res: good}, ref, policy)
	if err != nil || res.Reference.Digest != sum {
		t.Fatalf("got %v, %v", res, err)
	}

	notVerified := errors.Join(ociimage.ErrNotVerified, ociimage.ErrUnsigned)
	if _, err := ociimage.Verify(ctx, &fakeVerifier{err: notVerified}, ref, policy); !errors.Is(err, ociimage.ErrUnsigned) {
		t.Errorf("the verifier's error was not passed on: %v", err)
	}

	// Everything the verifier answers is checked, since it is the one part
	// that cannot be trusted to have been written carefully.
	unpinned := good
	unpinned.Reference = ref
	other := good
	other.Reference = mustParse(t, "ghcr.io/cloudengio/other:latest").WithDigest(sum)
	otherRegistry := good
	otherRegistry.Reference = ref.WithRegistry("evil.io").WithDigest(sum)
	otherTag := good
	otherTag.Reference = ref.WithTag("evil").WithDigest(sum)
	unsigned := good
	unsigned.Signatures = nil
	for name, bad := range map[string]ociimage.Verified{
		"unpinned": unpinned, "other image": other, "other registry": otherRegistry, "other tag": otherTag, "no signatures": unsigned,
	} {
		if _, err := ociimage.Verify(ctx, &fakeVerifier{res: bad}, ref, policy); !errors.Is(err, ociimage.ErrNotVerified) {
			t.Errorf("%v: got %v, want ErrNotVerified", name, err)
		}
	}
	// A verifier may strip the tag when returning a digest-pinned reference.
	stripped := good
	stripped.Reference = good.Reference.WithTag("")
	if res, err := ociimage.Verify(ctx, &fakeVerifier{res: stripped}, ref, policy); err != nil || res.Reference.Digest != sum {
		t.Errorf("a digest-pinned reference without tag was rejected: %v, %v", res, err)
	}
	// A digest in the request must be the one that was verified.
	pinned := ref.WithDigest("sha256:ffff")
	if _, err := ociimage.Verify(ctx, &fakeVerifier{res: good}, pinned, policy); !errors.Is(err, ociimage.ErrNotVerified) {
		t.Errorf("a different digest was accepted: %v", err)
	}
}

// TestVerifyFailsClosed covers the requests that must never reach, or be
// satisfied by, a verifier.
func TestVerifyFailsClosed(t *testing.T) {
	ctx := t.Context()
	policy := ociimage.Policy{Signers: []ociimage.Signer{keyless}}
	fv := &fakeVerifier{}
	if _, err := ociimage.Verify(ctx, fv, mustParse(t, "linux-ci"), policy); !errors.Is(err, ociimage.ErrNotVerified) {
		t.Errorf("a local image was not rejected: %v", err)
	}
	if _, err := ociimage.Verify(ctx, fv, mustParse(t, "ghcr.io/a/b"), ociimage.Policy{}); err == nil {
		t.Error("an empty policy was accepted")
	}
	if _, err := ociimage.Verify(ctx, nil, mustParse(t, "ghcr.io/a/b"), policy); !errors.Is(err, ociimage.ErrNotVerified) {
		t.Errorf("no verifier: %v", err)
	}
	if fv.calls != 0 {
		t.Errorf("the verifier was called %v times", fv.calls)
	}
}

func TestPullVerified(t *testing.T) {
	ctx := t.Context()
	policy := ociimage.Policy{Signers: []ociimage.Signer{keyless}}
	ref := mustParse(t, "ghcr.io/cloudengio/ci:latest")
	good := ociimage.Verified{Reference: ref.WithDigest(sum), Signatures: []ociimage.Signature{{}}}

	// The tag is dropped, so that the digest that was verified is all that
	// the reference names.
	pinned := good.Reference.WithTag("")
	p := &verifyPuller{}
	got, err := ociimage.PullVerified(ctx, p, &fakeVerifier{res: good}, ref, policy)
	if err != nil || got != pinned {
		t.Fatalf("got %v, %v, want %v", got, err, pinned)
	}
	// What is pulled is the digest that was verified, not the moving tag.
	if len(p.pulled) != 1 || p.pulled[0] != pinned {
		t.Errorf("pulled %v, want %v", p.pulled, pinned)
	}

	p = &verifyPuller{}
	if _, err := ociimage.PullVerified(ctx, p, &fakeVerifier{err: ociimage.ErrNotVerified}, ref, policy); err == nil || len(p.pulled) != 0 {
		t.Errorf("an image that failed verification was pulled: %v, %v", err, p.pulled)
	}
}

func TestSourceVerify(t *testing.T) {
	var src ociimage.Source
	cfg := `
image: ghcr.io/cloudengio/ci:latest
verify:
  signers:
    - issuer: https://token.actions.githubusercontent.com
      subject_regexp: ^https://github.com/cloudengio/.*
  require_transparency_log: true
`
	if err := yaml.Unmarshal([]byte(cfg), &src); err != nil {
		t.Fatal(err)
	}
	if err := src.Validate(); err != nil {
		t.Fatal(err)
	}
	if src.Verify == nil || !src.Verify.TransparencyLog || len(src.Verify.Signers) != 1 {
		t.Errorf("unexpected policy: %+v", src.Verify)
	}
	for name, s := range map[string]ociimage.Source{
		"empty policy": {Image: "ghcr.io/a/b", Verify: &ociimage.Policy{}},
		"local image":  {Image: "linux-ci", Verify: &ociimage.Policy{Signers: []ociimage.Signer{keyless}}},
	} {
		if err := s.Validate(); err == nil || !strings.Contains(err.Error(), "verify") {
			t.Errorf("%v: got %v", name, err)
		}
	}
}

func TestPin(t *testing.T) {
	ctx := t.Context()
	policy := ociimage.Policy{Signers: []ociimage.Signer{keyless}}
	ref := mustParse(t, "ghcr.io/cloudengio/ci:latest")
	good := ociimage.Verified{Reference: ref.WithDigest(sum), Signatures: []ociimage.Signature{{}}}

	// Without a policy nothing is verified, or even parsed.
	fv := &fakeVerifier{}
	if got, err := ociimage.Pin(ctx, fv, nil, "not a reference"); err != nil || got != "not a reference" || fv.calls != 0 {
		t.Errorf("got %q, %v", got, err)
	}
	got, err := ociimage.Pin(ctx, &fakeVerifier{res: good}, &policy, ref.String())
	if want := "ghcr.io/cloudengio/ci@" + sum; err != nil || got != want {
		t.Errorf("got %q, %v, want %q", got, err, want)
	}
	if _, err := ociimage.Pin(ctx, &fakeVerifier{err: ociimage.ErrUnsigned}, &policy, ref.String()); !errors.Is(err, ociimage.ErrUnsigned) {
		t.Errorf("got %v", err)
	}
	if _, err := ociimage.Pin(ctx, &fakeVerifier{res: good}, &policy, "ghcr.io//x"); !errors.Is(err, ociimage.ErrInvalidReference) {
		t.Errorf("got %v", err)
	}
}
