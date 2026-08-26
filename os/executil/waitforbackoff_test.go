// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package executil_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"cloudeng.io/algo/ratecontrol"
	"cloudeng.io/os/executil"
)

// fakeBackoff is a ratecontrol.Backoff whose delays are instant, so that tests
// exercise WaitForBackoff's control flow rather than the clock. It hands out
// steps ready-to-receive delays and then reports exhaustion with a closed
// channel, as the Backoff contract requires. When blocked is set, Next returns
// a channel that never becomes ready, so that the context is the only way out
// of the select and cancellation tests are deterministic.
type fakeBackoff struct {
	steps   int
	blocked bool
	retries int
	done    bool
}

func (b *fakeBackoff) Next() <-chan time.Time {
	if b.blocked {
		return make(chan time.Time)
	}
	if b.retries >= b.steps {
		b.done = true
		ch := make(chan time.Time)
		close(ch)
		return ch
	}
	b.retries++
	ch := make(chan time.Time, 1)
	ch <- time.Now()
	return ch
}

func (b *fakeBackoff) Wait(context.Context, any) (bool, error) {
	return b.Done(), nil
}

func (b *fakeBackoff) Done() bool { return b.done }

func (b *fakeBackoff) Retries() int { return b.retries }

var _ ratecontrol.Backoff = (*fakeBackoff)(nil)

func TestWaitForBackoff_ImmediateDone(t *testing.T) {
	backoff := &fakeBackoff{steps: 5}
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		return true, nil
	}
	if err := executil.WaitForBackoff(context.Background(), backoff, check); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("check called %d times, want 1", calls)
	}
	// The pre-loop check succeeded, so no delay should have been consumed.
	if got := backoff.Retries(); got != 0 {
		t.Errorf("retries: got %d, want 0", got)
	}
}

func TestWaitForBackoff_ImmediateDoneWithError(t *testing.T) {
	want := errors.New("done but failed")
	check := func(_ context.Context) (bool, error) { return true, want }
	err := executil.WaitForBackoff(context.Background(), &fakeBackoff{steps: 5}, check)
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestWaitForBackoff_PollsUntilDone(t *testing.T) {
	const target = 3
	backoff := &fakeBackoff{steps: 5}
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		return calls >= target, nil
	}
	if err := executil.WaitForBackoff(context.Background(), backoff, check); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != target {
		t.Errorf("check called %d times, want %d", calls, target)
	}
	// One delay per check beyond the first.
	if got, want := backoff.Retries(), target-1; got != want {
		t.Errorf("retries: got %d, want %d", got, want)
	}
}

func TestWaitForBackoff_DoneWithErrorAfterRetries(t *testing.T) {
	want := errors.New("done but failed")
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		if calls < 3 {
			return false, nil
		}
		return true, want
	}
	err := executil.WaitForBackoff(context.Background(), &fakeBackoff{steps: 5}, check)
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestWaitForBackoff_TransientErrorContinues(t *testing.T) {
	// A (false, non-nil err) result is ignored: the done flag alone terminates
	// the loop.
	transient := errors.New("transient")
	const target = 4
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		if calls < target {
			return false, transient
		}
		return true, nil
	}
	if err := executil.WaitForBackoff(context.Background(), &fakeBackoff{steps: 10}, check); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != target {
		t.Errorf("check called %d times, want %d", calls, target)
	}
}

func TestWaitForBackoff_Exhausted(t *testing.T) {
	const steps = 2
	backoff := &fakeBackoff{steps: steps}
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		return false, nil // never done
	}
	err := executil.WaitForBackoff(context.Background(), backoff, check)
	if err == nil {
		t.Fatal("got nil error, want the backoff to report that it is done")
	}
	if !strings.Contains(err.Error(), "backoff done") {
		t.Errorf("error %q does not report that the backoff is done", err)
	}
	// The pre-loop check plus one per delay, then the closed channel ends it.
	if got, want := calls, steps+1; got != want {
		t.Errorf("check called %d times, want %d", got, want)
	}
	if !backoff.Done() {
		t.Error("backoff reports it is not done")
	}
}

func TestWaitForBackoff_ContextCancelledWhileWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// The backoff never becomes ready, so ctx.Done is the only reachable case.
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		cancel()
		return false, nil
	}
	err := executil.WaitForBackoff(ctx, &fakeBackoff{blocked: true}, check)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Errorf("check called %d times, want exactly 1 (the pre-loop call)", calls)
	}
}

func TestWaitForBackoff_ContextAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// The initial check fires even when the context is already done; the
	// subsequent select then picks ctx.Done.
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		return false, nil
	}
	err := executil.WaitForBackoff(ctx, &fakeBackoff{blocked: true}, check)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Errorf("check called %d times, want exactly 1 (the pre-loop call)", calls)
	}
}

func TestWaitForBackoff_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*tick)
	defer cancel()

	err := executil.WaitForBackoff(ctx, &fakeBackoff{blocked: true},
		func(_ context.Context) (bool, error) { return false, nil })
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want context.DeadlineExceeded", err)
	}
}

func TestWaitForBackoff_CheckReceivesContext(t *testing.T) {
	ctx := t.Context()

	var gotCtx context.Context
	check := func(c context.Context) (bool, error) {
		gotCtx = c
		return true, nil
	}
	if err := executil.WaitForBackoff(ctx, &fakeBackoff{steps: 1}, check); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotCtx != ctx {
		t.Error("check did not receive the original context")
	}
}

// TestWaitForBackoff_ExponentialBackoff exercises the real backoff rather than
// the fake, pinning the closed-channel exhaustion protocol the two agree on.
func TestWaitForBackoff_ExponentialBackoff(t *testing.T) {
	const steps = 3
	backoff := ratecontrol.NewExponentialBackoff(tick, steps)
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		return false, nil // never done, so the backoff runs to its limit
	}
	err := executil.WaitForBackoff(context.Background(), backoff, check)
	if err == nil {
		t.Fatal("got nil error, want the backoff to report that it is done")
	}
	if !strings.Contains(err.Error(), "backoff done") {
		t.Errorf("error %q does not report that the backoff is done", err)
	}
	if got, want := calls, steps+1; got != want {
		t.Errorf("check called %d times, want %d", got, want)
	}
	if !backoff.Done() {
		t.Error("backoff reports it is not done")
	}
}

// TestWaitForBackoff_NoBackoff verifies that NoBackoff polls without delay and
// without limit: its channel is ready immediately and carries a value, so the
// receive succeeds rather than reporting exhaustion the way a closed channel
// would. The loop therefore spins as fast as check returns until check is done.
func TestWaitForBackoff_NoBackoff(t *testing.T) {
	const target = 4
	var backoff ratecontrol.NoBackoff
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		return calls >= target, nil
	}
	if err := executil.WaitForBackoff(context.Background(), backoff, check); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != target {
		t.Errorf("check called %d times, want %d", calls, target)
	}
	if backoff.Done() {
		t.Error("NoBackoff reports it is done, but it has no limit to reach")
	}
}

// TestWaitForBackoff_ExponentialBackoffOffset covers the offset variant, whose
// first delay is randomised and whose subsequent delays and exhaustion are
// inherited from ExponentialBackoff.
func TestWaitForBackoff_ExponentialBackoffOffset(t *testing.T) {
	const steps = 3
	backoff := ratecontrol.NewExponentialBackoffOffset(tick, steps)
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		return false, nil // never done, so the backoff runs to its limit
	}
	err := executil.WaitForBackoff(context.Background(), backoff, check)
	if err == nil {
		t.Fatal("got nil error, want the backoff to report that it is done")
	}
	if !strings.Contains(err.Error(), "backoff done") {
		t.Errorf("error %q does not report that the backoff is done", err)
	}
	if got, want := calls, steps+1; got != want {
		t.Errorf("check called %d times, want %d", got, want)
	}
	if !backoff.Done() {
		t.Error("backoff reports it is not done")
	}
}

// TestWaitForBackoff_ExponentialBackoffSucceedsBeforeLimit verifies that a
// check which succeeds partway through leaves the backoff with retries to
// spare rather than running it to exhaustion.
func TestWaitForBackoff_ExponentialBackoffSucceedsBeforeLimit(t *testing.T) {
	backoff := ratecontrol.NewExponentialBackoff(tick, 10)
	calls := 0
	check := func(_ context.Context) (bool, error) {
		calls++
		return calls >= 3, nil
	}
	if err := executil.WaitForBackoff(context.Background(), backoff, check); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := backoff.Retries(), 2; got != want {
		t.Errorf("retries: got %d, want %d", got, want)
	}
	if backoff.Done() {
		t.Error("backoff reports it is done, but it succeeded before its limit")
	}
}
