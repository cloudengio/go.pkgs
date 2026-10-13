# Package [cloudeng.io/encoding/json/jsonmsgs](https://pkg.go.dev/cloudeng.io/encoding/json/jsonmsgs?tab=doc)

```go
import cloudeng.io/encoding/json/jsonmsgs
```

Package jsonmsgs provides support for the efficient encoding and decoding
of arbitrary JSON messages over a stream. It uses the same framing as
browser native messaging since that is both minimal (4 bytes of length
+ payload) and can be used to communicate with browser extensions.
Different browsers and other environments impose different limits on message
sizes and hence support for fragmentation and reassembly is useful to avoid
the need for users of this package to vary their behaviour depending on
which environments they are running in. In addition, support is provided
for efficiently forwarding/proxying messages through intermediaries.
Support for proxies takes the form of a user supplied header that can be
decoded independently of the payload to avoid the unnecessary overhead of
decoding the entire payload. A payload can be opaque to intermediaries,
for example if it is encrypted by the sender. Note that a message that is
sent in an envelope must be valid UTF-8 and hence a sender that encrypts
must encode its ciphertext appropriately. The user header is always visible
to intermediaries.

# Reader and Writer

A Reader reads messages and a Writer writes them, each with its own options,
so that the two directions of a connection, or the two sides of a proxy,
can have different limits, in particular different frame sizes: a Reader
accepts frames of up to its maximum size, see WithMaxSize, whatever the
fragment size that the Writer at the other end of the stream uses, and a
Writer writes frames of at most its own size. Reader and Writer accept the
same Options, and ignore those that do not apply to them; the documentation
of each option says which it applies to. A Messager is a Reader and a Writer
with the same options, for convenience.

# Framing

Every frame is a 4 byte little endian length followed by exactly that many
bytes (the body), which are one complete JSON value:

    [len:4 LE][one complete JSON value]

This is the framing used by Chrome and Firefox to talk to a native messaging
host: "each message is serialized using JSON, UTF-8 encoded and is preceded
with 32-bit message length in native byte order". A message that fits in a
single frame is sent as its bare JSON, byte for byte what a browser sends
and receives, with nothing added to it: no flags, version or padding.
Browsers limit a message from the host to 1 MB, and one to the host to 64
MiB (Chrome) or 4 GB (Firefox); see DefaultMaxNativeMessageSize. Byte order
is always little endian, which is the native order of every platform that
browsers run on.

Both frames and messages must be valid JSON since a browser parses every
frame it receives as a single JSON value. Fragmentation is therefore done
inside the JSON, with the fragments of a message carried as JSON strings,
and must be UTF-8 aware to avoid splitting runes.

# Wire format

The body of a frame is one of two things, a bare message or an envelope,
and which it is is determined by its first bytes:

    frame    = length body
    length   = the number of bytes in body, as 4 bytes, little endian
    body     = bare | envelope
    bare     = a complete JSON value that does not begin with `{"~":[`
    envelope = first | later
    first    = `{"~":[0,` <total> [ `,` <hlen> ] `]` [ `,"h":` <header> ] `,"p":"` <payload> `"}`
    later    = `{"~":[` <seq> `],"p":"` <payload> `"}`

A bare message is the message and nothing else. An envelope is a JSON object
of exactly the form above, in that order, with no white space, and with up
to three members: "~", an array of numbers that describes the envelope, "h",
the user header, if there is one, and "p", the payload. The names in the
grammar are not members: they stand for the elements of the "~" array, in
order, and for the values of "h" and "p". The first envelope of a message,
which has a sequence number of 0, is the only one with a total and a user
header:

    seq      "~" element: the sequence number of the envelope within its
             message, from 0
    total    "~" element, of the first envelope only: the length in bytes of
             the whole message
    hlen     "~" element, of the first envelope only, and only if there is a
             user header: its length in bytes
    header   the value of "h", of the first envelope only, and only if there is
             a user header: a JSON value, see "User header"
    payload  the value of "p": a chunk of the message, as the contents of a
             JSON string, escaped as described under "Escaping"; it is never
             empty, and, if it is that of the first envelope and is total
             bytes long once unescaped, is the whole message

All numbers are decimal, with no sign or leading zeros, and of at most
10 digits. There is deliberately no member saying whether more envelopes
follow, no message identifier and no version: the last envelope of a message
is the one that brings the bytes received up to total, a single message is
in flight in each direction at a time, and a future version of this format
would use a key other than "~". An envelope costs 17 bytes plus the digits
of total on the first envelope of a message and 15 bytes plus the digits of
seq on the others, plus the user header and its length if the first has one,
which is of no consequence when a frame is of the order of a megabyte;
it is the encoding of the payload that determines the size of a message.

# Fragmentation

If fragmentation is enabled (see WithFragmentation) a message that does not
fit in one frame is sent in a series of envelopes, each carrying a fragment,
a chunk, of it, as in these examples, neither of which has a user header and
only the first of which has a total:

    {"~":[0,61],"p":"{\"key\":\"a value long enough to need fragmenting"}
    {"~":[1],"p":"\",\"n\":[1,2,3]}"}

The payloads of the envelopes, once unescaped and concatenated, are the
original message exactly. A message that fits in a frame is sent bare unless
it has a user header, see below, or begins with the bytes that mark an
envelope, {"~":[. Neither side ever ignores or reinterprets any other bytes.
A message that would fit into a bare frame, but coincidentally begins with
the bytes that mark an envelope, is sent in a single envelope, whether
or not fragmentation is enabled, and the reader unwraps it. Envelopes are
independent of fragmentation: fragmentation controls only whether a message
that does not fit in a frame may be split across envelopes. Without it a
message must fit in a single envelope. A reader returns ErrMessageTooLarge
for a message that declares a total larger than the maximum size,
and ErrInvalidFrame for one that is in more than one envelope but is not too
large, since it is the peer's use of fragmentation that is not accepted.

# Escaping

The payload is escaped exactly as JSON.stringify does, so that a browser
recovers it with JSON.parse and nothing more, no base64:

    "                                    \"     2 bytes
    \                                    \\     2 bytes
    backspace, form feed, newline,       \b \f \n \r \t   2 bytes
    carriage return, tab
    any other character below U+0020     \u00xx (lower case hex)   6 bytes
    everything else, including U+007F,   unchanged   1 to 4 bytes
    U+2028/9 and all non-ASCII

A message must therefore be valid UTF-8, as JSON requires, and is split only
on rune boundaries so that every payload is valid UTF-8 in its own right.
A JSON message produced by this package never contains a character of the 6
byte kind, so the payload of one is at most twice the size of the message,
and in practice a few tens of percent at most for typical JSON, whose
escapes are mostly the quotes around its strings and none at all for numbers
and text. Base64 would add a third.

# Fitting the size budget

A frame is never larger than the fragment size, see WithFragmentSize,
or than the maximum size, see WithMaxSize, if fragmentation is not enabled,
and this is enforced while the frame is built. The writer first writes
everything that precedes the payload, that is the "~" array and the user
header, if any, which are of known size since the sequence number, total and
length of the header are known, and sets aside the 2 bytes of the closing "}
to leave room for the payload. It then takes the message one rune at a time,
computes the exact length of that rune once escaped, and stops at the first
rune that would not fit, emitting a rune only if it, and so its escape
sequence, fits entirely. The frame is therefore within the budget for any
message, including one whose characters all escape to 6 bytes, and is as
full as it can be, since it ends only when the next rune does not fit.
Every envelope holds at least one rune, so the message is always consumed,
provided the fragment size is at least MinFragmentSizeWithHeader, which
NewWriter requires. Each envelope is built in a single buffer and is written
with a single call to Write. Without fragmentation the message must be in
a single envelope and ErrMessageTooLarge is returned if it does not fit,
which is known before anything is written.

# User header

Code that sits above this package may need to route a message without
decoding or reassembling it. The user header is intended to be used for
this purpose, see WriteMessageWithHeader. It is any JSON value, that is
opaque to this package, and is carried as the "h" member of an envelope,
with its length in bytes as the last element of the "~" array:

    {"~":[0,61,17],"h":{"to":7,"op":"x"},"p":"{\"key\":\"a value long enough"}
    {"~":[1],"p":"\",\"n\":[1,2,3]}"}

The header is part of the first envelope of the message, the one that
carries the first chunk of it, and not an envelope of its own, and is
on no other, since a single message is in flight in each direction and
so the first is all that is needed to route it. If fragmentation is not
enabled the message is sent in a single envelope which, with its overhead,
the user header and the escaped payload, must fit in the maximum size,
see WithMaxSize.

User headers are disabled unless WithMaxHeaderSize is set, and a header
received when they are disabled, or on any but the first envelope of a
message, is an error.

# Forwarding

A router, or proxy, that moves messages between connections does not need
to decode a message, only to look at its user header and pass it on,
and the frames that it is received in need not be those that it is sent in.
Reader.ReadFragments reads all of the frames of a message, and checks them,
into a Fragments, without reassembling or decoding the message: the
payloads of the frames are kept as they are, escaped, one after the other.
Writer.WriteFragments writes a Fragments to another stream refragmenting
if necessary, either to re-fragment them into smaller frames or merge
them into larger ones. If the frame sizes are identical the payloads are
written as they were read, without any re-escaping. All of the frames of
a message are held in memory, as they would be for ReadMessage, up to the
maximum fragmented message size. Fragments.Header returns the user header,
with which a message can be routed. A router that is the destination of a
message, and does not forward it, can decode it using Fragments.Reader,
which decodes the payloads as they are read, or Unescape for a payload.
A router that responds to a message addressed to it creates the response,
and its header, with Fragments.Set, which marshals them, and writes it with
Writer.WriteFragments as it does the messages that it forwards.

# Reading

ReadMessage returns a complete message: a fragmented message is reassembled
before it is returned, and its user header, if any, is available from
Decoder.Header.

# Errors

An error that matches ErrPermanent, tested with errors.Is, means that
the Reader or Writer that returned it can no longer be used and the
connection should be closed: all later calls return the same error.
This is the case for any error reading, other than io.EOF between messages,
since the position in the stream is then not known, and for an error writing
to the underlying writer, since a partially written frame or message cannot
be completed or undone. Any other error, such as ErrMessageTooLarge or
ErrInvalidFrame for a message to be written, is found before anything is
read or written, leaves the Reader or Writer as it was, and may be retried,
with another message for example.

An error from Writer.WriteFragments is permanent if part of the message has
been written before it is found, which cannot be completed, and otherwise,
as for WriteMessage, can be retried, perhaps by writing to a Writer that
accepts the header or has a larger frame size.

# Browser extension

The extension must reassemble fragmented messages that it receives from
the host (the limit on those is 1 MB, it is the direction in which it is
needed), and may send them, but need not, since the limit to the host is
much higher. It must also unwrap an envelope that holds a whole message,
which is how a message with a user header, or one that begins with the bytes
that mark an envelope, is sent. A reassembler in JavaScript is a few lines;
see testdata/fragments.js, which is also used to test this package against
an independent implementation:

    let parts = [], total = 0, got = 0;
    function onMessage(m) {  // m is the parsed JSON of a frame
      const h = (m !== null && typeof m === "object") ? m["~"] : undefined;
      if (!Array.isArray(h)) return m;           // an ordinary message
      if (h[0] !== parts.length) throw new Error("bad fragment");
      if (h[0] === 0) total = h[1];
      parts.push(m.p); got += new TextEncoder().encode(m.p).length;
      if (got < total) return undefined;         // more to come
      const text = parts.join(""); parts = []; got = 0;
      return JSON.parse(text);
    }

The user header, if any, is m.h of the first envelope. A sender in
JavaScript puts it in the object, between "~" and "p", and adds its length
in bytes, JSON.stringify(h) encoded as UTF-8, to the "~" array of the first
envelope.

An extension sees only the parsed JSON of a frame and so can tell an
envelope from an ordinary message only by its structure, as above,
whereas the Go reader tests for the exact bytes {"~":[. The two agree on
everything that this package writes, since a message that begins with those
bytes is always sent in an envelope, but not on an ordinary message that
has the same structure written differently, such as { "~": [1] }, which a
Go writer sends bare and an extension would take for an envelope. Messages
sent to an extension should therefore not have a top-level "~" member.
A JavaScript writer must split on code points and not UTF-16 code units so
that a surrogate pair is never divided between two fragments.

## Constants
### DefaultMaxFragmentedMessageSize
```go
DefaultMaxFragmentedMessageSize = 16 * 1024 * 1024 // 16MB


```
DefaultMaxFragmentedMessageSize is the default maximum total size of a
fragmented/reassembled message, in bytes (16MB).

### DefaultMaxNativeMessageSize
```go
DefaultMaxNativeMessageSize = 1_000_000

```
DefaultMaxNativeMessageSize is the default maximum size of a single frame,
in bytes: 1,000,000 bytes, which is below Chrome's and Firefox's limit of
1 MB on a message from a native messaging host whether that is taken to
be 10^6 or 2^20 bytes. The limit applies to the length of the frame, ie.
to the length of the message when it is not fragmented, and to the length of
each fragment, including its header and its escaped payload, when it is.

### DefaultMaxRetainedBuffer
```go
DefaultMaxRetainedBuffer = 4 * 1024 * 1024

```
DefaultMaxRetainedBuffer is the default capacity, in bytes, above which a
buffer is dropped rather than kept for reuse, see WithMaxRetainedBuffer,
unless 4 times the maximum size is larger.



## Variables
### ErrMessageTooLarge, ErrInvalidFrame, ErrPermanent, ErrClosed
```go
// ErrMessageTooLarge is returned, or wrapped by the error returned, when a
// message or frame exceeds the limits in effect.
ErrMessageTooLarge = errors.New("jsonmsgs: message too large")
// ErrInvalidFrame is returned, or wrapped by the error returned, when the
// data read is not a valid frame or sequence of fragments, or when a
// message to be written cannot be framed: it is empty, or is not valid UTF-8
// when it needs to be sent in an envelope, or its user header is empty, is
// not valid JSON or is sent when user headers are not enabled, see
// WithMaxHeaderSize.
ErrInvalidFrame = errors.New("jsonmsgs: invalid frame")
// ErrPermanent is matched, using errors.Is, by an error after which the
// Reader or Writer that returned it can no longer be used: it is returned,
// as the same error, by all later calls, and the connection should be
// closed. The error also matches the error that it wraps, such as
// ErrInvalidFrame, ErrMessageTooLarge or an error from the underlying reader
// or writer, and has the same text. An error that does not match
// ErrPermanent leaves the Reader or Writer as it was and the call may be
// retried, with a different message or Fragments, or later. io.EOF, returned
// between messages, is neither.
ErrPermanent = errors.New("jsonmsgs: permanent error")
// ErrClosed is wrapped by the error that a Writer returns, which matches
// ErrPermanent, for a write after it has been closed.
ErrClosed = errors.New("jsonmsgs: closed")

```



## Functions
### Func MinFragmentSize
```go
func MinFragmentSize(maxFragmentedMessageSize uint32) uint32
```
MinFragmentSize returns the smallest fragment size, see WithFragmentSize,
with which a message of up to maxFragmentedMessageSize bytes can always
be fragmented. It is the largest header that such a message needs, the 2
bytes that close a fragment and the longest escape sequence, so that every
fragment can carry at least one character. It is MinFragmentSizeWithHeader
with a maximum header size of 0, ie. for messages that have no user header.

### Func MinFragmentSizeWithHeader
```go
func MinFragmentSizeWithHeader(maxFragmentedMessageSize, maxHeaderSize uint32) uint32
```
MinFragmentSizeWithHeader is MinFragmentSize for a Writer that sends user
headers of up to maxHeaderSize bytes, see WithMaxHeaderSize. The largest
header a fragment needs is that of the first, which has a user header of
that size as well as the total.

### Func Unescape
```go
func Unescape(dst, payload []byte) ([]byte, error)
```
Unescape appends the bytes of the message that payload stands for to dst
and returns the extended slice. payload is the contents of a JSON string,
as the Payload of Fragments is, and is decoded as described under "Escaping"
in the package documentation, accepting every form of escape that a JSON
encoder may produce. This is for code that needs to look at the message
that Fragments carry, for example to route on it; the Payload of Fragments
that are Bare is already the message and is not escaped. An error that wraps
ErrInvalidFrame is returned if payload is not the valid contents of a JSON
string, or is not valid UTF-8 once decoded, in which case the length of dst
is unchanged. payload is not modified.



## Types
### Type Decoder
```go
type Decoder struct {
	*jsontext.Decoder
	// contains filtered or unexported fields
}
```
Decoder captures the state to decode a single message. It is created
and returned by Reader.ReadMessage and must be released by calling
Reader.ReleaseDecoder after which it cannot be used again.

### Methods

```go
func (d *Decoder) Header() jsontext.Value
```
Header returns the user header that was sent with the message,
see Writer.WriteMessageWithHeader, or nil if it was sent without one.
It is valid until the Decoder is released.




### Type Encoder
```go
type Encoder struct {
	*jsontext.Encoder
	// contains filtered or unexported fields
}
```
Encoder captures the state to encode and send a single message.
It must be obtained using Writer.NewEncoder. It will be reclaimed by
Writer.WriteMessage after which it cannot be used again.


### Type Fragments
```go
type Fragments struct {
	// contains filtered or unexported fields
}
```
Fragments are the frames that make up a single message, as read by
Reader.ReadFragments and written by Writer.WriteFragments. Fragments is
intended for use by message routers that do not need access to the body of
the message, just the metadata.

### Functions

```go
func NewFragments(header, message any) (*Fragments, error)
```
NewFragments returns Fragments for a message that is created, rather than
read, see Set.



### Methods

```go
func (f *Fragments) Bare() bool
```
Bare reports whether Payload is the message itself, and not escaped,
as it is if the message was read from a frame that was not an envelope,
or was created using Set.


```go
func (f *Fragments) Frames() int
```
Frames returns the number of frames that the message was read from.


```go
func (f *Fragments) Header() jsontext.Value
```
Header returns the user header that was sent with the message,
see Writer.WriteMessageWithHeader, or nil if it was sent without one.
It is valid until f is next used.


```go
func (f *Fragments) Len() int
```
Len returns the size in bytes of the message once the frames are reassembled
and unescaped, ie. the size that Unescape of Payload gives, or that of
Payload if the message is Bare.


```go
func (f *Fragments) Payload() []byte
```
Payload returns the payloads of the frames one after the other, which
are the contents of JSON strings that Unescape decodes to the message.
If the message is Bare it is the message, and is not escaped. It must not be
modified, and is valid until f is next used.


```go
func (f *Fragments) Reader() io.Reader
```
Reader returns an io.Reader for the message, with the escapes in the
payloads decoded as they are read, which is for a router that is the
destination of a message and not merely forwarding it, to decode the
message, using a jsontext.Decoder or the standard library's, for example,
without first assembling it in memory. Frames are decoded one at a time,
into the buffer that Read is given if it is large enough and otherwise
into one that the Reader keeps, which is no larger than a frame. A message
that is Bare is read as it is. The Reader is valid until f is next used,
and several may be in use at a time, but each by one goroutine.


```go
func (f *Fragments) Set(header, message any) error
```
Set replaces what f holds with the user header to send with a message,
if there is one, and the message, that it marshals as JSON, so that a
router, say, can respond to a message that is addressed to it using
Writer.WriteFragments, as it forwards the messages that are not,
without encoding the message itself. Each is marshalled as the standard
library's json.Marshal does, other than that the characters <, > and &
are not escaped, and so can be of any type that it can marshal, including
a jsontext.Value or json.RawMessage that holds JSON, but not a []byte,
which is marshalled as a base64 string. The header is nil if there is none.
As both are of type any, the order of header and message is not checked
by the compiler. It is an error, that wraps ErrInvalidFrame and the error
from marshalling, if either cannot be marshalled, or if the message is nil.
If there is an error f is empty. The message is not checked against the
limits of a Writer, which are those of WriteFragments. f is reused,
like the Fragments that are passed to ReadFragments, and can be written,
as can any other, to any number of Writers.




### Type Messager
```go
type Messager struct {
	*Reader
	*Writer
	// contains filtered or unexported fields
}
```
Messager is a Reader and a Writer, for convenience when the same options
apply to both directions of a connection: NewMessager(rd, wr, opts...)
is NewReader(rd, opts...) and NewWriter(wr, opts...), and its methods are
theirs. Use a Reader and a Writer when each direction needs its own options,
for example a proxy that reads frames of one size and writes those of
another.

### Functions

```go
func NewMessager(rd io.ReadCloser, wr io.Writer, opts ...Option) *Messager
```
NewMessager creates a Messager that reads from rd and writes to wr,
see NewReader and NewWriter.



### Methods

```go
func (m *Messager) Close() error
```
Close closes the Reader and the Writer, see their Close methods, and returns
the errors from both. If they read and write the same object, such as a
net.Conn, it is closed once.




### Type Option
```go
type Option func(*options)
```
Option represents an option for configuring a Reader or a Writer, which
ignore the options that do not apply to them: a Reader uses WithMaxSize,
WithMaxFragmentedMessageSize, WithFragmentation, WithMaxHeaderSize,
WithMaxRetainedBuffer and WithDecoderOptions; a Writer uses WithMaxSize,
WithFragmentSize, WithMaxFragmentedMessageSize, WithFragmentation,
WithMaxHeaderSize, WithMaxRetainedBuffer, WithWriteTimeout and
WithEncoderOptions.

### Functions

```go
func WithDecoderOptions(opts jsontext.Options) Option
```
WithDecoderOptions sets the options for the decoders returned by
Reader.ReadMessage.


```go
func WithEncoderOptions(opts jsontext.Options) Option
```
WithEncoderOptions sets the options for the encoders returned by
Writer.NewEncoder.


```go
func WithFragmentSize(fragmentSize uint32) Option
```
WithFragmentSize sets the maximum size in bytes of a frame written by a
Writer when fragmentation is enabled, including a message sent whole: a
message that does not fit is fragmented into frames of at most this size. It
defaults to the maximum size, and is set to it if it is 0 or larger than it.
It is ignored if fragmentation is not enabled, when the maximum size is the
limit, and by a Reader, which accepts frames of any size up to its maximum.
If fragmentation is enabled, NewWriter panics if fragmentSize is less than
MinFragmentSizeWithHeader for the maximum fragmented message size and header
size.


```go
func WithFragmentation(enable bool) Option
```
WithFragmentation controls whether automatic message fragmentation
is enabled. The default is false. When false, a Writer returns
ErrMessageTooLarge if a message exceeds the maximum size, and a Reader
returns an error if a message in more than one envelope is received. When
true, messages exceeding the fragment size (which defaults to the maximum
size) are automatically fragmented into multiple frames. The Reader and the
Writer at the two ends of a stream must both have it enabled for fragmented
messages to be exchanged, but a message that is not fragmented is the same
either way. It does not affect the use of envelopes, which carry a user
header, see WithMaxHeaderSize, or a message that begins with the bytes that
mark one: when disabled such a message must fit in a single envelope.


```go
func WithMaxFragmentedMessageSize(maxSize uint32) Option
```
WithMaxFragmentedMessageSize sets the maximum total size of a message
that can be fragmented in bytes, which is to say the largest that
can be written and that will be accepted when reassembled. If 0,
DefaultMaxFragmentedMessageSize (16MB) is used. It applies only if
fragmentation is enabled, otherwise a message is limited by the maximum
size, see WithMaxSize. A Writer returns ErrMessageTooLarge for a message
that exceeds it and a Reader for one that declares a total that does, before
reading any of it. It protects receivers from unbounded memory growth,
and prevents deadlocks when writing over buffered channels that could fill
up before a complete message is sent. NewReader and NewWriter panic if it
exceeds math.MaxInt32.


```go
func WithMaxHeaderSize(maxHeaderSize uint32) Option
```
WithMaxHeaderSize enables user headers, see WriteMessageWithHeader, of up
to maxHeaderSize bytes, which is also the largest accepted by a Reader:
a frame with a header is an error unless this option is set. It does not
require fragmentation, in which case a message with a header must fit in a
single envelope, and if fragmentation is enabled it increases the smallest
fragment size that NewWriter accepts, see MinFragmentSizeWithHeader.
NewReader and NewWriter panic if maxHeaderSize exceeds math.MaxInt32.


```go
func WithMaxRetainedBuffer(maxRetained uint32) Option
```
WithMaxRetainedBuffer sets the capacity, in bytes, above which a buffer is
dropped, rather than kept for reuse, when it has been used for a message:
the buffers of the Decoders, Encoders and Fragments that are released or
reused, and those that a Reader or Writer keeps for frames. It stops one
very large message from pinning that memory for the life of the Reader or
Writer, at the cost of allocating the buffer again for the next. If 0,
the default, it is DefaultMaxRetainedBuffer or, if it is larger, 4 times the
maximum size, see WithMaxSize, so that a Reader or Writer for large frames
does not drop the buffer for each frame. A message larger than the limit is
allocated for, as it is read or written, in any case.


```go
func WithMaxSize(maxSize uint32) Option
```
WithMaxSize sets the maximum size of a single frame in bytes: no frame
larger is accepted by a Reader, which rejects it on seeing its length, and,
if fragmentation is not enabled, no frame larger is written by a Writer,
so that a message that is sent in an envelope, because it has a user header
for example, must be smaller by the size of the envelope. If maxSize is 0,
DefaultMaxNativeMessageSize is used. A Reader's should be at least as large
as the largest frame a peer will send; the limit of a browser on frames to
the host is much larger than that on frames from it. NewReader and NewWriter
panic if maxSize is less than 2 or exceeds math.MaxInt32-4.


```go
func WithWriteTimeout(timeout time.Duration) Option
```
WithWriteTimeout sets the time that a Writer allows for each of the
frames of a message to be written, if the io.Writer that it writes to is a
net.Conn, or anything else that has a SetWriteDeadline method, as a net.Conn
does: before a frame is written the write deadline of the connection is set
to timeout from then, and afterwards cleared. A write that does not complete
in that time fails, with an error that wraps os.ErrDeadlineExceeded and is
a net.Error that is a timeout, which is permanent, since part of the frame
may have been written. A message of many frames may take longer than the
timeout. It is ignored if the io.Writer is something else, or if timeout is
not greater than 0, the default, when writes have no limit but any that the
connection has.




### Type Reader
```go
type Reader struct {
	// contains filtered or unexported fields
}
```
Reader reads the messages written by a Writer, or by a browser, from a
stream, see the package documentation, and Option for the options that apply
to it. A Reader is safe for concurrent use: a message is read whole by any
one call and calls are serialized.

### Functions

```go
func NewReader(rd io.ReadCloser, opts ...Option) *Reader
```
NewReader creates a new Reader that reads from rd, which may be nil for
a stream that is empty. It panics if the options are invalid, see the
individual options.



### Methods

```go
func (r *Reader) Close() error
```
Close closes the underlying reader of the Reader causing a pending
ReadMessage to return. It returns the error from closing the underlying
reader, if any, and closes it only once, however many times it is called,
from any number of goroutines.


```go
func (r *Reader) ReadFragments(into *Fragments) (*Fragments, error)
```
ReadFragments reads all of the frames of the next message, and checks them,
without reassembling or decoding the message, so that it can be routed using
its Header and forwarded using Writer.WriteFragments, see "Forwarding" in
the package documentation. They are read into into, which is reused, and
returned, or into a new Fragments if it is nil. Whatever it held is lost,
and what is returned is valid until it is next used. The memory for the
message is allocated when its first frame has been read, from the size
that it declares, which cannot exceed the maximum fragmented message size.
It returns io.EOF if the stream ends between messages, and otherwise the
errors that ReadMessage does.


```go
func (r *Reader) ReadMessage() (*Decoder, error)
```
ReadMessage reads the next message, reassembling it if it was fragmented,
and returns a Decoder for it, which must be released using ReleaseDecoder
when no longer needed. The Decoder's Header returns the user header, if any.
It blocks until a complete message is read or an error occurs, and returns
io.EOF if the stream ends between messages. See "Errors" in the package
documentation for the errors that can be retried.


```go
func (r *Reader) ReleaseDecoder(dec *Decoder)
```
ReleaseDecoder returns a Decoder obtained from ReadMessage, which cannot be
used again. A nil Decoder, or one that has no buffer, is ignored.




### Type Writer
```go
type Writer struct {
	// contains filtered or unexported fields
}
```
Writer writes messages to a stream, fragmenting them if need be, see the
package documentation, and Option for the options that apply to it. A Writer
is safe for concurrent use: a message is written whole by any one call and
calls are serialized. Close stops it writing, and closes the io.Writer if it
is an io.Closer.

### Functions

```go
func NewWriter(wr io.Writer, opts ...Option) *Writer
```
NewWriter creates a new Writer that writes to wr, which may be nil to
discard what is written. It panics if the options are invalid, see the
individual options.



### Methods

```go
func (w *Writer) Close() error
```
Close closes the underlying writer if it is an io.Closer, such as a
net.Conn, which causes a write that is blocked on it to return, with an
error, and stops the Writer from writing: all later writes return an error
that wraps ErrClosed and matches ErrPermanent, as does a message that is
part way through being written, which is then not completed. Close does not
wait for a write that is in progress. It returns the error from closing the
underlying writer, if any, and closes it only once, however many times it is
called, from any number of goroutines. An underlying writer that is not an
io.Closer is left as it is.


```go
func (w *Writer) NewEncoder() *Encoder
```
NewEncoder creates a new Encoder for encoding a single message.


```go
func (w *Writer) ReleaseEncoder(enc *Encoder)
```
ReleaseEncoder should only be called if WriteMessage will not be called,
for example if there is an error during encoding that will cause the message
to be discarded.


```go
func (w *Writer) WriteFragments(f *Fragments) error
```
WriteFragments writes the message in f, as read by Reader.ReadFragments,
to the underlying writer, see "Forwarding" in the package documentation.
It is not checked again, other than that its user header is acceptable to
this Writer, since it was checked when it was read. It is written in frames
of at most the size of this Writer's, as WriteMessage would: the frames
that it was read from are used as they are, other than that they are put
together, to send fewer frames, if they are smaller than this Writer's,
and that any that are larger are split. A message that was not Bare and is
written with fragmentation not enabled must fit in a single frame. The user
header is sent with the first frame if there is one. f is not modified,
and may be written again. See "Errors" in the package documentation for the
errors that can be retried.


```go
func (w *Writer) WriteMessage(enc *Encoder) error
```
WriteMessage writes the message encoded by enc, as a bare frame or in
envelopes as described in the package documentation, each frame being
written with a single call to Write. The newline that the encoder appends to
a message is not written. enc is returned to the pool whatever the outcome
and must not be used again. See "Errors" in the package documentation for
the errors that can be retried.


```go
func (w *Writer) WriteMessageWithHeader(enc *Encoder, header jsontext.Value) error
```
WriteMessageWithHeader is like WriteMessage but sends header with the
message, see "User header" in the package documentation. The header must be
a valid JSON value of at most the size set by WithMaxHeaderSize, which is
checked.







