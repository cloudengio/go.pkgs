// Copyright 2025 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package chromedputil

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/chromedp/cdproto/log"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"
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

// NewListenHandler returns a handler for a specific event type that forwards
// the event to the provided channel.
func NewListenHandler[T any](ctx context.Context, ch chan<- T) func(ctx context.Context, ev any) bool {
	return func(ctx context.Context, ev any) bool {
		event, ok := ev.(T)
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

/*
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
*/

// ConsoleArgsAsJSON converts the console API call arguments to a slice of marshalled
// JSON data, one per each argument to the original console.log call.
func ConsoleArgsAsJSON(ctx context.Context, event *runtime.EventConsoleAPICalled) ([][]byte, error) {
	values := make([][]byte, 0, len(event.Args))
	for _, arg := range event.Args {
		val, _, err := GetRemoteObjectValueJSON(ctx, arg)
		if err != nil {
			return nil, fmt.Errorf("failed to get remote object value: %+v %w", arg, err)
		}
		out, err := json.Marshal(val)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		values = append(values, out)
	}
	return values, nil
}

type loggingOptions struct {
	consoleCh         chan *runtime.EventConsoleAPICalled
	exceptionCh       chan *runtime.EventExceptionThrown
	networkResponseCh chan *network.EventResponseReceived
	networkRrequestCh chan *network.EventRequestWillBeSent
	eventCh           chan *log.EventEntryAdded
	anyCh             chan any
	handlers          []func(ctx context.Context, ev any) bool
}

// LoggingOption represents options to RunLoggingListener.
type LoggingOption func(*loggingOptions)

func handlerOption[T any](ctx context.Context,
	handlers []func(ctx context.Context, ev any) bool) (chan T, []func(ctx context.Context, ev any) bool) {
	ch := make(chan T, 10)
	return ch, append(handlers, NewListenHandler(ctx, ch))
}

func initCh[T any](ch chan T) chan T {
	if ch == nil {
		return make(chan T, 10)
	}
	return ch
}

// WithConsoleLogging enables logging of events of type 'runtime.EventConsoleAPICalled'.
func WithConsoleLogging(ctx context.Context) LoggingOption {
	return func(opts *loggingOptions) {
		opts.consoleCh, opts.handlers = handlerOption[*runtime.EventConsoleAPICalled](ctx, opts.handlers)
	}
}

// WithExceptionLogging enables logging of events of type 'runtime.EventExceptionThrown'.
func WithExceptionLogging(ctx context.Context) LoggingOption {
	return func(opts *loggingOptions) {
		opts.exceptionCh, opts.handlers = handlerOption[*runtime.EventExceptionThrown](ctx, opts.handlers)
	}
}

// WithEventEntryLogging enables logging of events of type 'log.EventEntryAdded'.
func WithEventEntryLogging(ctx context.Context) LoggingOption {
	return func(opts *loggingOptions) {
		opts.eventCh, opts.handlers = handlerOption[*log.EventEntryAdded](ctx, opts.handlers)
	}
}

// WithNetworkLogging enables logging of events of type 'network.EventResponseReceived'.
func WithNetworkLogging(ctx context.Context) LoggingOption {
	return func(opts *loggingOptions) {
		opts.networkResponseCh, opts.handlers = handlerOption[*network.EventResponseReceived](ctx, opts.handlers)
		opts.networkRrequestCh, opts.handlers = handlerOption[*network.EventRequestWillBeSent](ctx, opts.handlers)
	}
}

// WithAnyEventLogging enables logging for events of type 'any'.
// This is a catch all and should generally be the last handler in the list.
func WithAnyEventLogging(ctx context.Context) LoggingOption {
	return func(opts *loggingOptions) {
		opts.anyCh, opts.handlers = handlerOption[any](ctx, opts.handlers)
	}
}

type jsonValue struct {
	Value jsontext.Value `json:"value"`
}

// RunLoggingListener starts the logging listener for Chrome DevTools Protocol events.
// It returns a channel that is closed when the goroutine that listens on events
// terminates.
func RunLoggingListener(ctx context.Context, logger *slog.Logger, opts ...LoggingOption) chan struct{} {
	var options loggingOptions
	for _, opt := range opts {
		opt(&options)
	}
	options.consoleCh = initCh(options.consoleCh)
	options.exceptionCh = initCh(options.exceptionCh)
	options.networkResponseCh = initCh(options.networkResponseCh)
	options.networkRrequestCh = initCh(options.networkRrequestCh)
	options.eventCh = initCh(options.eventCh)
	options.anyCh = initCh(options.anyCh)

	Listen(ctx, options.handlers...)

	doneCh := make(chan struct{})
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
					var jv jsonValue
					if err := json.Unmarshal([]byte(arg), &jv); err != nil {
						attrs = append(attrs, slog.Attr{Key: fmt.Sprintf("%03d", i), Value: slog.StringValue(string(arg))})
					} else {
						attrs = append(attrs, slog.Attr{Key: fmt.Sprintf("%03d", i), Value: slog.AnyValue(jv.Value)})
					}
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
			case response := <-options.networkResponseCh:
				logger.Info("Network response received", "url", response.Response.URL, "status", response.Response.Status)
			case request := <-options.networkRrequestCh:
				logger.Info("Network request received", "url", request.Request.URL, "method", request.Request.Method)
			case <-ctx.Done():
				close(doneCh)
				return
			}
		}
	}()
	<-ch
	return doneCh
}
