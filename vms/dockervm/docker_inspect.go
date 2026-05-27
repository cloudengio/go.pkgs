// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dockervm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"cloudeng.io/os/executil"
	"cloudeng.io/vms"
)

// ContainerStateInfo represents the State portion of docker inspect output.
type ContainerStateInfo struct {
	Status     string // "created", "running", "paused", "restarting", "removing", "exited", "dead"
	Running    bool
	Paused     bool
	Restarting bool
	Dead       bool
	ExitCode   int
}

// ContainerNetworkInfo represents the network information for a container.
type ContainerNetworkInfo struct {
	IPAddress string
}

// ContainerInfo holds the parsed output from "docker inspect <name>".
type ContainerInfo struct {
	Name            string
	State           ContainerStateInfo
	NetworkSettings ContainerNetworkInfo
}

// VMSState maps the docker container status to a vms.State.
func (c ContainerInfo) VMSState() vms.State {
	switch c.State.Status {
	case "running":
		return vms.StateRunning
	case "paused":
		return vms.StateSuspended
	case "created", "exited":
		return vms.StateStopped
	case "dead":
		return vms.StateErrorUnknown
	default:
		return vms.StateInitial
	}
}

// InspectContainer runs "docker inspect <name>" and returns the parsed result.
// Returns (zero, false, nil) if the container does not exist.
func InspectContainer(ctx context.Context, name string) (ContainerInfo, bool, error) {
	stdoutBuf := bytes.NewBuffer(make([]byte, 0, 4096))
	stderrBuf := executil.NewTailWriter(1024)
	cmd := exec.CommandContext(ctx, "docker", "inspect", name)
	cmd.Stdout = stdoutBuf
	cmd.Stderr = stderrBuf
	if err := cmd.Run(); err != nil {
		stderr := string(stderrBuf.Bytes())
		if isContainerNotFound(stderr) {
			return ContainerInfo{}, false, nil
		}
		return ContainerInfo{}, false, fmt.Errorf("docker inspect %s: %s: %w", name, stderr, err)
	}

	var raw []struct {
		Name  string
		State struct {
			Status     string
			Running    bool
			Paused     bool
			Restarting bool
			Dead       bool
			ExitCode   int
		}
		NetworkSettings struct {
			IPAddress string
		}
	}
	if err := json.Unmarshal(stdoutBuf.Bytes(), &raw); err != nil {
		return ContainerInfo{}, false, fmt.Errorf("docker inspect %s: parse JSON: %w", name, err)
	}
	if len(raw) == 0 {
		return ContainerInfo{}, false, nil
	}
	r := raw[0]
	return ContainerInfo{
		Name: strings.TrimPrefix(r.Name, "/"),
		State: ContainerStateInfo{
			Status:     r.State.Status,
			Running:    r.State.Running,
			Paused:     r.State.Paused,
			Restarting: r.State.Restarting,
			Dead:       r.State.Dead,
			ExitCode:   r.State.ExitCode,
		},
		NetworkSettings: ContainerNetworkInfo{
			IPAddress: r.NetworkSettings.IPAddress,
		},
	}, true, nil
}
