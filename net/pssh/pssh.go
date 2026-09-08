// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package pssh provides a persistent ssh connection, namely, one that
// will be recreated if the connection is lost.
package pssh

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cloudeng.io/algo/ratecontrol"
	"cloudeng.io/logging/ctxlog"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Client represents a persistent ssh connection to a server. It is created
// with NewClient, which takes the network and address of the server, and
// optional configuration. The persistent connection is established with
// ConnectAndWait and terminated with Close.
type Client struct {
	network, addr string
	closeOnce     sync.Once
	doneCh        chan struct{}
	opts          options
}

type forward struct {
	localPort  int
	remotePort int
}

type options struct {
	dialTimeout   time.Duration
	user          string
	hostKey       ssh.PublicKey
	knownHosts    []string
	acceptNewHost bool
	localForwards []forward
	logger        *slog.Logger
}

type Option func(*options)

// WithLocalPortForward specifies a port forward from localPort to remotePort. It
// can be used multiple times to specify multiple forwards.
func WithLocalPortForward(localPort, remotePort int) Option {
	return func(o *options) {
		o.localForwards = append(o.localForwards, forward{localPort: localPort, remotePort: remotePort})
	}
}

const (
	// DefaultDialTimeout is the default timeout for establishing a connection to
	// the server. It is used if WithDialTimeout is not specified.
	DefaultDialTimeout = 30 * time.Second
)

// WithDialTimeout specifies the timeout for establishing a connection to the
// server. The default is DefaultDialTimeout.
func WithDialTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.dialTimeout = timeout
	}
}

// WithHostKey pins the public key that the server is required to present,
// bypassing the known hosts files entirely. It takes precedence over
// WithKnownHosts.
func WithHostKey(key ssh.PublicKey) Option {
	return func(o *options) {
		o.hostKey = key
	}
}

// WithKnownHosts specifies the known hosts files used to verify the server,
// in the format used by ssh(1). If no files are named, or this option is not
// used at all, ~/.ssh/known_hosts and /etc/ssh/ssh_known_hosts are used.
// Named files that do not exist are ignored, as ssh(1) ignores them.
func WithKnownHosts(files ...string) Option {
	return func(o *options) {
		o.knownHosts = files
		if len(files) == 0 {
			o.knownHosts = defaultKnownHostsFiles()
		}
	}
}

// WithAcceptNewHostKeys determines what happens when the server is not listed
// in any known hosts file. The default, and the behaviour when accept is
// false, is to refuse the connection: there is no user to prompt in the way
// that ssh(1) does, and a client that reconnects indefinitely must not accept
// a new key on every reconnection.
//
// When accept is true a host that is not yet known is trusted and recorded,
// which is the behaviour of ssh(1)'s StrictHostKeyChecking=accept-new. A host
// that is already known but presents a different key is still refused: that is
// the case this option is not intended to relax.
func WithAcceptNewHostKeys(accept bool) Option {
	return func(o *options) {
		o.acceptNewHost = accept
	}
}

// WithUser specifies the user name used to authenticate to the server. If it is
// not specified, the value of the USER environment variable is used.
func WithUser(user string) Option {
	return func(o *options) {
		o.user = user
	}
}

// WithLogger specifies the logger used to report connection events. If it is
// not specified, the logger from ctxlog.Logger is used.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// NewClient returns an instance of Client with the specified network and
// address, and optional configuration options. A connection is not
// established until ConnectAndWait is called.
func NewClient(ctx context.Context, network, addr string, opts ...Option) *Client {
	c := &Client{
		network: network,
		addr:    addr,
		doneCh:  make(chan struct{}),
	}
	for _, fn := range opts {
		fn(&c.opts)
	}
	if c.opts.dialTimeout == 0 {
		c.opts.dialTimeout = DefaultDialTimeout
	}
	if c.opts.user == "" {
		c.opts.user = os.Getenv("USER")
	}
	if c.opts.logger == nil {
		c.opts.logger = ctxlog.Logger(ctx)
	}
	c.opts.logger = c.opts.logger.With("ssh.network", network, "ssh.addr", addr)
	return c
}

// defaultKnownHostsFiles returns the files consulted by ssh(1) by default.
func defaultKnownHostsFiles() []string {
	var files []string
	if home, err := os.UserHomeDir(); err == nil {
		files = append(files, filepath.Join(home, ".ssh", "known_hosts"))
	}
	return append(files, "/etc/ssh/ssh_known_hosts")
}

// existingFiles returns those of files that can be opened. knownhosts.New
// fails if any file is missing, whereas ssh(1) simply ignores a known hosts
// file that is not there.
func existingFiles(files []string) []string {
	existing := make([]string, 0, len(files))
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			existing = append(existing, f)
		}
	}
	return existing
}

var knownHostsMu sync.Mutex

// hostKeyCallback returns the callback used to verify the server, honouring
// WithHostKey, WithKnownHosts and WithAcceptNewHostKeys in that order.
func (c *Client) hostKeyCallback() (ssh.HostKeyCallback, error) {
	if c.opts.hostKey != nil {
		return ssh.FixedHostKey(c.opts.hostKey), nil
	}
	files := c.opts.knownHosts
	if len(files) == 0 {
		files = defaultKnownHostsFiles()
	}
	// The file that a newly accepted host is recorded in is the first one
	// named, whether or not it exists yet.
	record := files[0]

	if c.opts.acceptNewHost && record == "/etc/ssh/ssh_known_hosts" {
		return nil, fmt.Errorf("%w: cannot record new host keys to system file %v: specify a user known hosts file with WithKnownHosts", ErrKnownHosts, record)
	}

	var callback ssh.HostKeyCallback
	if existing := existingFiles(files); len(existing) > 0 {
		var err error
		if callback, err = knownhosts.New(existing...); err != nil {
			return nil, fmt.Errorf("%w: reading %v: %v", ErrKnownHosts, existing, err)
		}
	} else {
		if !c.opts.acceptNewHost {
			return nil, fmt.Errorf("%w: none of %v exist and new host keys are not accepted: use WithKnownHosts or WithAcceptNewHostKeys", ErrKnownHosts, files)
		}
		// Every host is unknown, which acceptNewHostKey below records.
		callback = func(string, net.Addr, ssh.PublicKey) error {
			return &knownhosts.KeyError{}
		}
	}
	if !c.opts.acceptNewHost {
		return callback, nil
	}
	return acceptNewHostKey(callback, record), nil
}

// acceptNewHostKey wraps callback so that a host which is not yet known is
// trusted and appended to record. A host that is known but whose key differs
// is left to callback to refuse.
func acceptNewHostKey(callback ssh.HostKeyCallback, record string) ssh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := callback(hostname, remote, key)
		var keyErr *knownhosts.KeyError
		if !errors.As(err, &keyErr) || len(keyErr.Want) > 0 {
			// Either the key was accepted, or it was refused for a reason
			// that accepting new hosts does not cover: a mismatch with a key
			// already recorded for this host, or a revoked key.
			return err
		}
		if err := appendKnownHost(record, hostname, key); err != nil {
			return fmt.Errorf("%w: recording host key for %v into %v: %v", ErrKnownHosts, hostname, record, err)
		}
		return nil
	}
}

// appendKnownHost records key for hostname in the known hosts file at path,
// creating it if need be.
func appendKnownHost(path, hostname string, key ssh.PublicKey) (err error) {
	knownHostsMu.Lock()
	defer knownHostsMu.Unlock()

	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); err == nil {
			err = closeErr
		}
	}()
	line := knownhosts.Line([]string{knownhosts.Normalize(hostname)}, key)
	// A known hosts file written by other means may lack a final newline,
	// which would otherwise merge the existing last entry with this one.
	prefix := ""
	if fi, err := f.Stat(); err == nil && fi.Size() > 0 {
		var b [1]byte
		if _, err := f.ReadAt(b[:], fi.Size()-1); err == nil && b[0] != '\n' {
			prefix = "\n"
		}
	}
	_, err = f.WriteString(prefix + line + "\n")
	return err
}

// sshConfigForAgent returns a configuration that authenticates using the keys
// held by the local ssh agent, located via the SSH_AUTH_SOCK environment
// variable. golang.org/x/crypto/ssh does not consult the agent, ~/.ssh/config
// or ~/.ssh/id_* of its own accord, so this is the only means by which agent
// held keys are used.
//
// Private keys are never exposed to this process: every signature is computed
// by the agent over the returned connection, which must therefore remain open
// for the duration of the ssh handshake. The returned function closes it and
// must be called once the handshake has completed, or when this function's
// result is otherwise discarded.
//
// The keys are obtained via ssh.PublicKeysCallback rather than being read
// here, so that the agent is queried at handshake time: keys added to it after
// this call, or an agent that is unlocked later, are still found. Note that
// this offers the server every key the agent holds, so an agent holding more
// keys than the server's MaxAuthTries permits can be refused before the usable
// key is reached.
func (c *Client) sshConfigForAgent(user string) (*ssh.ClientConfig, func(), error) {
	hostKeyCallback, err := c.hostKeyCallback()
	if err != nil {
		return nil, nil, err
	}
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return nil, nil, fmt.Errorf("%w: SSH_AUTH_SOCK is not set", ErrNoAgent)
	}
	// The socket path comes from the environment by design: it is how the
	// agent advertises itself, and is a local unix socket rather than a
	// network address.
	conn, err := net.Dial("unix", sock) //nolint:gosec // G704: not a network address.
	if err != nil {
		// Not ErrNoAgent: the agent is configured but unreachable, which a
		// restarted agent resolves, so this is worth retrying.
		return nil, nil, fmt.Errorf("connecting to ssh agent at %v: %w", sock, err)
	}
	algorithms := ssh.SupportedAlgorithms()
	config := &ssh.ClientConfig{
		Config: ssh.Config{
			KeyExchanges: algorithms.KeyExchanges,
			Ciphers:      algorithms.Ciphers,
			MACs:         algorithms.MACs,
		},
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(agent.NewClient(conn).Signers),
		},
		HostKeyCallback: hostKeyCallback,
		// ssh(1) orders this by the key types it already has recorded for the
		// host, so that it is not offered a key type it cannot match. Doing
		// the same here needs the recorded types, which knownhosts does not
		// expose, so a host recorded under one type but offering another is
		// reported as a mismatch rather than as an unknown host.
		HostKeyAlgorithms: algorithms.HostKeys,
	}
	return config, func() { conn.Close() }, nil
}

// ConnectAndWait establishes a connection to the server, along with any
// configured port forwards, and maintains it: whenever the connection is lost
// it is established again. It does not return once the connection is up, but
// only when it stops being maintained, which is when the context is
// cancelled, when the client is closed, or when an attempt fails for a reason
// that retrying cannot resolve. The error returned says which; Retryable
// describes how that last case is determined.
//
// backoff is called to obtain the delays between attempts, and is called again
// after each successful connection so that the delays accumulated reaching a
// server do not carry over into the next outage. A backoff that gives up
// bounds how long a connection is retried for, and ConnectAndWait then returns
// the error from the last attempt.
func (c *Client) ConnectAndWait(ctx context.Context, backoff func() ratecontrol.Backoff) error {
	select {
	case <-c.doneCh:
		return ErrClientClosed
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		select {
		case <-c.doneCh:
			cancel()
		case <-runCtx.Done():
		}
	}()

	errCh := make(chan error, 1)
	go c.connect(runCtx, backoff, errCh)

	select {
	case <-c.doneCh:
		cancel()
		<-errCh
		return ErrClientClosed
	case <-ctx.Done():
		cancel()
		<-errCh
		return ctx.Err()
	case err := <-errCh:
		select {
		case <-c.doneCh:
			return ErrClientClosed
		default:
		}
		return err
	}
}

func (c *Client) stoppingErr(ctx context.Context) error {
	select {
	case <-c.doneCh:
		return ErrClientClosed
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (c *Client) wait(ctx context.Context, client *ssh.Client, pf *forwarder) error {
	waitDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = client.Close()
		case <-c.doneCh:
			_ = client.Close()
		case <-waitDone:
		}
	}()
	waitErr := client.Wait()
	close(waitDone)
	if waitErr != nil {
		c.opts.logger.Debug("ssh connection closed", "err", waitErr)
	}
	pf.stopAll(context.WithoutCancel(ctx))
	_ = client.Close()
	return c.stoppingErr(ctx)
}

func (c *Client) createConnectionAndForwarding(ctx context.Context) (*ssh.Client, *forwarder, error) {
	config, closeAgent, err := c.sshConfigForAgent(c.opts.user)
	if err != nil {
		return nil, nil, err
	}
	// The agent is needed only whilst the handshake below is signing, and is
	// redialled on every reconnection so that a restarted agent is picked up.
	defer closeAgent()

	dialer := net.Dialer{Timeout: c.opts.dialTimeout}
	conn, err := dialer.DialContext(ctx, c.network, c.addr)
	if err != nil {
		return nil, nil, err
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, c.addr, config)
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	pf := &forwarder{
		client:   client,
		forwards: c.opts.localForwards,
		logger:   c.opts.logger,
	}
	if err := pf.forwardAll(ctx); err != nil {
		pf.stopAll(context.WithoutCancel(ctx))
		conn.Close()
		client.Close()
		return nil, nil, err
	}
	return client, pf, nil
}

func (c *Client) connect(ctx context.Context, backoffFn func() ratecontrol.Backoff, ch chan<- error) {
	backoff := backoffFn()
	for {
		if err := c.stoppingErr(ctx); err != nil {
			ch <- err
			return
		}

		client, forwarder, err := c.createConnectionAndForwarding(ctx)
		if err == nil {
			if err := c.wait(ctx, client, forwarder); err != nil {
				ch <- err
				return
			}
			// The connection was established, so the delays accumulated
			// reaching it no longer describe the state of the server.
			backoff = backoffFn()
			continue
		}

		if err := c.stoppingErr(ctx); err != nil {
			ch <- err
			return
		}
		if !Retryable(err) {
			ch <- err
			return
		}
		select {
		case <-c.doneCh:
			ch <- ErrClientClosed
			return
		case <-ctx.Done():
			ch <- ctx.Err()
			return
		case _, ok := <-backoff.Next():
			if !ok {
				ch <- fmt.Errorf("backoff exhausted: %w", err)
				return
			}
		}
	}
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.doneCh)
	})
}
