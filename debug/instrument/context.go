package instrument

import (
	"context"
)

type roleKeyType string

var roleKey roleKeyType

type callTraceKeyType struct{}

var callTraceKey callTraceKeyType

type messageTraceKeyType struct{}

var messageTraceKey messageTraceKeyType

type ContextValue interface {
	Value(interface{}) interface{}
}

// CallTraceFrom extracts a CallTrace from the supplied context. It returns an
// empty, unused trace (i.e. its ID() method will return 0) if no trace
// is found.
func CallTraceFrom(ctx ContextValue) *CallTrace {
	if val := ctx.Value(callTraceKey); val != nil {
		if v, ok := val.(*CallTrace); ok {
			return v
		}
	}
	return &CallTrace{} // id will be zero.
}

// MessageTraceFrom extracts a MessageTrace from the supplied context. It
// returns an empty, unused trace (i.e. its ID() method will return 0) if no
// trace is found.
func MessageTraceFrom(ctx ContextValue) *MessageTrace {
	if val := ctx.Value(messageTraceKey); val != nil {
		if v, ok := val.(*MessageTrace); ok {
			return v
		}
	}
	return &MessageTrace{} // id will be zero.
}

// WithCallTrace returns a context.Context that is guaranted to contain
// a call trace. If the context already had a trace then it is left in place
// and the same context is returned, otherwise a new context is returneed
// with an empty trace.
func WithCallTrace(ctx context.Context) context.Context {
	ct := CallTraceFrom(ctx)
	if ct.ID() == 0 {
		return context.WithValue(ctx, callTraceKey, ct)
	}
	return ctx
}

// WithMesageTrace returns a context.Context that is guaranted to contain
// a message trace. If the context already had a trace then it is left in place
// and the same context is returned, otherwise a new context is returneed
// with an empty trace.
func WithMessageTrace(ctx context.Context) context.Context {
	ct := MessageTraceFrom(ctx)
	if ct.ID() == 0 {
		return context.WithValue(ctx, messageTraceKey, ct)
	}
	return ctx
}

// CopyCallTrace will copy a call trace from one context to another.
func CopyCallTrace(from, to context.Context) context.Context {
	if val := from.Value(callTraceKey); val != nil {
		if v, ok := val.(*CallTrace); ok {
			return context.WithValue(to, callTraceKey, v)
		}
	}
	return from
}

// CopyMessageTrace will copy a call trace from one context to another.
func CopyMessageTrace(from, to context.Context) context.Context {
	if val := from.Value(messageTraceKey); val != nil {
		if v, ok := val.(*MessageTrace); ok {
			return context.WithValue(to, messageTraceKey, v)
		}
	}
	return from
}
