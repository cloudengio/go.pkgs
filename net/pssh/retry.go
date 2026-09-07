// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"context"
	"errors"
	"net"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Errors that a retry cannot resolve, since they describe how the client is
// configured rather than the state of the network or the server.
var (
	// ErrNoAgent is returned when no ssh agent is available at all.
	ErrNoAgent = errors.New("no ssh agent available")
	// ErrKnownHosts is returned when the known hosts files cannot be used,
	// either because none of them exist or because they cannot be parsed.
	ErrKnownHosts = errors.New("no usable known hosts file")
	// ErrClientClosed is returned when the client has been closed.
	ErrClientClosed = errors.New("client closed")
)

// Retryable reports whether a connection attempt that failed with err may
// succeed if it is attempted again, as opposed to one that will fail the same
// way however often it is repeated.
//
// Errors are treated as retryable unless they are known not to be. A
// persistent client exists to stay connected, so an unrecognised failure
// should not end it; the backoff bounds how long an error that is in fact
// permanent is retried for. The cases below are therefore the exceptions.
//
// Not retryable:
//
//   - Anything to do with the identity of the server. A key that does not
//     match the one recorded may be an interception attempt, and reconnecting
//     to a host that presents it is exactly what must not happen. An unknown
//     host stays unknown, and a revoked key stays revoked.
//   - Authentication failure. A retry offers the server the same keys, and
//     repeating it counts against MaxAuthTries and any lockout policy.
//   - This package's own configuration errors: ErrNoAgent, ErrKnownHosts.
//   - Algorithm negotiation failure. The two ends do not have a cipher, MAC,
//     key exchange or host key type in common, and will not acquire one.
//   - Malformed networks and addresses, which are in the configuration.
//   - A hostname that does not exist. Note that this is distinguished from a
//     DNS server that could not be reached, which is retryable.
//   - A local forward whose port is already bound, which needs the port to be
//     freed or the configuration changed.
//   - Being refused permission by the local system.
//
// Cancellation is not retryable either, but it is not a failed attempt: err is
// context.Canceled or ErrClientClosed because the caller asked to stop. The
// caller should test its own context rather than rely on this, and must do so
// to distinguish a context whose deadline has expired -- reported here as
// retryable, since a dial timeout is otherwise indistinguishable from it --
// from one that is still live.
func Retryable(err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, ErrClientClosed):
		// The caller asked to stop rather than the attempt having failed.
		return false
	case isHostKeyError(err):
		return false
	case errors.Is(err, ErrNoAgent), errors.Is(err, ErrKnownHosts):
		return false
	case isAuthError(err):
		return false
	case isNegotiationError(err):
		return false
	case isAddrError(err):
		return false
	case isDNSNotFound(err):
		return false
	case errors.Is(err, syscall.EADDRINUSE):
		return false
	case errors.Is(err, syscall.EACCES), errors.Is(err, syscall.EPERM):
		return false
	}
	// Everything else, notably: connections refused, reset or timed out, a
	// host or network that is unreachable, an EOF part way through the
	// handshake, an agent that is momentarily unreachable, a DNS server that
	// did not answer, and exhaustion of file descriptors or ports.
	return true
}

// isHostKeyError reports whether err is a failure to verify the server's host
// key, from either a known hosts file or a pinned key.
func isHostKeyError(err error) bool {
	var keyErr *knownhosts.KeyError
	var revoked *knownhosts.RevokedError
	if errors.As(err, &keyErr) || errors.As(err, &revoked) {
		return true
	}
	// ssh.FixedHostKey reports a mismatch as an error carrying nothing but
	// its text, so there is nothing else to match on. The knownhosts texts
	// are matched too, covering a callback error that some intermediate
	// wrapping flattened; RevokedError is deliberately not among them, so
	// that it is matched by type above.
	msg := err.Error()
	return strings.Contains(msg, "host key mismatch") ||
		strings.Contains(msg, "knownhosts: key is unknown") ||
		strings.Contains(msg, "knownhosts: key mismatch")
}

// isAuthError reports whether err is a failure to authenticate to the server.
// x/crypto/ssh reports the exhaustion of the client's authentication methods
// with an unstructured error, so its text is all there is to match on. The
// types it does export for authentication, ServerAuthError and
// PartialSuccessError, are produced by the server half of the package and are
// never returned to a client.
func isAuthError(err error) bool {
	return strings.Contains(err.Error(), "unable to authenticate")
}

// isNegotiationError reports whether the two ends failed to agree on an
// algorithm.
func isNegotiationError(err error) bool {
	var negotiation *ssh.AlgorithmNegotiationError
	if errors.As(err, &negotiation) {
		return true
	}
	return strings.Contains(err.Error(), "ssh: no common algorithm")
}

// isAddrError reports whether err describes an address or network that cannot
// be used as given.
func isAddrError(err error) bool {
	var addrErr *net.AddrError
	var unknownNet net.UnknownNetworkError
	var invalidAddr net.InvalidAddrError
	if errors.As(err, &addrErr) || errors.As(err, &unknownNet) || errors.As(err, &invalidAddr) {
		return true
	}
	return errors.Is(err, syscall.EAFNOSUPPORT) || errors.Is(err, syscall.EPROTONOSUPPORT)
}

// isDNSNotFound reports whether err is a name that does not exist, as opposed
// to a name that could not be looked up.
func isDNSNotFound(err error) bool {
	var dnsErr *net.DNSError
	if !errors.As(err, &dnsErr) {
		return false
	}
	// A temporary failure or a timeout means the answer is unknown rather
	// than negative, whatever IsNotFound may say.
	if dnsErr.IsTemporary || dnsErr.IsTimeout {
		return false
	}
	return dnsErr.IsNotFound
}
