// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

func newKey(t *testing.T) (ed25519.PrivateKey, ssh.Signer) {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	return priv, signer
}

// serveAgent runs an in-process ssh agent holding key on a unix socket and
// points SSH_AUTH_SOCK at it, so that sshConfigForAgent finds it the same way
// it would find the user's own agent.
func serveAgent(t *testing.T, key ed25519.PrivateKey) {
	t.Helper()
	keyring := agent.NewKeyring()
	if err := keyring.Add(agent.AddedKey{PrivateKey: key}); err != nil {
		t.Fatal(err)
	}
	// t.TempDir() embeds the test name, which can exceed the length a unix
	// socket address allows.
	dir, err := os.MkdirTemp("", "pssh")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })

	l, err := net.Listen("unix", filepath.Join(dir, "s"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return // the listener was closed by the cleanup above.
			}
			go agent.ServeAgent(keyring, conn) //nolint:errcheck // io.EOF when the client disconnects.
		}
	}()
	t.Setenv("SSH_AUTH_SOCK", filepath.Join(dir, "s"))
}

// connPair returns the two ends of a connected loopback socket. net.Pipe is
// unsuitable here: it is unbuffered and synchronous, so the ssh version
// exchange deadlocks with both ends writing before either reads.
func connPair(t *testing.T) (client, server net.Conn) {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	type accepted struct {
		conn net.Conn
		err  error
	}
	ch := make(chan accepted, 1)
	go func() {
		conn, err := l.Accept()
		ch <- accepted{conn, err}
	}()
	client, err = net.Dial("tcp", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	a := <-ch
	if a.err != nil {
		t.Fatal(a.err)
	}
	t.Cleanup(func() {
		client.Close()
		a.conn.Close()
	})
	return client, a.conn
}

// serveSSH accepts a single ssh connection on conn, requiring that it
// authenticates as user with the public key of userKey. It reports what the
// server saw on the returned channel.
func serveSSH(conn net.Conn, hostKey, userKey ssh.Signer, user string) <-chan error {
	ch := make(chan error, 1)
	// Public key authentication is the only method configured, so reaching
	// this callback is itself proof that a key was offered.
	cfg := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if c.User() != user {
				return nil, fmt.Errorf("got user %q, want %q", c.User(), user)
			}
			if !bytes.Equal(key.Marshal(), userKey.PublicKey().Marshal()) {
				return nil, fmt.Errorf("unknown public key of type %v", key.Type())
			}
			return &ssh.Permissions{}, nil
		},
	}
	cfg.AddHostKey(hostKey)
	go func() {
		defer close(ch)
		_, chans, reqs, err := ssh.NewServerConn(conn, cfg)
		if err != nil {
			ch <- fmt.Errorf("server handshake: %w", err)
			return
		}
		// The server connection is deliberately left open: reporting a
		// successful handshake is not the end of it, and closing it here
		// would stop any channel from being opened on it. connPair closes
		// the sockets underneath it when the test ends.
		go ssh.DiscardRequests(reqs)
		go serveChannels(chans)
		ch <- nil
	}()
	return ch
}

const testAddr = "host.example.com:22"

// handshake runs a complete ssh handshake between c and an in-process server
// presenting hostSigner, returning the first error from either side.
func handshake(t *testing.T, c *Client, hostSigner, userSigner ssh.Signer) error {
	t.Helper()
	config, closeAgent, err := c.sshConfigForAgent(c.opts.user)
	if err != nil {
		return err
	}
	defer closeAgent()

	clientConn, serverConn := connPair(t)
	serverCh := serveSSH(serverConn, hostSigner, userSigner, c.opts.user)

	cconn, _, _, err := ssh.NewClientConn(clientConn, testAddr, config)
	if err != nil {
		return err
	}
	defer cconn.Close()
	select {
	case err := <-serverCh:
		return err
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for the server side of the handshake")
		return nil
	}
}

// knownHostsFile writes a known hosts file naming testAddr for each key.
func knownHostsFile(t *testing.T, keys ...ssh.PublicKey) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "known_hosts")
	var buf bytes.Buffer
	for _, key := range keys {
		buf.WriteString(knownhosts.Line([]string{knownhosts.Normalize(testAddr)}, key) + "\n")
	}
	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

// directTCPIP is the payload of a direct-tcpip channel request, which is what
// a local port forward opens for each connection it accepts.
type directTCPIP struct {
	DestAddr string
	DestPort uint32
	OrigAddr string
	OrigPort uint32
}

// serveChannels implements the server side of local port forwarding: each
// direct-tcpip channel is connected to the address it names, so that a
// forward can be exercised end to end.
func serveChannels(chans <-chan ssh.NewChannel) {
	for nc := range chans {
		if nc.ChannelType() != "direct-tcpip" {
			nc.Reject(ssh.UnknownChannelType, "only direct-tcpip is served") //nolint:errcheck
			continue
		}
		var req directTCPIP
		if err := ssh.Unmarshal(nc.ExtraData(), &req); err != nil {
			nc.Reject(ssh.ConnectionFailed, err.Error()) //nolint:errcheck
			continue
		}
		target, err := net.Dial("tcp", fmt.Sprintf("%v:%v", req.DestAddr, req.DestPort))
		if err != nil {
			nc.Reject(ssh.ConnectionFailed, err.Error()) //nolint:errcheck
			continue
		}
		ch, reqs, err := nc.Accept()
		if err != nil {
			target.Close()
			continue
		}
		go ssh.DiscardRequests(reqs)
		go func() {
			defer target.Close()
			defer ch.Close()
			go io.Copy(target, ch) //nolint:errcheck
			io.Copy(ch, target)    //nolint:errcheck
		}()
	}
}

// dialSSH performs a handshake against an in-process server and returns the
// resulting client.
func dialSSH(t *testing.T, c *Client, hostSigner, userSigner ssh.Signer) *ssh.Client {
	t.Helper()
	config, closeAgent, err := c.sshConfigForAgent(c.opts.user)
	if err != nil {
		t.Fatal(err)
	}
	defer closeAgent()
	clientConn, serverConn := connPair(t)
	serveSSH(serverConn, hostSigner, userSigner, c.opts.user)
	sshConn, chans, reqs, err := ssh.NewClientConn(clientConn, testAddr, config)
	if err != nil {
		t.Fatal(err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	t.Cleanup(func() { client.Close() })
	return client
}

// TestSSHConfigForAgentHandshake verifies that the configuration authenticates
// using a key held only by the agent. The private key is never given to the
// client, so a successful handshake is only possible if the agent was
// consulted for the signature.
func TestSSHConfigForAgentHandshake(t *testing.T) {
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)

	c := NewClient(t.Context(), "tcp", testAddr, WithUser("auser"), WithHostKey(hostSigner.PublicKey()))
	config, closeAgent, err := c.sshConfigForAgent(c.opts.user)
	if err != nil {
		t.Fatal(err)
	}
	closeAgent()
	if got, want := config.User, "auser"; got != want {
		t.Errorf("User: got %v, want %v", got, want)
	}
	if err := handshake(t, c, hostSigner, userSigner); err != nil {
		t.Fatal(err)
	}
}

// TestHostKeyPinned verifies that WithHostKey holds the server to that key.
func TestHostKeyPinned(t *testing.T) {
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	_, otherSigner := newKey(t)
	serveAgent(t, userPriv)

	c := NewClient(t.Context(), "tcp", testAddr, WithUser("auser"), WithHostKey(otherSigner.PublicKey()))
	err := handshake(t, c, hostSigner, userSigner)
	if err == nil {
		t.Fatal("expected the handshake to fail on the host key")
	}
	if got, want := err.Error(), "host key mismatch"; !strings.Contains(got, want) {
		t.Errorf("error %q does not mention %q", got, want)
	}
}

// TestKnownHosts verifies verification against a known hosts file, in the
// three cases that ssh(1) distinguishes: the host is listed with this key, it
// is listed with a different key, and it is not listed at all.
func TestKnownHosts(t *testing.T) {
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	_, otherSigner := newKey(t)
	serveAgent(t, userPriv)

	for _, tc := range []struct {
		name string
		file string
		want string // empty for a handshake that must succeed.
	}{
		{"listed with this key", knownHostsFile(t, hostSigner.PublicKey()), ""},
		{"listed with another key", knownHostsFile(t, otherSigner.PublicKey()), "key mismatch"},
		{"not listed", knownHostsFile(t), "key is unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient(t.Context(), "tcp", testAddr, WithUser("auser"), WithKnownHosts(tc.file))
			err := handshake(t, c, hostSigner, userSigner)
			if tc.want == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected the handshake to fail")
			}
			if got := err.Error(); !strings.Contains(got, tc.want) {
				t.Errorf("error %q does not contain %q", got, tc.want)
			}
		})
	}
}

// TestKnownHostsAcceptNew verifies that an unknown host is recorded when new
// host keys are accepted, and is then verified against on the next connection
// without that option.
func TestKnownHostsAcceptNew(t *testing.T) {
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)
	file := knownHostsFile(t)

	c := NewClient(t.Context(), "tcp", testAddr, WithUser("auser"), WithKnownHosts(file), WithAcceptNewHostKeys(true))
	if err := handshake(t, c, hostSigner, userSigner); err != nil {
		t.Fatal(err)
	}

	recorded, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(recorded), knownhosts.Line([]string{knownhosts.Normalize(testAddr)}, hostSigner.PublicKey())+"\n"; got != want {
		t.Errorf("recorded %q, want %q", got, want)
	}

	// The recorded key must now satisfy a client that does not accept new
	// hosts, which is what recording it was for.
	strict := NewClient(t.Context(), "tcp", testAddr, WithUser("auser"), WithKnownHosts(file))
	if err := handshake(t, strict, hostSigner, userSigner); err != nil {
		t.Fatal(err)
	}
}

// TestKnownHostsAcceptNewRefusesChangedKey verifies that accepting new hosts
// does not extend to a host whose key has changed, which is the case the
// option exists to keep refusing.
func TestKnownHostsAcceptNewRefusesChangedKey(t *testing.T) {
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	_, otherSigner := newKey(t)
	serveAgent(t, userPriv)
	file := knownHostsFile(t, otherSigner.PublicKey())

	c := NewClient(t.Context(), "tcp", testAddr, WithUser("auser"), WithKnownHosts(file), WithAcceptNewHostKeys(true))
	err := handshake(t, c, hostSigner, userSigner)
	if err == nil {
		t.Fatal("expected the handshake to fail on the changed host key")
	}
	if got, want := err.Error(), "key mismatch"; !strings.Contains(got, want) {
		t.Errorf("error %q does not contain %q", got, want)
	}

	// The file must be left as it was: a changed key is never recorded.
	recorded, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(recorded), knownhosts.Line([]string{knownhosts.Normalize(testAddr)}, otherSigner.PublicKey())+"\n"; got != want {
		t.Errorf("recorded %q, want %q", got, want)
	}
}

// TestKnownHostsMissingFiles verifies the handling of known hosts files that
// do not exist, which ssh(1) ignores rather than treating as an error.
func TestKnownHostsMissingFiles(t *testing.T) {
	userPriv, userSigner := newKey(t)
	_, hostSigner := newKey(t)
	serveAgent(t, userPriv)
	missing := filepath.Join(t.TempDir(), "nested", "known_hosts")

	// Without any file to verify against, and without accepting new hosts,
	// there is nothing that could authorise the connection.
	c := NewClient(t.Context(), "tcp", testAddr, WithUser("auser"), WithKnownHosts(missing))
	err := handshake(t, c, hostSigner, userSigner)
	if err == nil {
		t.Fatal("expected an error when no known hosts file exists")
	}
	if got, want := err.Error(), "new host keys are not accepted"; !strings.Contains(got, want) {
		t.Errorf("error %q does not contain %q", got, want)
	}

	// Accepting new hosts creates the file, and the directory holding it.
	c = NewClient(t.Context(), "tcp", testAddr, WithUser("auser"), WithKnownHosts(missing), WithAcceptNewHostKeys(true))
	if err := handshake(t, c, hostSigner, userSigner); err != nil {
		t.Fatal(err)
	}
	recorded, err := os.ReadFile(missing)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(recorded), knownhosts.Line([]string{knownhosts.Normalize(testAddr)}, hostSigner.PublicKey())+"\n"; got != want {
		t.Errorf("recorded %q, want %q", got, want)
	}
}

// TestAppendKnownHostNewline verifies that an entry appended to a file which
// does not end in a newline is not merged with the existing last entry.
func TestAppendKnownHostNewline(t *testing.T) {
	_, first := newKey(t)
	_, second := newKey(t)
	path := filepath.Join(t.TempDir(), "known_hosts")
	existing := knownhosts.Line([]string{"other.example.com"}, first.PublicKey())
	if err := os.WriteFile(path, []byte(existing), 0600); err != nil {
		t.Fatal(err)
	}
	if err := appendKnownHost(path, testAddr, second.PublicKey()); err != nil {
		t.Fatal(err)
	}
	added := knownhosts.Line([]string{knownhosts.Normalize(testAddr)}, second.PublicKey())
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := existing + "\n" + added + "\n"; string(got) != want {
		t.Errorf("got %q, want %q", got, want)
	}
	// The result must be parseable, which it would not be had the entries run
	// together.
	if _, err := knownhosts.New(path); err != nil {
		t.Errorf("the resulting file does not parse: %v", err)
	}
}

func TestSSHConfigForAgentErrors(t *testing.T) {
	_, hostSigner := newKey(t)
	hostKey := hostSigner.PublicKey()
	for _, tc := range []struct {
		name string
		opts []Option
		sock string
		want string
	}{
		{"no agent", []Option{WithHostKey(hostKey)}, "", "SSH_AUTH_SOCK is not set"},
		{"unreachable agent", []Option{WithHostKey(hostKey)}, "/nonexistent/agent.sock", "connecting to ssh agent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SSH_AUTH_SOCK", tc.sock)
			c := NewClient(t.Context(), "tcp", testAddr, tc.opts...)
			config, closeAgent, err := c.sshConfigForAgent(c.opts.user)
			if err == nil {
				closeAgent()
				t.Fatalf("expected an error, got a config for user %q", config.User)
			}
			if got := err.Error(); !strings.Contains(got, tc.want) {
				t.Errorf("error %q does not contain %q", got, tc.want)
			}
			if config != nil || closeAgent != nil {
				t.Error("expected nil results alongside the error")
			}
		})
	}
}

// TestHostKeyCallbackRefusesSystemFile verifies that accepting new host keys
// refuses to record into system-wide /etc/ssh/ssh_known_hosts.
func TestHostKeyCallbackRefusesSystemFile(t *testing.T) {
	c := NewClient(t.Context(), "tcp", testAddr,
		WithKnownHosts("/etc/ssh/ssh_known_hosts"),
		WithAcceptNewHostKeys(true))
	_, err := c.hostKeyCallback()
	if err == nil {
		t.Fatal("expected an error when attempting to accept new host keys to system file")
	}
	if !errors.Is(err, ErrKnownHosts) {
		t.Errorf("got %v, want ErrKnownHosts", err)
	}
}

// TestAppendKnownHostConcurrent verifies that concurrent appends do not
// corrupt the known hosts file.
func TestAppendKnownHostConcurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "known_hosts")
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		_, key := newKey(t)
		host := fmt.Sprintf("host%d.example.com", i)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := appendKnownHost(path, host, key.PublicKey()); err != nil {
				t.Errorf("append failed: %v", err)
			}
		}()
	}
	wg.Wait()

	// The resulting file must be parseable.
	if _, err := knownhosts.New(path); err != nil {
		t.Errorf("resulting file corrupted by concurrent writes: %v", err)
	}
}
