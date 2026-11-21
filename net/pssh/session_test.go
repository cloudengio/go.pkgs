// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package pssh

import (
	"os/exec"
	"syscall"

	"golang.org/x/crypto/ssh"
)

// serveSession implements the server side of a session channel for exec
// requests, as sshd does: the command is run by /bin/sh -c, its standard
// streams are the channel's and its exit status is sent when it completes. A
// SIGTERM signal request is delivered to the command.
func serveSession(nc ssh.NewChannel) {
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
	cmd.Stdin, cmd.Stdout, cmd.Stderr = ch, ch, ch.Stderr()
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
