# Package [cloudeng.io/encoding/json/jsonmsgs](https://pkg.go.dev/cloudeng.io/encoding/json/jsonmsgs?tab=doc)

```go
import cloudeng.io/encoding/json/jsonmsgs
```

Package jsonmsgs provides support for efficient encoding and decoding
arbitrary json messages over a stream, ie. an arbitrary io.Reader or
io.Writer etc. The message format is simply a 4 byte little endian length
followed by the encoded json data.

## Constants
### FlagFragment, FlagMore, LengthMask
```go
// FlagFragment indicates that the frame is a fragment of a larger message.
// It occupies bit 31 of the 4-byte frame header.
FlagFragment uint32 = 1 << 31
// FlagMore indicates that more fragments follow for the current message.
// It occupies bit 30 of the 4-byte frame header.
FlagMore uint32 = 1 << 30
// LengthMask masks the 30-bit frame payload length (bits 0..29).
LengthMask uint32 = 0x3fffffff

```

### DefaultMaxFragmentedMessageSize
```go
DefaultMaxFragmentedMessageSize = 16 * 1024 * 1024 // 16MB


```
DefaultMaxFragmentedMessageSize is the default maximum total size of a
fragmented/reassembled message, in bytes (16MB).

### DefaultMaxNativeMessageSize
```go
DefaultMaxNativeMessageSize = 1024 * 1024 // 1MB


```
DefaultMaxNativeMessageSize is the default maximum size of a single frame,
in bytes.



## Variables
### ErrMessageTooLarge
```go
ErrMessageTooLarge = errors.New("jsonmsgs: message too large")

```



## Types
### Type Decoder
```go
type Decoder struct {
	*jsontext.Decoder
	// contains filtered or unexported fields
}
```
Decoder captures the state to decode a single message. It is created
and returned by Messager.ReadMessage and must released by calling
Messager.ReleaseDecoder after which it cannot be used again.


### Type Encoder
```go
type Encoder struct {
	*jsontext.Encoder
	// contains filtered or unexported fields
}
```
Encoder captures the state to encode and send a single message.
It must be obtained using Messager.NewEncoder. It will be reclaimed by
Messager.WriteMessage after which it cannot be used again.


### Type Messager
```go
type Messager struct {
	// contains filtered or unexported fields
}
```

### Functions

```go
func NewMessager(rd io.ReadCloser, wr io.Writer, opts ...Option) *Messager
```
NewMessager creates a new Messager with the given readCloser and writer.
If maxSize is not specified via WithMaxSize, DefaultMaxNativeMessageSize
(1MB) is used. If fragmentSize is not specified via WithFragmentSize,
it defaults to maxSize. NewMessager panics if maxSize exceeds LengthMask
(see WithMaxSize).



### Methods

```go
func (m *Messager) Close() error
```
Close closes the underlying reader of the Messager causing a pending
ReadMessage to return.


```go
func (m *Messager) NewEncoder() *Encoder
```
NewEncoder creates a new Encoder for encoding a single message.


```go
func (m *Messager) ReadMessage() (*Decoder, error)
```
ReadMessage reads a message from the underlying reader, returning a Decoder
that can be used to decode the message. If the message was fragmented, all
fragments are read and reassembled into a single Decoder. The Decoder must
be released by calling ReleaseDecoder when no longer needed. ReadMessage
will block until a complete message is read or an error occurs.


```go
func (m *Messager) ReleaseDecoder(dec *Decoder)
```


```go
func (m *Messager) ReleaseEncoder(enc *Encoder)
```
ReleaseEncoder should only be called if WriteMessage will not be called,
for example if there is an error during encoding that will cause the message
to be discarded.


```go
func (m *Messager) WriteMessage(enc *Encoder) error
```
WriteMessage writes a message to the underlying writer with a
4-byte little-endian length prefix. If fragmentation is enabled (via
WithFragmentation) and the serialized message exceeds the configured
fragment size (which defaults to MaxSize), it is transparently fragmented
into multiple frames of at most fragment size bytes. If fragmentation is
disabled (the default) and the message exceeds MaxSize, ErrMessageTooLarge
is returned. The encoder is returned to the pool after use regardless of
error.




### Type Option
```go
type Option func(*options)
```
Option represents an option for configuring a Messager.

### Functions

```go
func WithDecoderOptions(opts jsontext.Options) Option
```


```go
func WithEncoderOptions(opts jsontext.Options) Option
```


```go
func WithFragmentSize(fragmentSize uint32) Option
```
WithFragmentSize sets the maximum size of a fragment frame payload in bytes.
It defaults to MaxSize, but can be configured to be smaller. If fragmentSize
is 0 or exceeds MaxSize, it is set to MaxSize.


```go
func WithFragmentation(enable bool) Option
```
WithFragmentation controls whether automatic message fragmentation
is enabled. The default is false. When false, WriteMessage returns
ErrMessageTooLarge if a message exceeds MaxSize, and ReadMessage returns
an error if a fragment frame is received. When true, messages exceeding the
fragment size (which defaults to MaxSize) are automatically fragmented into
multiple frames.


```go
func WithMaxFragmentedMessageSize(maxSize uint32) Option
```
WithMaxFragmentedMessageSize sets the maximum total size of a message that
can be fragmented in bytes. If 0, DefaultMaxFragmentedMessageSize (16MB) is
used. If set (> 0), WriteMessage returns ErrMessageTooLarge if a message
to be fragmented exceeds this size, and ReadMessage returns an error if
the total reassembled size exceeds this limit. This prevents deadlocks when
writing over buffered channels that could fill up before a complete request
is sent, as well as protecting receivers from unbounded memory growth.


```go
func WithMaxSize(maxSize uint32) Option
```
WithMaxSize sets the maximum size of a single frame in bytes. maxSize must
not exceed LengthMask (~1GiB); NewMessager panics otherwise, since bits 30
and 31 of the frame header are reserved for FlagMore/FlagFragment and cannot
represent a larger single-frame length.







