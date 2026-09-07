// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// TestRetryableHandshakeErrors classifies the errors that the ssh handshake
// itself produces. They are obtained by performing handshakes that fail in
// each way rather than by constructing them, since x/crypto/ssh reports most
// of them without a type to match on and the wrapping it applies is what
// determines whether they can be recognised at all.
func TestRetryableHandshakeErrors(t *testing.T) {
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	_, otherSigner := newKey(t)
	serveAgent(t, userPriv)
	_, unknownUser := newKey(t)

	for _, tc := range []struct {
		name   string
		client *Client
		// server presents this host key, and requires this user key.
		hostKey, userKey ssh.Signer
	}{
		{"pinned host key mismatch",
			NewClient(t.Context(), "tcp", testAddr, WithUser("auser"), WithHostKey(otherSigner.PublicKey())),
			hostSigner, userSigner},
		{"known hosts mismatch",
			NewClient(t.Context(), "tcp", testAddr, WithUser("auser"), WithKnownHosts(knownHostsFile(t, otherSigner.PublicKey()))),
			hostSigner, userSigner},
		{"unknown host",
			NewClient(t.Context(), "tcp", testAddr, WithUser("auser"), WithKnownHosts(knownHostsFile(t))),
			hostSigner, userSigner},
		{"authentication failure",
			NewClient(t.Context(), "tcp", testAddr, WithUser("auser"), WithHostKey(hostSigner.PublicKey())),
			hostSigner, unknownUser},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := handshake(t, tc.client, tc.hostKey, tc.userKey)
			if err == nil {
				t.Fatal("expected the handshake to fail")
			}
			if Retryable(err) {
				t.Errorf("%v: reported as retryable, want permanent", err)
			}
		})
	}
}

// TestRetryableConfigErrors classifies the errors this package raises before
// any connection is attempted.
func TestRetryableConfigErrors(t *testing.T) {
	_, hostSigner := newKey(t)

	t.Run("no agent", func(t *testing.T) {
		t.Setenv("SSH_AUTH_SOCK", "")
		c := NewClient(t.Context(), "tcp", testAddr, WithHostKey(hostSigner.PublicKey()))
		_, _, err := c.sshConfigForAgent(c.opts.user)
		if !errors.Is(err, ErrNoAgent) {
			t.Fatalf("got %v, want ErrNoAgent", err)
		}
		if Retryable(err) {
			t.Errorf("%v: reported as retryable, want permanent", err)
		}
	})

	t.Run("no known hosts", func(t *testing.T) {
		c := NewClient(t.Context(), "tcp", testAddr, WithKnownHosts(filepath.Join(t.TempDir(), "known_hosts")))
		_, err := c.hostKeyCallback()
		if !errors.Is(err, ErrKnownHosts) {
			t.Fatalf("got %v, want ErrKnownHosts", err)
		}
		if Retryable(err) {
			t.Errorf("%v: reported as retryable, want permanent", err)
		}
	})

	t.Run("unparseable known hosts", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "known_hosts")
		if err := os.WriteFile(path, []byte("this is not a known hosts entry\n"), 0600); err != nil {
			t.Fatal(err)
		}
		c := NewClient(t.Context(), "tcp", testAddr, WithKnownHosts(path))
		_, err := c.hostKeyCallback()
		if !errors.Is(err, ErrKnownHosts) {
			t.Fatalf("got %v, want ErrKnownHosts", err)
		}
		if Retryable(err) {
			t.Errorf("%v: reported as retryable, want permanent", err)
		}
	})
}

// TestRetryableNetworkErrors classifies errors obtained from the network
// stack, again by provoking them rather than constructing them, so that the
// wrapping net applies is the wrapping being classified.
func TestRetryableNetworkErrors(t *testing.T) {
	// A port that was listening and is not any more is refused, which is what
	// a server that is down or restarting looks like.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closed := l.Addr().String()
	l.Close()
	if _, err := net.Dial("tcp", closed); err == nil {
		t.Skip("the address was reused before it could be dialled")
	} else if !Retryable(err) {
		t.Errorf("connection refused (%v): reported as permanent, want retryable", err)
	}

	// A port already bound cannot be bound again, which is what a local
	// forward finds when its port is in use.
	held, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close()
	if _, err := net.Listen("tcp", held.Addr().String()); err == nil {
		t.Error("expected the second listen to fail")
	} else if Retryable(err) {
		t.Errorf("address in use (%v): reported as retryable, want permanent", err)
	}

	// An unusable network is a configuration error.
	if _, err := net.Dial("nonsense", "127.0.0.1:1"); err == nil {
		t.Error("expected an unknown network to fail")
	} else if Retryable(err) {
		t.Errorf("unknown network (%v): reported as retryable, want permanent", err)
	}

	// As is an address that is not one.
	if _, err := net.Dial("tcp", "no-port-here"); err == nil {
		t.Error("expected a malformed address to fail")
	} else if Retryable(err) {
		t.Errorf("malformed address (%v): reported as retryable, want permanent", err)
	}
}

// TestRetryableClassification covers the remaining cases, which are either
// impractical to provoke or are sentinel values.
func TestRetryableClassification(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"cancelled", context.Canceled, false},
		{"cancelled, wrapped", fmt.Errorf("dialling: %w", context.Canceled), false},
		{"client closed", ErrClientClosed, false},
		// Revocation applies to host certificates, which knownhosts checks
		// via ssh.CertChecker, so this is not reachable through the plain
		// host key handshakes above.
		{"revoked host key", &knownhosts.RevokedError{}, false},
		{"revoked host key, wrapped", fmt.Errorf("ssh: handshake failed: %w", &knownhosts.RevokedError{}), false},
		{"host that does not exist", &net.DNSError{Err: "no such host", IsNotFound: true}, false},
		{"dns server did not answer", &net.DNSError{Err: "server misbehaving", IsTemporary: true, IsNotFound: true}, true},
		{"dns lookup timed out", &net.DNSError{Err: "i/o timeout", IsTimeout: true}, true},
		{"permission denied", &net.OpError{Op: "dial", Err: syscall.EACCES}, false},
		{"address family unsupported", &net.OpError{Op: "dial", Err: syscall.EAFNOSUPPORT}, false},

		{"deadline exceeded", context.DeadlineExceeded, true},
		{"connection reset", &net.OpError{Op: "read", Err: syscall.ECONNRESET}, true},
		{"host unreachable", &net.OpError{Op: "dial", Err: syscall.EHOSTUNREACH}, true},
		{"network unreachable", &net.OpError{Op: "dial", Err: syscall.ENETUNREACH}, true},
		{"connection timed out", &net.OpError{Op: "dial", Err: syscall.ETIMEDOUT}, true},
		{"too many open files", &net.OpError{Op: "socket", Err: syscall.EMFILE}, true},
		{"eof during the handshake", fmt.Errorf("ssh: handshake failed: %w", io.EOF), true},
		{"unexpected eof", fmt.Errorf("ssh: handshake failed: %w", io.ErrUnexpectedEOF), true},
		{"agent momentarily unreachable", fmt.Errorf("connecting to ssh agent at /tmp/s: %w", &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}), true},
		{"unrecognised", errors.New("something unanticipated"), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Retryable(tc.err); got != tc.want {
				t.Errorf("Retryable(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
