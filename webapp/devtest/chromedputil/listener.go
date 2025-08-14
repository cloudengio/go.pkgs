// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package chromedputil

import (
	"context"

	"github.com/chromedp/cdproto/log"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/go-json-experiment/json"
)

// Listen sets up a listener for Chrome DevTools Protocol events and calls each
// of the supplied handlers in turn when an event is received. The first handler
// to return true stops the event propagation.
func Listen(ctx context.Context, handlers ...func(ctx context.Context, ev any) bool) {
	chromedp.ListenTarget(ctx, func(ev any) {
		for _, handler := range handlers {
			if handler(ctx, ev) {
				break
			}
		}
	})
}

// NewLogExceptionHandler returns a handler for log exceptions that forwards
// the event to the provided channel.
func NewLogExceptionHandler(ch chan<- *runtime.EventExceptionThrown) func(ctx context.Context, ev any) bool {
	return func(ctx context.Context, ev any) bool {
		event, ok := ev.(*runtime.EventExceptionThrown)
		if !ok {
			return false
		}
		ch <- event
		return true
	}
}

// NewEventEntryHandler returns a handler for log entry events that forwards
// the event to the provided channel.
func NewEventEntryHandler(ch chan<- *log.EventEntryAdded) func(ctx context.Context, ev any) bool {
	return func(ctx context.Context, ev any) bool {
		event, ok := ev.(*log.EventEntryAdded)
		if !ok {
			return false
		}
		ch <- event
		return true
	}
}

// NewEventConsoleHandler returns a handler for console events that forwards
// the event to the provided channel.
func NewEventConsoleHandler(ch chan<- *runtime.EventConsoleAPICalled) func(ctx context.Context, ev any) bool {
	return func(ctx context.Context, ev any) bool {
		event, ok := ev.(*runtime.EventConsoleAPICalled)
		if !ok {
			return false
		}
		ch <- event
		return true
	}
}

// NewAnyHandler returns a handler for all/any events that forwards
// the event to the provided channel. It should generally be the last
// handler in the list passed to Listen.
func NewAnyHandler(ch chan<- any) func(ctx context.Context, ev any) bool {
	return func(ctx context.Context, ev any) bool {
		ch <- ev
		return true
	}
}

// ConsoleArgsAsJSON converts the console API call arguments to a slice of marshalled
// JSON data, one per each argument to the original console.log call.
func ConsoleArgsAsJSON(ctx context.Context, event *runtime.EventConsoleAPICalled) ([][]byte, error) {
	values := make([][]byte, 0, len(event.Args))
	for _, arg := range event.Args {
		val, _, err := GetRemoteObjectValueJSON(ctx, arg)
		if err != nil {
			return nil, err
		}
		out, err := json.Marshal(val)
		if err != nil {
			return nil, err
		}
		values = append(values, out)
	}
	return values, nil
}
