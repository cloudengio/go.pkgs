// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package chromedputil

import (
	"context"
	"fmt"
	"log/slog"

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
		select {
		case ch <- event:
		case <-ctx.Done():
		}
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
		select {
		case ch <- event:
		case <-ctx.Done():
		}
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
		select {
		case ch <- event:
		case <-ctx.Done():
		}
		return true
	}
}

// NewAnyHandler returns a handler for all/any events that forwards
// the event to the provided channel. It should generally be the last
// handler in the list passed to Listen.
func NewAnyHandler(ch chan<- any) func(ctx context.Context, ev any) bool {
	return func(ctx context.Context, ev any) bool {
		select {
		case ch <- ev:
		case <-ctx.Done():
		}
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

type loggingOptions struct {
	consoleCh   chan *runtime.EventConsoleAPICalled
	exceptionCh chan *runtime.EventExceptionThrown
	eventCh     chan *log.EventEntryAdded
	anyCh       chan any
	handlers    []func(ctx context.Context, ev any) bool
}

type LoggingOption func(*loggingOptions)

func WithConsoleLogging() LoggingOption {
	return func(opts *loggingOptions) {
		opts.consoleCh = make(chan *runtime.EventConsoleAPICalled, 10)
		opts.handlers = append(opts.handlers, NewEventConsoleHandler(opts.consoleCh))
	}
}

func WithExceptionLogging() LoggingOption {
	return func(opts *loggingOptions) {
		opts.exceptionCh = make(chan *runtime.EventExceptionThrown, 10)
		opts.handlers = append(opts.handlers, NewLogExceptionHandler(opts.exceptionCh))
	}
}

func WithEventEntryLogging() LoggingOption {
	return func(opts *loggingOptions) {
		opts.eventCh = make(chan *log.EventEntryAdded, 10)
		opts.handlers = append(opts.handlers, NewEventEntryHandler(opts.eventCh))
	}
}

func WithAnyEventLogging() LoggingOption {
	return func(opts *loggingOptions) {
		opts.anyCh = make(chan any, 10)
		opts.handlers = append(opts.handlers, NewAnyHandler(opts.anyCh))
	}
}

// RunLoggingListener starts the logging listener for Chrome DevTools Protocol events.
func RunLoggingListener(ctx context.Context, logger *slog.Logger, opts ...LoggingOption) {
	var options loggingOptions
	for _, opt := range opts {
		opt(&options)
	}
	if options.consoleCh == nil {
		options.consoleCh = make(chan *runtime.EventConsoleAPICalled, 10)
	}
	if options.exceptionCh == nil {
		options.exceptionCh = make(chan *runtime.EventExceptionThrown, 10)
	}
	if options.eventCh == nil {
		options.eventCh = make(chan *log.EventEntryAdded, 10)
	}
	if options.anyCh == nil {
		options.anyCh = make(chan any, 10)
	}

	Listen(ctx, options.handlers...)

	ch := make(chan struct{})
	go func() {
		close(ch)
		for {
			select {
			case event := <-options.consoleCh:
				s, err := ConsoleArgsAsJSON(ctx, event)
				if err != nil {
					logger.Error("Failed to marshal console args to JSON", "error", err)
					continue
				}
				attrs := []slog.Attr{}
				for i, arg := range s {
					attrs = append(attrs, slog.Attr{Key: fmt.Sprintf("%03d", i), Value: slog.StringValue(string(arg))})
				}
				logger.LogAttrs(ctx, slog.LevelInfo, "Console API called", attrs...)
			case event := <-options.exceptionCh:
				logger.Error("Exception thrown", slog.Any("event", event))
				logger.Error("Exception details",
					slog.Any("stackTrace", event.ExceptionDetails.StackTrace),
					slog.Any("exception", event.ExceptionDetails.Exception),
					slog.Any("lineNumber", event.ExceptionDetails.LineNumber),
					slog.Any("columnNumber", event.ExceptionDetails.ColumnNumber),
				)
			case event := <-options.eventCh:
				logger.Info("Log entry added", slog.Any("event", event.Entry))
			case event := <-options.anyCh:
				logger.Info("Other event", slog.Any("event", event))
			case <-ctx.Done():
				return
			}
		}
	}()
	<-ch
}
