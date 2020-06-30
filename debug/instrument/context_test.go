// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package instrument_test

import (
	"context"
	"testing"

	"cloudeng.io/debug/instrument"
)

func TestCallTraceContext(t *testing.T) {
	ctx := context.Background()
	ct := instrument.CallTraceFrom(ctx)
	if got, want := ct.ID(), int64(0); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	ctx = instrument.WithCallTrace(ctx)
	ct = instrument.CallTraceFrom(ctx)
	instrument.CallTraceFrom(ctx).Logf(1, "hello: %s", "world")
	if instrument.CallTraceFrom(ctx).ID() == 0 {
		t.Errorf("expected non-zero ID")
	}
	if got, want := instrument.WithCallTrace(ctx), ctx; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	rid, id := ct.RootID(), ct.ID()
	nctx := context.Background()
	if got, want := instrument.CallTraceFrom(nctx).ID(), int64(0); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	nctx = instrument.CopyCallTrace(ctx, nctx)
	nct := instrument.CallTraceFrom(nctx)
	if got, want := nct.ID(), id; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := nct.RootID(), rid; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	octx := context.Background()
	if got, want := instrument.CopyCallTrace(octx, nctx), octx; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMessageTraceContext(t *testing.T) {
	ctx := context.Background()
	mt := instrument.MessageTraceFrom(ctx)
	if got, want := mt.ID(), int64(0); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	ctx = instrument.WithMessageTrace(ctx)
	mt = instrument.MessageTraceFrom(ctx)
	instrument.MessageTraceFrom(ctx).Log(1, instrument.MessageSent, localAddr, remoteAddr, "sent")
	if instrument.MessageTraceFrom(ctx).ID() == 0 {
		t.Errorf("expected non-zero ID")
	}
	if got, want := instrument.WithMessageTrace(ctx), ctx; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	rid, id := mt.RootID(), mt.ID()
	nctx := context.Background()
	if got, want := instrument.MessageTraceFrom(nctx).ID(), int64(0); got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	nctx = instrument.CopyMessageTrace(ctx, nctx)
	nmt := instrument.MessageTraceFrom(nctx)
	if got, want := nmt.ID(), id; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
	if got, want := nmt.RootID(), rid; got != want {
		t.Errorf("got %v, want %v", got, want)
	}

	octx := context.Background()
	if got, want := instrument.CopyMessageTrace(octx, nctx), octx; got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
