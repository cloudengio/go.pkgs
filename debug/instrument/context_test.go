package goroutine_test

import (
	"context"
	"testing"

	"cloudeng.io/debug/goroutine"
)

func TestWithContext(t *testing.T) {
	ctx := context.Background()
	ct := goroutine.CallTraceFrom(ctx)
	if got, want := ct.ID(), int64(0); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	mt := goroutine.MessageTraceFrom(ctx)
	if got, want := mt.ID(), int64(0); got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	ctx = goroutine.WithCallTrace(ctx)
	goroutine.CallTraceFrom(ctx).Logf(1, "hello: %s", "world")
	if goroutine.CallTraceFrom(ctx).ID() == 0 {
		t.Errorf("expected non-zero ID")
	}

	ctx = goroutine.WithMessageTrace(ctx)
	goroutine.MessageTraceFrom(ctx).Log(1, goroutine.MessageSent, localAddr, remoteAddr, "sent")
	if goroutine.MessageTraceFrom(ctx).ID() == 0 {
		t.Errorf("expected non-zero ID")
	}
}
