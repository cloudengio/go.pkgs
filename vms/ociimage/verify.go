// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package ociimage

import (
	"context"
	"errors"
	"fmt"
	"regexp"
)

var (
	// ErrNotVerified is returned, or wrapped, by a Verifier for an image that
	// does not satisfy the policy, and by Verify for one that cannot be
	// verified at all.
	ErrNotVerified = errors.New("image signature not verified")
	// ErrUnsigned is returned, or wrapped, along with ErrNotVerified for an
	// image that has no signatures, as opposed to one whose signatures are
	// not acceptable.
	ErrUnsigned = errors.New("image is not signed")
)

// Signer is a signer that a Policy trusts: either a public key or, for
// keyless signing, the identity that was certified to sign. Exactly one of
// Key, or the Issuer and Subject, is set.
type Signer struct {
	// Key identifies a public key. How it is spelled, such as a file, a URL
	// or a KMS URI, is for the Verifier to say.
	Key string `yaml:"key" doc:"the public key that signed the image, as a file, URL or KMS reference understood by the verifier."`
	// Issuer and IssuerRegexp are the OIDC issuer of a keyless identity, as an exact
	// string or a regular expression.
	Issuer       string `yaml:"issuer" doc:"the OIDC issuer of the identity that signed the image."`
	IssuerRegexp string `yaml:"issuer_regexp" doc:"a regular expression that the OIDC issuer must match, instead of issuer."`
	// Subject and SubjectRegexp are the subject of a keyless identity, such as an
	// email address or a workflow URL.
	Subject       string `yaml:"subject" doc:"the subject of the identity that signed the image, such as an email address or workflow URL."`
	SubjectRegexp string `yaml:"subject_regexp" doc:"a regular expression that the subject must match, instead of subject."`
}

// Keyless reports whether s is an identity rather than a key.
func (s Signer) Keyless() bool {
	return s.Key == ""
}

// Validate reports a Signer that does not say who is trusted: both a key and
// an identity, neither, an identity that lacks its issuer or subject, or a
// regular expression that does not compile. Requiring both an issuer and a
// subject, and not merely one, ensures that a signer is never "anyone that
// the issuer certifies".
func (s Signer) Validate() error {
	if s.Key != "" {
		if s.Issuer != "" || s.IssuerRegexp != "" || s.Subject != "" || s.SubjectRegexp != "" {
			return errors.New("a signer is a key or an identity, not both")
		}
		return nil
	}
	for _, f := range []struct {
		name, exact, expr string
	}{
		{"issuer", s.Issuer, s.IssuerRegexp},
		{"subject", s.Subject, s.SubjectRegexp},
	} {
		if (f.exact == "") == (f.expr == "") {
			return fmt.Errorf("a signer needs a key, or an identity with exactly one of %[1]s and %[1]s_regexp", f.name)
		}
		if _, err := regexp.Compile(f.expr); err != nil {
			return fmt.Errorf("%s_regexp: %w", f.name, err)
		}
	}
	return nil
}

// Policy is what an image must satisfy to be accepted: it carries a
// valid signature from at least one of the Signers, and the signature meets
// the other requirements. It holds no more than a Verifier needs to decide, so
// that it can be read from a configuration file, see Source.
type Policy struct {
	Signers []Signer `yaml:"signers" doc:"the signers that are trusted: an image is accepted if it carries a valid signature from any one of them."`
	// TransparencyLog requires that the signature is recorded in a
	// transparency log, such as Rekor.
	TransparencyLog bool `yaml:"require_transparency_log" doc:"require that the signature is recorded in a transparency log."`
	// Annotations must all be present, with these values, in the signature.
	Annotations map[string]string `yaml:"annotations" doc:"annotations that the signature must carry, with these values."`
}

// Validate reports a policy that cannot accept anything meaningful, which is
// one that trusts no signer: a policy is never to be silently permissive.
func (p Policy) Validate() error {
	if len(p.Signers) == 0 {
		return errors.New("a verification policy needs at least one signer")
	}
	for i, s := range p.Signers {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("signers[%d]: %w", i, err)
		}
	}
	return nil
}

// Signature describes a signature that satisfied a policy.
type Signature struct {
	// Key is the key that verified the signature, empty for a keyless one.
	Key string
	// Issuer and Subject are the certified identity of a keyless signature.
	Issuer, Subject string
	// Annotations are those the signature carries.
	Annotations map[string]string
	// Logged is true if the signature is recorded in a transparency log.
	Logged bool
}

// Verified is the result of a successful verification.
type Verified struct {
	// Reference is the image that was verified, pinned to its digest. It is
	// what is to be pulled and used, rather than the reference that was
	// asked about: a tag can be moved between verifying it and using it, but
	// the digest that carries the signature cannot.
	Reference Reference
	// Signatures are those that satisfied the policy.
	Signatures []Signature
}

// Verifier verifies the signatures of images, in the form that cosign and
// others store them in the registry that holds the image. It is implemented
// by whatever depends on a signing toolkit, so that this package does not.
//
// An implementation resolves the reference to a digest if it does not
// have one, and verifies the signatures of that digest and no other. It
// returns an error that satisfies errors.Is(err, ErrNotVerified) if the image
// does not satisfy the policy, and may return any other error for a failure
// to carry out the verification, such as a registry that cannot be reached.
// Either way the image is not to be used.
type Verifier interface {
	Verify(ctx context.Context, ref Reference, policy Policy) (Verified, error)
}

// Verify verifies ref with v against policy, after checking everything about
// the request and the answer that does not depend on the toolkit: that there
// is a verifier and a valid policy, that the image is remote since a local one
// has no registry to hold a signature and so can never satisfy a policy, and
// that the verified reference is the one asked about, pinned to a digest.
func Verify(ctx context.Context, v Verifier, ref Reference, policy Policy) (Verified, error) {
	if v == nil {
		return Verified{}, fmt.Errorf("%w: cannot verify %v: no verifier", ErrNotVerified, ref)
	}
	if err := policy.Validate(); err != nil {
		return Verified{}, fmt.Errorf("cannot verify %v: %w", ref, err)
	}
	if !ref.Remote() {
		return Verified{}, fmt.Errorf("%w: %v names no registry, so it has no signature to verify", ErrNotVerified, ref)
	}
	res, err := v.Verify(ctx, ref, policy)
	if err != nil {
		return Verified{}, err
	}
	if res.Reference.Digest == "" {
		return Verified{}, fmt.Errorf("%w: %v: the verifier did not pin the image to a digest", ErrNotVerified, ref)
	}
	if ref.Digest != "" && ref.Digest != res.Reference.Digest ||
		ref.Path() != res.Reference.Path() ||
		ref.Registry != "" && ref.Registry != res.Reference.Registry {
		return Verified{}, fmt.Errorf("%w: %v: the verifier verified %v instead", ErrNotVerified, ref, res.Reference)
	}
	if len(res.Signatures) == 0 {
		return Verified{}, fmt.Errorf("%w: %v: the verifier reported no signature", ErrNotVerified, ref)
	}
	return res, nil
}

// PullVerified verifies ref and, only if it is accepted, pulls the verified
// reference, which is returned for the caller to create VMs or containers
// from, see Verified.Reference. Like Pull it needs a Puller, but not one that
// does nothing for a local reference: that is an error, see Verify.
func PullVerified(ctx context.Context, p Puller, v Verifier, ref Reference, policy Policy, opts PullOptions) (Reference, error) {
	res, err := Verify(ctx, v, ref, policy)
	if err != nil {
		return Reference{}, err
	}
	if err := Pull(ctx, p, res.Reference, opts); err != nil {
		return Reference{}, err
	}
	return res.Reference, nil
}

// Pin returns the reference to create a VM or container from when it is
// created from the image named source, for the backends that do so. If policy
// is nil, no verification was asked for and source is returned as it is.
// Otherwise the image is verified, see Verify, and the verified digest is
// returned as a reference to it alone, without its tag, so that what is
// created is exactly what was verified: an image that is unsigned, or not
// signed as the policy requires, is an error and not to be used.
func Pin(ctx context.Context, v Verifier, policy *Policy, source string) (string, error) {
	if policy == nil {
		return source, nil
	}
	ref, err := Parse(source)
	if err != nil {
		return "", err
	}
	res, err := Verify(ctx, v, ref, *policy)
	if err != nil {
		return "", err
	}
	return res.Reference.WithTag("").String(), nil
}
