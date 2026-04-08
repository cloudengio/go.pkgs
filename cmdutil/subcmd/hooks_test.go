// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package subcmd_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"cloudeng.io/cmdutil/subcmd"
)

type hookTraceKey struct{}

// appendTrace appends a string to the hook trace stored in the context.
func appendTrace(ctx context.Context, s string) context.Context {
	existing, _ := ctx.Value(hookTraceKey{}).([]string)
	return context.WithValue(ctx, hookTraceKey{}, append(existing, s))
}

func getTrace(ctx context.Context) []string {
	v, _ := ctx.Value(hookTraceKey{}).([]string)
	return v
}

// newTracingHook returns a PreHook that appends preMsg to the trace before the
// runner and postMsg after; a nil postMsg means no PostHook is returned.
func newTracingHook(preMsg string, postMsg string) subcmd.PreHook {
	return func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		ctx = appendTrace(ctx, preMsg)
		if postMsg == "" {
			return ctx, nil, nil
		}
		post := func(ctx context.Context) error {
			_ = appendTrace(ctx, postMsg) // context returned here is discarded
			return nil
		}
		return ctx, post, nil
	}
}

func TestPreHookRunsBeforeRunner(t *testing.T) {
	ctx := context.Background()
	var capturedCtx context.Context

	runner := func(ctx context.Context, _ any, _ []string) error {
		capturedCtx = ctx
		return nil
	}

	hook := newTracingHook("pre", "")
	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner,
		subcmd.WithoutArguments(),
		subcmd.WithPreHooks(hook),
	)
	cmd.Document("test command")
	cs := subcmd.NewCommandSet(cmd)
	if err := cs.DispatchWithArgs(ctx, "test", "cmd"); err != nil {
		t.Fatal(err)
	}
	trace := getTrace(capturedCtx)
	if len(trace) != 1 || trace[0] != "pre" {
		t.Errorf("expected trace [pre], got %v", trace)
	}
}

func TestPostHookRunsAfterRunner(t *testing.T) {
	ctx := context.Background()
	postCalled := false

	runner := func(_ context.Context, _ any, _ []string) error {
		return nil
	}
	pre := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		post := func(_ context.Context) error {
			postCalled = true
			return nil
		}
		return ctx, post, nil
	}

	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner,
		subcmd.WithoutArguments(),
		subcmd.WithPreHooks(pre),
	)
	cmd.Document("test command")
	cs := subcmd.NewCommandSet(cmd)
	if err := cs.DispatchWithArgs(ctx, "test", "cmd"); err != nil {
		t.Fatal(err)
	}
	if !postCalled {
		t.Error("post-hook was not called")
	}
}

func TestNilPostHookIsIgnored(t *testing.T) {
	ctx := context.Background()

	runner := func(_ context.Context, _ any, _ []string) error { return nil }
	pre := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		return ctx, nil, nil // no post-hook
	}

	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner,
		subcmd.WithoutArguments(),
		subcmd.WithPreHooks(pre),
	)
	cmd.Document("test command")
	cs := subcmd.NewCommandSet(cmd)
	// Must not panic or return an error.
	if err := cs.DispatchWithArgs(ctx, "test", "cmd"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMultiplePreHooksRunInOrder(t *testing.T) {
	ctx := context.Background()
	var order []string

	makeHook := func(name string) subcmd.PreHook {
		return func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
			order = append(order, "pre-"+name)
			post := func(_ context.Context) error {
				order = append(order, "post-"+name)
				return nil
			}
			return ctx, post, nil
		}
	}

	runner := func(_ context.Context, _ any, _ []string) error {
		order = append(order, "runner")
		return nil
	}

	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner,
		subcmd.WithoutArguments(),
		subcmd.WithPreHooks(makeHook("a"), makeHook("b")),
	)
	cmd.Document("test command")
	cs := subcmd.NewCommandSet(cmd)
	if err := cs.DispatchWithArgs(ctx, "test", "cmd"); err != nil {
		t.Fatal(err)
	}

	// Post-hooks run in LIFO order (last registered, first called).
	want := []string{"pre-a", "pre-b", "runner", "post-b", "post-a"}
	if len(order) != len(want) {
		t.Fatalf("got %v, want %v", order, want)
	}
	for i, v := range want {
		if order[i] != v {
			t.Errorf("step %d: got %q, want %q", i, order[i], v)
		}
	}
}

func TestPreHookContextPropagation(t *testing.T) {
	type ctxKey struct{}
	ctx := context.Background()

	pre := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		return context.WithValue(ctx, ctxKey{}, "injected"), nil, nil
	}

	var gotValue string
	runner := func(ctx context.Context, _ any, _ []string) error {
		gotValue, _ = ctx.Value(ctxKey{}).(string)
		return nil
	}

	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner,
		subcmd.WithoutArguments(),
		subcmd.WithPreHooks(pre),
	)
	cmd.Document("test command")
	cs := subcmd.NewCommandSet(cmd)
	if err := cs.DispatchWithArgs(ctx, "test", "cmd"); err != nil {
		t.Fatal(err)
	}
	if gotValue != "injected" {
		t.Errorf("got %q, want %q", gotValue, "injected")
	}
}

func TestPreHookErrorPreventsRunner(t *testing.T) {
	ctx := context.Background()
	runnerCalled := false

	pre := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		return ctx, nil, fmt.Errorf("hook error")
	}
	runner := func(_ context.Context, _ any, _ []string) error {
		runnerCalled = true
		return nil
	}

	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner,
		subcmd.WithoutArguments(),
		subcmd.WithPreHooks(pre),
	)
	cmd.Document("test command")
	cs := subcmd.NewCommandSet(cmd)
	err := cs.DispatchWithArgs(ctx, "test", "cmd")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "pre-hook failed") {
		t.Errorf("error should mention pre-hook failed: %v", err)
	}
	if !strings.Contains(err.Error(), "hook error") {
		t.Errorf("error should contain original error: %v", err)
	}
	if runnerCalled {
		t.Error("runner should not have been called after pre-hook failure")
	}
}

func TestPreHookErrorRunsAlreadyRegisteredPostHooks(t *testing.T) {
	ctx := context.Background()
	postACalled := false

	hookA := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		post := func(_ context.Context) error {
			postACalled = true
			return nil
		}
		return ctx, post, nil
	}
	hookB := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		return ctx, nil, fmt.Errorf("hookB failed")
	}

	runner := func(_ context.Context, _ any, _ []string) error { return nil }

	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner,
		subcmd.WithoutArguments(),
		subcmd.WithPreHooks(hookA, hookB),
	)
	cmd.Document("test command")
	cs := subcmd.NewCommandSet(cmd)
	err := cs.DispatchWithArgs(ctx, "test", "cmd")
	if err == nil {
		t.Fatal("expected an error from hookB")
	}
	if !postACalled {
		t.Error("post-hook from hookA should have been called even though hookB failed")
	}
}

func TestPostHookErrorIsReported(t *testing.T) {
	ctx := context.Background()

	pre := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		post := func(_ context.Context) error {
			return fmt.Errorf("post-hook error")
		}
		return ctx, post, nil
	}
	runner := func(_ context.Context, _ any, _ []string) error { return nil }

	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner,
		subcmd.WithoutArguments(),
		subcmd.WithPreHooks(pre),
	)
	cmd.Document("test command")
	cs := subcmd.NewCommandSet(cmd)
	err := cs.DispatchWithArgs(ctx, "test", "cmd")
	if err == nil {
		t.Fatal("expected an error from post-hook")
	}
	if !strings.Contains(err.Error(), "post-hook failed") {
		t.Errorf("error should mention post-hook failed: %v", err)
	}
	if !strings.Contains(err.Error(), "post-hook error") {
		t.Errorf("error should contain original error: %v", err)
	}
}

func TestAppendPreHooksOnCommand(t *testing.T) {
	ctx := context.Background()
	var order []string

	runner := func(_ context.Context, _ any, _ []string) error {
		order = append(order, "runner")
		return nil
	}
	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner, subcmd.WithoutArguments())
	cmd.Document("test command")

	cmd.AppendPreHooks(func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		order = append(order, "hook1")
		return ctx, nil, nil
	})
	cmd.AppendPreHooks(func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		order = append(order, "hook2")
		return ctx, nil, nil
	})

	cs := subcmd.NewCommandSet(cmd)
	if err := cs.DispatchWithArgs(ctx, "test", "cmd"); err != nil {
		t.Fatal(err)
	}
	want := []string{"hook1", "hook2", "runner"}
	if len(order) != len(want) {
		t.Fatalf("got %v, want %v", order, want)
	}
	for i, v := range want {
		if order[i] != v {
			t.Errorf("step %d: got %q, want %q", i, order[i], v)
		}
	}
}

func TestAppendPreHooksOnCommandSet(t *testing.T) {
	ctx := context.Background()
	hookCallCount := 0

	hook := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		hookCallCount++
		return ctx, nil, nil
	}

	runner := func(_ context.Context, _ any, _ []string) error { return nil }
	cmdA := subcmd.NewCommand("cmd-a", subcmd.NewFlagSet(), runner, subcmd.WithoutArguments())
	cmdA.Document("cmd a")
	cmdB := subcmd.NewCommand("cmd-b", subcmd.NewFlagSet(), runner, subcmd.WithoutArguments())
	cmdB.Document("cmd b")

	cs := subcmd.NewCommandSet(cmdA, cmdB)
	cs.AppendPreHooks(hook)

	hookCallCount = 0
	if err := cs.DispatchWithArgs(ctx, "test", "cmd-a"); err != nil {
		t.Fatal(err)
	}
	if hookCallCount != 1 {
		t.Errorf("hook called %d times for cmd-a, want 1", hookCallCount)
	}

	hookCallCount = 0
	if err := cs.DispatchWithArgs(ctx, "test", "cmd-b"); err != nil {
		t.Fatal(err)
	}
	if hookCallCount != 1 {
		t.Errorf("hook called %d times for cmd-b, want 1", hookCallCount)
	}
}

func TestSetPreHooksOnCommandSet(t *testing.T) {
	ctx := context.Background()

	hookA := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		return appendTrace(ctx, "hookA"), nil, nil
	}
	hookB := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		return appendTrace(ctx, "hookB"), nil, nil
	}

	var capturedCtx context.Context
	runner := func(ctx context.Context, _ any, _ []string) error {
		capturedCtx = ctx
		return nil
	}

	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner, subcmd.WithoutArguments())
	cmd.Document("test command")
	cs := subcmd.NewCommandSet(cmd)

	// Set hookA, run, replace with hookB, run — only hookB should appear.
	cs.AppendPreHooks(hookA)
	cs.SetPreHooks(hookB)

	if err := cs.DispatchWithArgs(ctx, "test", "cmd"); err != nil {
		t.Fatal(err)
	}
	trace := getTrace(capturedCtx)
	if len(trace) != 1 || trace[0] != "hookB" {
		t.Errorf("expected trace [hookB], got %v", trace)
	}
}

func TestPreHookWithSubcommands(t *testing.T) {
	ctx := context.Background()
	hookCallCount := 0

	hook := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		hookCallCount++
		return ctx, nil, nil
	}

	runner := func(_ context.Context, _ any, _ []string) error { return nil }
	inner := subcmd.NewCommand("inner", subcmd.NewFlagSet(), runner, subcmd.WithoutArguments())
	inner.Document("inner command")
	innerSet := subcmd.NewCommandSet(inner)

	outer := subcmd.NewCommandLevel("outer", innerSet)
	outer.Document("outer command")
	cs := subcmd.NewCommandSet(outer)
	cs.AppendPreHooks(hook)

	if err := cs.DispatchWithArgs(ctx, "test", "outer", "inner"); err != nil {
		t.Fatal(err)
	}
	if hookCallCount != 1 {
		t.Errorf("hook called %d times, want 1", hookCallCount)
	}
}

func TestPostHooksLIFOOrder(t *testing.T) {
	ctx := context.Background()
	var order []string

	makeHook := func(name string) subcmd.PreHook {
		return func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
			order = append(order, "pre-"+name)
			post := func(_ context.Context) error {
				order = append(order, "post-"+name)
				return nil
			}
			return ctx, post, nil
		}
	}

	runner := func(_ context.Context, _ any, _ []string) error {
		order = append(order, "runner")
		return nil
	}

	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner,
		subcmd.WithoutArguments(),
		subcmd.WithPreHooks(makeHook("a"), makeHook("b"), makeHook("c")),
	)
	cmd.Document("test command")
	cs := subcmd.NewCommandSet(cmd)
	if err := cs.DispatchWithArgs(ctx, "test", "cmd"); err != nil {
		t.Fatal(err)
	}

	// Pre-hooks run FIFO; post-hooks run LIFO (reverse of registration order).
	want := []string{"pre-a", "pre-b", "pre-c", "runner", "post-c", "post-b", "post-a"}
	if len(order) != len(want) {
		t.Fatalf("got %v, want %v", order, want)
	}
	for i, v := range want {
		if order[i] != v {
			t.Errorf("step %d: got %q, want %q", i, order[i], v)
		}
	}
}

func ExampleWithPreHooks() {
	type dbKey struct{}

	// A PreHook that "opens" a resource and closes it via its PostHook.
	dbHook := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		fmt.Println("pre: opening db connection")
		ctx = context.WithValue(ctx, dbKey{}, "db-handle")
		post := func(context.Context) error {
			fmt.Println("post: closing db connection")
			return nil
		}
		return ctx, post, nil
	}

	runner := func(ctx context.Context, _ any, _ []string) error {
		handle, _ := ctx.Value(dbKey{}).(string)
		fmt.Printf("runner: using %s\n", handle)
		return nil
	}

	cmd := subcmd.NewCommand("query", subcmd.NewFlagSet(), runner,
		subcmd.WithoutArguments(),
		subcmd.WithPreHooks(dbHook),
	)
	cmd.Document("run a query")
	cs := subcmd.NewCommandSet(cmd)
	if err := cs.DispatchWithArgs(context.Background(), "mytool", "query"); err != nil {
		panic(err)
	}
	// Output:
	// pre: opening db connection
	// runner: using db-handle
	// post: closing db connection
}

func TestMidChainPreHookFailure(t *testing.T) {
	ctx := context.Background()

	// 5 pre-hooks; hook 3 fails.
	// Hooks 1 and 2 each register a post-hook.
	// Hooks 4 and 5 must never be called.
	// The runner must never be called.
	// Post-hooks for hooks 1 and 2 must run in LIFO order.

	var order []string
	failErr := fmt.Errorf("hook-3 failed")

	makeHook := func(name string) subcmd.PreHook {
		return func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
			order = append(order, "pre-"+name)
			post := func(_ context.Context) error {
				order = append(order, "post-"+name)
				return nil
			}
			return ctx, post, nil
		}
	}
	failingHook := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		order = append(order, "pre-3")
		return ctx, nil, failErr
	}

	runner := func(_ context.Context, _ any, _ []string) error {
		order = append(order, "runner")
		return nil
	}

	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner,
		subcmd.WithoutArguments(),
		subcmd.WithPreHooks(
			makeHook("1"),
			makeHook("2"),
			failingHook,
			makeHook("4"),
			makeHook("5"),
		),
	)
	cmd.Document("test command")
	cs := subcmd.NewCommandSet(cmd)
	err := cs.DispatchWithArgs(ctx, "test", "cmd")

	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "hook-3 failed") {
		t.Errorf("error should contain original cause: %v", err)
	}
	if !strings.Contains(err.Error(), "pre-hook failed") {
		t.Errorf("error should mention pre-hook failed: %v", err)
	}

	// LIFO: post-2 runs before post-1; hooks 4 and 5 and runner never run.
	want := []string{"pre-1", "pre-2", "pre-3", "post-2", "post-1"}
	if len(order) != len(want) {
		t.Fatalf("got %v, want %v", order, want)
	}
	for i, v := range want {
		if order[i] != v {
			t.Errorf("step %d: got %q, want %q", i, order[i], v)
		}
	}
}

func TestMidChainPreHookFailureWithPostHookError(t *testing.T) {
	// Same scenario, but a compensating post-hook also fails.
	// Both the pre-hook error and the post-hook error must appear in the
	// returned error; the other post-hook must still run.
	ctx := context.Background()

	var order []string

	hook1 := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		order = append(order, "pre-1")
		post := func(_ context.Context) error {
			order = append(order, "post-1")
			return fmt.Errorf("post-1 error")
		}
		return ctx, post, nil
	}
	hook2 := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		order = append(order, "pre-2")
		post := func(_ context.Context) error {
			order = append(order, "post-2")
			return nil
		}
		return ctx, post, nil
	}
	hook3 := func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
		order = append(order, "pre-3")
		return ctx, nil, fmt.Errorf("pre-3 error")
	}

	runner := func(_ context.Context, _ any, _ []string) error {
		order = append(order, "runner")
		return nil
	}

	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner,
		subcmd.WithoutArguments(),
		subcmd.WithPreHooks(hook1, hook2, hook3),
	)
	cmd.Document("test command")
	cs := subcmd.NewCommandSet(cmd)
	err := cs.DispatchWithArgs(ctx, "test", "cmd")

	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "pre-3 error") {
		t.Errorf("error should contain pre-hook cause: %v", err)
	}
	if !strings.Contains(err.Error(), "post-1 error") {
		t.Errorf("error should contain post-hook cause: %v", err)
	}

	// LIFO: post-2 runs first (succeeds), post-1 runs second (fails); runner never runs.
	want := []string{"pre-1", "pre-2", "pre-3", "post-2", "post-1"}
	if len(order) != len(want) {
		t.Fatalf("got %v, want %v", order, want)
	}
	for i, v := range want {
		if order[i] != v {
			t.Errorf("step %d: got %q, want %q", i, order[i], v)
		}
	}
}

// ExamplePostHook_lifoOrder demonstrates that post-hooks run in LIFO order,
// mirroring the defer stack: the last pre-hook to run is the first post-hook
// called. This ensures inner resources are released before outer ones.
func ExamplePostHook_lifoOrder() {
	makeHook := func(name string) subcmd.PreHook {
		return func(ctx context.Context) (context.Context, subcmd.PostHook, error) {
			fmt.Printf("open:  %s\n", name)
			post := func(_ context.Context) error {
				fmt.Printf("close: %s\n", name)
				return nil
			}
			return ctx, post, nil
		}
	}

	runner := func(_ context.Context, _ any, _ []string) error {
		fmt.Println("runner")
		return nil
	}

	cmd := subcmd.NewCommand("cmd", subcmd.NewFlagSet(), runner,
		subcmd.WithoutArguments(),
		subcmd.WithPreHooks(makeHook("outer"), makeHook("middle"), makeHook("inner")),
	)
	cmd.Document("lifo example")
	cs := subcmd.NewCommandSet(cmd)
	if err := cs.DispatchWithArgs(context.Background(), "tool", "cmd"); err != nil {
		panic(err)
	}
	// Output:
	// open:  outer
	// open:  middle
	// open:  inner
	// runner
	// close: inner
	// close: middle
	// close: outer
}
