# Package [cloudeng.io/encoding/json/jsonmsgs](https://pkg.go.dev/cloudeng.io/encoding/json/jsonmsgs?tab=doc)

```go
import cloudeng.io/encoding/json/jsonmsgs
```

Package jsonmsgs provides support for efficient encoded and decoding
arbitrary json messages over a stream, ie. an arbitrary io.Reader or
io.Writer etc. The message format is simply a 4 byte little endian length
followed by the encoded json data.

## Constants
### DefaultMaxNativeMessageSize
```go
DefaultMaxNativeMessageSize = 1024 * 1024 // 1MB


```
DefaultMaxNativeMessageSize is the default maximum size of a native message,
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
Decoder captures the state to decode a single native message. It is
created and returned by Messager.ReadMessage and must released by calling
Messager.ReleaseDecoder after which it cannot be used again.


### Type Encoder
```go
type Encoder struct {
	*jsontext.Encoder
	// contains filtered or unexported fields
}
```
Encoder captures the state to encode and send a single native message.
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
func NewMessager(wr io.Writer, rd io.ReadCloser, opts ...Option) *Messager
```
NewMessager creates a new NativeMessager with the given writer
and readCloser. If maxSize is not specified via WithMaxSize,
DefaultMaxNativeMessageSize (1MB) is used.



### Methods

```go
func (m *Messager) Close() error
```
Close closes the underlying reader of the Messager causing a pending
ReadMessage to return.


```go
func (m *Messager) NewEncoder() *Encoder
```
NewEncoder creates a new Encoder for encoding a single native message.


```go
func (m *Messager) ReadMessage() (*Decoder, error)
```
ReadMessage reads a message from the underlying reader, returning a Decoder
that can be used to decode the message. The Decoder must be released by
calling ReleaseDecoder when no longer needed. ReadMessage will block until a
complete message is read or an error occurs.


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
WriteMessage writes a message to the underlying writer with a 4-byte
little-endian length prefix. The encoder is returned to the pool after use
regardless of error.




### Type Option
```go
type Option func(*options)
```
Option represents an option for configuring a NativeMessager.

### Functions

```go
func WithDecoderOptions(opts jsontext.Options) Option
```


```go
func WithEncoderOptions(opts jsontext.Options) Option
```


```go
func WithMaxSize(maxSize uint32) Option
```
WithMaxSize sets the maximum size of a native message in bytes.







