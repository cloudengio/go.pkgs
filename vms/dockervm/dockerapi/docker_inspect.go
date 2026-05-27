// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dockerapi

import (
	"context"
	"fmt"
	"strings"

	"github.com/containerd/errdefs"
	sdkclient "github.com/docker/go-sdk/client"
	mobyclient "github.com/moby/moby/client"

	"cloudeng.io/vms"
)

// ContainerStateInfo represents the state portion of a Docker container inspection.
type ContainerStateInfo struct {
	Status     string // "created", "running", "paused", "restarting", "removing", "exited", "dead"
	Running    bool
	Paused     bool
	Restarting bool
	Dead       bool
	ExitCode   int
}

// ContainerNetworkInfo represents the primary network information for a container.
type ContainerNetworkInfo struct {
	IPAddress string
}

// ContainerInfo holds the parsed result of a Docker container inspection.
type ContainerInfo struct {
	Name            string
	State           ContainerStateInfo
	NetworkSettings ContainerNetworkInfo
}

// VMSState maps the Docker container status to a vms.State.
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

// InspectContainerClient queries the Docker daemon using the supplied client for the named container using the Docker API.
// Returns (zero, false, nil) if the container does not exist.
func InspectContainerClient(ctx context.Context, client sdkclient.SDKClient, name string) (ContainerInfo, bool, error) {

	result, err := client.ContainerInspect(ctx, name, mobyclient.ContainerInspectOptions{})
	if err != nil {
		if errdefs.IsNotFound(err) {
			return ContainerInfo{}, false, nil
		}
		return ContainerInfo{}, false, fmt.Errorf("docker inspect %s: %w", name, err)
	}

	resp := result.Container
	var state ContainerStateInfo
	if resp.State != nil {
		state = ContainerStateInfo{
			Status:     string(resp.State.Status),
			Running:    resp.State.Running,
			Paused:     resp.State.Paused,
			Restarting: resp.State.Restarting,
			Dead:       resp.State.Dead,
			ExitCode:   resp.State.ExitCode,
		}
	}

	var ip string
	if resp.NetworkSettings != nil {
		for _, nw := range resp.NetworkSettings.Networks {
			if nw != nil && nw.IPAddress.IsValid() {
				ip = nw.IPAddress.String()
				break
			}
		}
	}

	return ContainerInfo{
		Name:            strings.TrimPrefix(resp.Name, "/"),
		State:           state,
		NetworkSettings: ContainerNetworkInfo{IPAddress: ip},
	}, true, nil
}

// InspectContainer queries the Docker daemon for the named container using the Docker API.
// Returns (zero, false, nil) if the container does not exist.
func InspectContainer(ctx context.Context, name string) (ContainerInfo, bool, error) {
	client, err := sdkclient.New(ctx)
	if err != nil {
		return ContainerInfo{}, false, fmt.Errorf("docker client: %w", err)
	}
	return InspectContainerClient(ctx, client, name)
}
