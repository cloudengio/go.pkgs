# Package [cloudeng.io/debug/instrument](https://pkg.go.dev/cloudeng.io/debug/instrument?tab=doc)

```go
import cloudeng.io/debug/instrument
```

Package instrument provides support for instrumenting complex applications
to trace their operation and communication behaviour across multiple
goroutines and contexts.

The underlying data structure used for the trace related instrumentation
tools is a combination of a linked list of trace records where each
individual record in that linked list can itself host any number linked
lists of trace records. This mimics a linear execution which can spawn
concurrent traces from any point, that is, a linear execution which spawns
goroutines. The various uses of this underlying structure offer a 'Go'
method to be used in conjunction with goroutines to allow the traces to span
multiple goroutines.

Functions are provided to register and retrieve traces from context.Context
instances and thus to pass them through multiple API boundaries.

## Functions
### Func CopyCallTrace
```go
func CopyCallTrace(from, to context.Context) context.Context
```
CopyCallTrace will copy a call trace from one context to another.

### Func CopyMessageTrace
```go
func CopyMessageTrace(from, to context.Context) context.Context
```
CopyMessageTrace will copy a call trace from one context to another.

### Func WithCallTrace
```go
func WithCallTrace(ctx context.Context) context.Context
```
WithCallTrace returns a context.Context that is guaranteed to contain a call
trace. If the context already had a trace then it is left in place and the
same context is returned, otherwise a new context is returneed with an empty
trace.

### Func WithMessageTrace
```go
func WithMessageTrace(ctx context.Context) context.Context
```
WithMesageTrace returns a context.Context that is guaranteed to contain a
message trace. If the context already had a trace then it is left in place
and the same context is returned, otherwise a new context is returneed with
an empty trace.

### Func WriteFrames
```go
func WriteFrames(out io.Writer, prefix string, frames []runtime.Frame)
```
WriteFrames writes out the supplied []runtime.Frame a frame per line
prefixed by the supplied string.



## Types
### Type CallRecord
```go
type CallRecord struct {
	// ID is the id of the current trace, and RootID the ID of the
	// trace that created this one via a GoLog or GoLogf call.
	ID, RootID int64
	// Level is the number of GoLog or GoLogf calls that preceded
	// the creation of this record. It is used to generate relative
	// relative stack traces when printing traces.
	Level int
	// Time is the time that the record was created at.
	Time time.Time
	// GoCall is true if this record was generated by a GoLo or GoLogf
	// call.
	GoCall bool
	// Full is the full stack frame of recorded location, whereas Relative
	// is relative to the previous recorded location.
	Full, Relative []runtime.Frame
	// GoCaller is the full stack frame of the call to GoLog or GoLogf
	// that created this sub-trace.
	GoCaller []runtime.Frame
	// Arguments is either the formatted string for Logf or a slice
	// containing the arguments to Log.
	Arguments interface{}
}
```
CallRecord represents a recorded trace location.

### Methods

```go
func (cr *CallRecord) String() string
```
String implements fmt.Stringer.




### Type CallTrace
```go
type CallTrace struct {
	// contains filtered or unexported fields
}
```
CallTrace provides the ability to log specific points in a linear execution
(Log, Logf) as well as to span the creation of goroutines and points in
their execution (GoLog, GoLogf). A log record consists of the parameters to
the logging function and the location of the call (ie. caller stackframes).

### Functions

```go
func CallTraceFrom(ctx context.Context) *CallTrace
```
CallTraceFrom extracts a CallTrace from the supplied context. It returns
an empty, unused trace (i.e. its ID() method will return 0) if no trace is
found.



### Methods

```go
func (ct *CallTrace) GoLog(skip int, args ...interface{}) *CallTrace
```
GoLog logs the current call site and returns a new CallTrace, that is
a child of the existing one, to be used in a goroutine started from the
current one. Skip is the number of callers to skip, as per runtime.Callers.


```go
func (ct *CallTrace) GoLogf(skip int, format string, args ...interface{}) *CallTrace
```
GoLogf logs the current call site and returns a new CallTrace, that is
a child of the existing one, to be used in a goroutine started from the
current one. Skip is the number of callers to skip, as per runtime.Callers.


```go
func (ct *CallTrace) ID() int64
```
ID returns the id of this call trace. All traces are allocated a unique id
on first use, otherwise their id is zero.


```go
func (ct *CallTrace) Log(skip int, args ...interface{})
```
Log logs the current call site and its arguments. The supplied arguments are
stored in a slice and retained until ReleaseArguments is called. Skip is the
number of callers to skip, as per runtime.Callers.


```go
func (ct *CallTrace) Logf(skip int, format string, args ...interface{})
```
Logf logs the current call site with its arguments being immediately used
to create a string (using fmt.Sprintf) that is stored within the trace.
Skip is the number of callers to skip, as per runtime.Callers.


```go
func (ct *CallTrace) Print(out io.Writer, callers, relative bool)
```
Print will print the trace to the supplied io.Writer, if callers is set then
the stack frame will be printed and if relative is set each displayed stack
frame will be relative to the previous one for


```go
func (ct *CallTrace) ReleaseArguments()
```
ReleaseArguments releases all stored arguments from previous calls to Log or
Logf.


```go
func (ct *CallTrace) RootID() int64
```
RootID returns the root id of this call trace, ie. the id that was allocated
to the first record in this call trace hierarchy.


```go
func (ct *CallTrace) String() string
```
String implements fmt.Stringer.


```go
func (ct *CallTrace) Walk(fn func(cr CallRecord))
```
Walk traverses the call trace calling the supplied function for each record.




### Type MessagePrimitive
```go
type MessagePrimitive int
```
MessagePrimitive represents the supported message operations.

### Constants
### MessageWait, MessageAcceptWait, MessageAccepted, MessageSent, MessageReceived
```go
MessageWait MessagePrimitive = iota + 1
MessageAcceptWait
MessageAccepted
MessageSent
MessageReceived

```
The above are the defined communication primitives. They are defined in
order of preference when sorting by MergeMessageTraces.



### Methods

```go
func (m MessagePrimitive) String() string
```
String implements fmt.Stringer.




### Type MessageRecord
```go
type MessageRecord struct {
	CallRecord
	Tag           string           // Tag assigned to this message trace by Flatten.
	Status        MessagePrimitive // The status of the message.
	Local, Remote net.Addr         // The local and remote addresses for the message.
}
```
MessageRecord represents the metadata for a recorded message.

### Methods

```go
func (mr MessageRecord) String() string
```
String implements fmt.Stringer.




### Type MessageRecords
```go
type MessageRecords []MessageRecord
```

### Functions

```go
func MergeMessageTraces(traces ...MessageRecords) MessageRecords
```



### Methods

```go
func (ms MessageRecords) String() string
```




### Type MessageTrace
```go
type MessageTrace struct {
	// contains filtered or unexported fields
}
```
MessageTrace provides the ability to log various communication primitives
(e.g. message sent, received etc) and their location in a linear execution
(Log, Logf) as well as to span the creation of goroutines and the execution
of said primitives in their linear execution (GoLog, GoLogf). A log record
consists of the parameters to the logging function and the location of the
call (ie. caller stackframes).

### Functions

```go
func MessageTraceFrom(ctx context.Context) *MessageTrace
```
MessageTraceFrom extracts a MessageTrace from the supplied context.
It returns an empty, unused trace (i.e. its ID() method will return 0) if no
trace is found.



### Methods

```go
func (mt *MessageTrace) Flatten(tag string) MessageRecords
```
Flatten returns a slice of MessageRecords sorted by level, rootID, ID,
time and finally by message status (in order of Waiting, Sent and Received).


```go
func (mt *MessageTrace) GoLog(skip int, args ...interface{}) *MessageTrace
```
GoLog logs the current call site and returns a new MessageTrace, that is
a child of the existing one, to be used in a goroutine started from the
current one. Skip is the number of callers to skip, as per runtime.Callers.


```go
func (mt *MessageTrace) GoLogf(skip int, format string, args ...interface{}) *MessageTrace
```
GoLogf logs the current call site and returns a new MessageTrace, that is
a child of the existing one, to be used in a goroutine started from the
current one. Skip is the number of callers to skip, as per runtime.Callers.


```go
func (mt *MessageTrace) ID() int64
```
ID returns the id of this message trace. All traces are allocated a unique
id on first use, otherwise their id is zero.


```go
func (mt *MessageTrace) Log(skip int, status MessagePrimitive, local, remote net.Addr, args ...interface{})
```
Log logs the current call site and its arguments. The supplied arguments are
stored in a slice and retained until ReleaseArguments is called. Skip is the
number of callers to skip, as per runtime.Callers.


```go
func (mt *MessageTrace) Logf(skip int, status MessagePrimitive, local, remote net.Addr, format string, args ...interface{})
```
Logf logs the current call site with its arguments being immediately used
to create a string (using fmt.Sprintf) that is stored within the trace.
Skip is the number of callers to skip, as per runtime.Callers.


```go
func (mt *MessageTrace) Print(out io.Writer, callers, relative bool)
```
Print will print the trace to the supplied io.Writer, if callers is set then
the stack frame will be printed and if relative is set each displayed stack
frame will be relative to the previous one for


```go
func (mt *MessageTrace) ReleaseArguments()
```
ReleaseArguments releases all stored arguments from previous calls to Log or
Logf.


```go
func (mt *MessageTrace) RootID() int64
```
RootID returns the root id of this message trace, that is the id that is
allocated to the first MessageTrace record in this call trace.


```go
func (mt *MessageTrace) String() string
```
String implements fmt.Stringer.


```go
func (mt *MessageTrace) Walk(fn func(mr MessageRecord))
```
Walk traverses the call trace calling the supplied function for each record.






## Examples
### [ExampleCallTrace](https://pkg.go.dev/cloudeng.io/debug/instrument?tab=doc#example-CallTrace)

### [ExampleMessageTrace](https://pkg.go.dev/cloudeng.io/debug/instrument?tab=doc#example-MessageTrace)




