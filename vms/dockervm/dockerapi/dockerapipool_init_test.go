// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package dockerapi_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync/atomic"
	"time"

	"cloudeng.io/cicd"
	"cloudeng.io/vms"
	"cloudeng.io/vms/dockervm/dockerapi"
	"cloudeng.io/vms/vmspool"
	"cloudeng.io/vms/vmstestutil"
)

// dockerAPIConstructor implements vmspool.Constructor for API-backed containers.
type dockerAPIConstructor struct {
	image   string
	counter atomic.Int64
}

func (c *dockerAPIConstructor) New() vms.Instance {
	n := c.counter.Add(1)
	name := fmt.Sprintf("vmstest-%d-%d", time.Now().Unix()%100000, n)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{})).With("test", name, "image", c.image)
	inst, err := dockerapi.New(context.Background(), c.image, name, dockerapi.WithLogger(logger))
	if err != nil {
		panic(fmt.Sprintf("dockerapi.New: %v", err))
	}
	return inst
}

func newDockerAPIConstructor() *dockerAPIConstructor {
	return &dockerAPIConstructor{image: testImage}
}

func rwc(id string) io.Writer {
	return os.Stderr
}

//go:generate astest --import "cloudeng.io/cicd" --match='^TestPool' --preamble=cicd.LongRunningTest(t,1);cfg=poolConfig.Get(t.Name()) --pkg-path cloudeng.io/vms/vmstestutil ./dockerapipool_test.go
//go:generate astest --import "cloudeng.io/cicd" --match='^TestInstance' --preamble=cicd.LongRunningTest(t,1);cfg=instanceConfig.Get(t.Name()) --pkg-path cloudeng.io/vms/vmstestutil ./dockerapiinstance_test.go

var poolConfig = cicd.ConfigManager[vmstestutil.PoolTestConfig]{}

var defaultPoolConfig = vmstestutil.PoolTestConfig{
	Constructor:      newDockerAPIConstructor(),
	PoolSize:         2,
	ExecCmd:          "echo",
	ExecArgs:         []string{"hello"},
	ExecStdoutOutput: "hello\n",

	StdoutRWC: rwc,
	StderrRWC: rwc,

	Timeout:          5 * time.Minute,
	StagingBehaviour: vmspool.StagingBehaviourStopped,
}

var instanceConfig = cicd.ConfigManager[vmstestutil.InstanceTestConfig]{}

var defaultInstanceConfig = vmstestutil.InstanceTestConfig{
	Constructor: newDockerAPIConstructor(),

	Timeout: 5 * time.Minute,

	ExecCmd:    "echo",
	ExecArgs:   []string{"hello"},
	ExecStdout: "hello\n",
	ExecStderr: "",

	RequireUnderlyingState: dockerAPIRequireState,
}

func init() {
	poolConfig.SetDefault(defaultPoolConfig)
	instanceConfig.SetDefault(defaultInstanceConfig)
}

// dockerAPILookup inspects the named container using the Docker API.
func dockerAPILookup(ctx context.Context, name string) (dockerapi.ContainerInfo, bool, error) {
	info, found, err := dockerapi.InspectContainer(ctx, name)
	if err != nil {
		return dockerapi.ContainerInfo{}, false, fmt.Errorf("docker inspect: %v", err)
	}
	return info, found, nil
}

func dockerAPIRequireState(ctx context.Context, inst vms.Instance, msg string, final vms.State, intermediate ...vms.State) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	if err := vms.WaitForState(ctx, inst, time.Millisecond, final, intermediate...); err != nil {
		return fmt.Errorf("++: %s: waiting for VMS state %v: %v", msg, final, err)
	}
	if final == vms.StateInitial {
		return nil
	}

	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	info, found, err := dockerAPILookup(ctx, inst.ID())
	if err != nil {
		return err
	}

	wantStatus := func(status string) error {
		if !found {
			return fmt.Errorf("++: %s: docker inspect: container %q not found", msg, inst.ID())
		}
		if info.State.Status != status {
			return fmt.Errorf("++: %s: docker inspect: container %q status=%q, want %q", msg, inst.ID(), info.State.Status, status)
		}
		return nil
	}

	switch final {
	case vms.StateDeleted:
		if found {
			return fmt.Errorf("++: %s: docker inspect: container %q still present after delete", msg, inst.ID())
		}
	case vms.StateRunning:
		return wantStatus("running")
	case vms.StateStopped:
		if !found {
			return fmt.Errorf("++: %s: docker inspect: container %q not found", msg, inst.ID())
		}
		if info.State.Status != "exited" && info.State.Status != "created" {
			return fmt.Errorf("++: %s: docker inspect: container %q status=%q, want exited or created", msg, inst.ID(), info.State.Status)
		}
	}
	return nil
}
