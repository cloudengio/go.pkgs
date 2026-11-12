// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dockerapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	sdkclient "github.com/docker/go-sdk/client"
	mobyclient "github.com/moby/moby/client"

	"cloudeng.io/vms"
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

type stubSDKClient struct {
	sdkclient.SDKClient
	created []mobyclient.ContainerCreateOptions
}

func (s *stubSDKClient) ContainerCreate(_ context.Context, opts mobyclient.ContainerCreateOptions) (mobyclient.ContainerCreateResult, error) {
	s.created = append(s.created, opts)
	return mobyclient.ContainerCreateResult{ID: "container-123"}, nil
}

func newVerifyTestInstance(cli sdkclient.SDKClient, image string, opts ...Option) *Instance {
	o := options{
		pollingInterval:  DefaultPollingInterval,
		forceStopTimeout: DefaultForceStopTimeout,
		containerCmd:     DefaultContainerCmd(),
	}
	for _, opt := range opts {
		opt(&o)
	}
	return &Instance{
		image:  image,
		name:   "c",
		state:  vms.StateInitial,
		opts:   o,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		cli:    cli,
	}
}

func TestCloneVerifies(t *testing.T) {
	cli := &stubSDKClient{}
	inst := newVerifyTestInstance(cli, "ghcr.io/cloudengio/ci:latest", WithVerification(stubVerifier{}, verifyPolicy))
	if err := inst.Clone(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(cli.created) != 1 || cli.created[0].Image != "ghcr.io/cloudengio/ci@"+verifiedDigest {
		t.Errorf("the verified digest was not used: %+v", cli.created)
	}

	cli = &stubSDKClient{}
	inst = newVerifyTestInstance(cli, "ghcr.io/cloudengio/ci:latest")
	if err := inst.Clone(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(cli.created) != 1 || cli.created[0].Image != "ghcr.io/cloudengio/ci:latest" {
		t.Errorf("unverified image was altered: %+v", cli.created)
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
		cli := &stubSDKClient{}
		inst := newVerifyTestInstance(cli, tc.image, WithVerification(stubVerifier{err: tc.err}, verifyPolicy))
		if err := inst.Clone(t.Context()); !errors.Is(err, ociimage.ErrNotVerified) {
			t.Errorf("%v: got %v, want ErrNotVerified", name, err)
		}
		if len(cli.created) != 0 {
			t.Errorf("%v: ContainerCreate was called: %+v", name, cli.created)
		}
		if got := inst.State(t.Context()); got != vms.StateInitial {
			t.Errorf("%v: state %v", name, got)
		}
	}
}
