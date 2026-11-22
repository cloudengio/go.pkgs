// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// ptyRequest is what the server was asked for by a pty-req or a window-change
// request.
type ptyRequest struct {
	Type       string
	Term       string
	Cols, Rows uint32
}

var (
	ptyHookMu sync.Mutex
	ptyHook   chan<- ptyRequest
)

// observePTYRequests returns the pty-req and window-change requests that
// sessions subsequently receive, until the test ends.
func observePTYRequests(t *testing.T) <-chan ptyRequest {
	t.Helper()
	ch := make(chan ptyRequest, 16)
	ptyHookMu.Lock()
	ptyHook = ch
	ptyHookMu.Unlock()
	t.Cleanup(func() {
		ptyHookMu.Lock()
		ptyHook = nil
		ptyHookMu.Unlock()
	})
	return ch
}

func reportPTYRequest(req *ssh.Request) {
	var r ptyRequest
	r.Type = req.Type
	switch req.Type {
	case "pty-req":
		var p struct {
			Term          string
			Cols, Rows    uint32
			Width, Height uint32
			Modes         string
		}
		if ssh.Unmarshal(req.Payload, &p) == nil {
			r.Term, r.Cols, r.Rows = p.Term, p.Cols, p.Rows
		}
	case "window-change":
		var p struct{ Cols, Rows, Width, Height uint32 }
		if ssh.Unmarshal(req.Payload, &p) == nil {
			r.Cols, r.Rows = p.Cols, p.Rows
		}
	}
	ptyHookMu.Lock()
	defer ptyHookMu.Unlock()
	if ptyHook != nil {
		ptyHook <- r
	}
}

// agentForwarding records what servers see of agent forwarding.
type agentForwarding struct {
	requests atomic.Int32
	deny     atomic.Bool
	listed   chan agentListing
}

// agentListing is what the client's agent, reached through the forwarded
// channel, listed.
type agentListing struct {
	keys []*agent.Key
	err  error
}

var forwardingHook atomic.Pointer[agentForwarding]

// observeAgentForwarding returns what subsequent sessions' requests for agent
// forwarding result in, until the test ends.
func observeAgentForwarding(t *testing.T) *agentForwarding {
	t.Helper()
	af := &agentForwarding{listed: make(chan agentListing, 16)}
	forwardingHook.Store(af)
	t.Cleanup(func() { forwardingHook.Store(nil) })
	return af
}

// serveAgentRequest answers a request for agent forwarding as sshd does,
// by opening an agent channel to the client for the session's use. Here the
// server uses it itself, to show that it reaches the client's agent.
func serveAgentRequest(req *ssh.Request, conn ssh.Conn) {
	af := forwardingHook.Load()
	if af == nil || conn == nil || af.deny.Load() {
		if af != nil {
			af.requests.Add(1)
		}
		req.Reply(false, nil) //nolint:errcheck
		return
	}
	af.requests.Add(1)
	req.Reply(true, nil) //nolint:errcheck
	go func() {
		ch, reqs, err := conn.OpenChannel("auth-agent@openssh.com", nil)
		if err != nil {
			af.listed <- agentListing{err: err}
			return
		}
		go ssh.DiscardRequests(reqs)
		defer ch.Close()
		keys, err := agent.NewClient(ch).List()
		af.listed <- agentListing{keys: keys, err: err}
	}()
}

// serveSession implements the server side of a session channel for exec
// requests, as sshd does: the command is run by /bin/sh -c, its standard
// streams are the channel's and its exit status is sent when it completes. A
// SIGTERM signal request is delivered to the command.
func serveSession(nc ssh.NewChannel, conn ssh.Conn) {
	ch, reqs, err := nc.Accept()
	if err != nil {
		return
	}
	var cmd *exec.Cmd
	for req := range reqs {
		switch req.Type {
		case "exec":
			if cmd != nil {
				req.Reply(false, nil) //nolint:errcheck
				continue
			}
			cmd = startExec(ch, req)
		case "pty-req", "window-change":
			reportPTYRequest(req)
			req.Reply(true, nil) //nolint:errcheck
		case "auth-agent-req@openssh.com":
			serveAgentRequest(req, conn)
		case "signal":
			var payload struct{ Signal string }
			if cmd != nil && ssh.Unmarshal(req.Payload, &payload) == nil && payload.Signal == "TERM" {
				cmd.Process.Signal(syscall.SIGTERM) //nolint:errcheck
			}
			req.Reply(false, nil) //nolint:errcheck
		default:
			req.Reply(false, nil) //nolint:errcheck
		}
	}
}

// startExec starts the command named by an exec request, returning nil if it
// could not be started. Its completion is reported on ch.
func startExec(ch ssh.Channel, req *ssh.Request) *exec.Cmd {
	var payload struct{ Command string }
	if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
		req.Reply(false, nil) //nolint:errcheck
		return nil
	}
	cmd := exec.Command("/bin/sh", "-c", payload.Command) //nolint:gosec
	cmd.Stdout, cmd.Stderr = ch, ch.Stderr()
	// As sshd does, do not wait for input that a finished command will never
	// read: the copy is not waited for.
	stdin, err := cmd.StdinPipe()
	if err != nil {
		req.Reply(false, nil) //nolint:errcheck
		return nil
	}
	go func() {
		io.Copy(stdin, ch) //nolint:errcheck
		stdin.Close()
	}()
	if err := cmd.Start(); err != nil {
		req.Reply(false, nil) //nolint:errcheck
		ch.Close()
		return nil
	}
	req.Reply(true, nil) //nolint:errcheck
	go func() {
		reportExit(ch, cmd.Wait())
		ch.Close()
	}()
	return cmd
}

// reportExit sends the exit status, or the signal that killed the command, as
// sshd does.
func reportExit(ch ssh.Channel, err error) {
	status := 0
	if ee, ok := err.(*exec.ExitError); ok {
		status = ee.ExitCode()
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			ch.SendRequest("exit-signal", false, ssh.Marshal(struct { //nolint:errcheck
				Signal string
				Core   bool
				Msg    string
				Lang   string
			}{ws.Signal().String(), false, "", ""}))
			return
		}
	} else if err != nil {
		status = 1
	}
	ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{uint32(status)})) //nolint:errcheck
}
