# Package [cloudeng.io/encoding/json/jsonmsgs](https://pkg.go.dev/cloudeng.io/encoding/json/jsonmsgs?tab=doc)

```go
import cloudeng.io/encoding/json/jsonmsgs
```

Package jsonmsgs provides support for efficient encoding and decoding
arbitrary json messages over a stream, ie. an arbitrary io.Reader or
io.Writer etc., using the framing used by browser native messaging, and
transparently fragmenting messages that are too large for a single frame.

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

Both frames and messages must be valid JSON: a browser parses every frame
it receives as a single JSON value, so nothing may be wrapped around, or
interleaved with, the JSON of a frame. That rules out the usual approach to
fragmentation of splitting a message's bytes across frames, since the pieces
are not themselves JSON; fragmentation is therefore done inside the JSON,
as follows.

# Wire format

The body of a frame is one of two things, a bare message or an envelope,
and which it is is decided by its first bytes alone:

    frame    = length body
    length   = the number of bytes in body, as 4 bytes, little endian
    body     = bare | envelope
    bare     = a complete JSON value that does not begin with `{"~":[`
    envelope = `{"~":[` seq [ `,` total ] [ `,` hlen ] `]` [ `,"h":` header ] `,"p":"` payload `"}`

A bare message is the message and nothing else. An envelope is a JSON object
of exactly the form above, in that order, with no white space. Its parts
are:

    seq      the sequence number of the envelope within its message, from 0
    total    only if seq is 0: the length in bytes of the whole message
    hlen     only if there is a user header: its length in bytes
    header   only if there is a user header: a JSON value, see "User header"
    payload  a chunk of the message, as the contents of a JSON string, escaped
             as described under "Escaping"; it is the whole message if the
             chunk, once unescaped, is total bytes long

All numbers are decimal, with no sign or leading zeros, and of at most 10
digits. There is deliberately no field saying whether more envelopes follow,
no message identifier and no version: the last envelope of a message is
the one that brings the bytes received up to total, a single message is in
flight in each direction at a time, and a future version of this format
would use a key other than "~". The envelope costs 19 bytes on the first
envelope of a message and 15 to 20 on the others, plus the user header if
there is one, which is of no consequence when a frame is of the order of a
megabyte; it is the encoding of the payload that determines the size of a
message.

# Fragmentation

If fragmentation is enabled (see WithFragmentation) a message that does not
fit in one frame is sent in a series of envelopes, each carrying a fragment,
a chunk, of it, as in these examples, in which the second envelope has no
total and the first has no user header:

    {"~":[0,61],"p":"{\"key\":\"a value long enough to need fragmenting"}
    {"~":[1],"p":"\",\"n\":[1,2,3]}"}

The payloads of the envelopes, once unescaped and concatenated, are the
original message exactly. A message that fits in a frame is sent bare unless
it has a user header, see below, or begins with the bytes that mark an
envelope, {"~":[. Neither side ever ignores or reinterprets any other bytes,
so there is no ambiguity. To keep that true of messages that happen to
begin with those bytes, a writer with fragmentation enabled sends such a
message in an envelope, with a total that is the size of the message, in
which case the reader unwraps it and delivers it unchanged, and one without
fragmentation enabled fails to write it, with ErrInvalidFrame.

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
and this is enforced, not estimated, while the frame is built. The writer
first writes the header of the fragment, the size of which is known exactly
since the sequence number and total are known, and sets aside the 2 bytes
of the closing "} to leave room for the payload. It then takes the message
one rune at a time, computes the exact length of that rune once escaped,
and stops at the first rune that would not fit, emitting a rune only if it,
and so its escape sequence, fits entirely. The frame is therefore within
the budget for any message, including one whose characters all escape to 6
bytes, and is as full as it can be, since it ends only when the next rune
does not fit. Every fragment holds at least one rune, so the message is
always consumed, provided the fragment size is at least MinFragmentSize,
which NewMessager requires. The same pass scans, escapes and copies,
into a single buffer that is written with a single call to Write.

# User header

Code that sits above this package may need to know where a message is to go,
or what it is, without decoding it, or even reassembling it. For that a
message may be sent with a user header, see WriteMessageWithHeader: any JSON
value, that is opaque to this package, that is a member of the envelope,
"h", between the numbers and the payload, as in the wire format above,
along with its length in bytes, which is the last of the numbers:

    {"~":[0,61,17],"h":{"to":7,"op":"x"},"p":"{\"key\":\"a value long enough"}
    {"~":[1],"p":"\",\"n\":[1,2,3]}"}

The header is part of the envelope that carries the first chunk of the
message, and not an envelope of its own, so that it is all that is needed to
route the message, since a single message is in flight in each direction.
If WithRepeatHeader is set it is also part of every other envelope, as in
{"~":[1,17],"h":{"to":7,"op":"x"},"p":"…"}, for a receiver that cannot rely
on having seen the first. The length means that a receiver finds the end of
the header without parsing it, and there is nothing to be confused by in
what the header contains: the reader checks that it is a valid JSON value,
of no more than WithMaxHeaderSize bytes, followed by exactly the bytes
that begin the payload, and that a header that is repeated is that of the
first envelope. User headers are disabled unless WithMaxHeaderSize is set,
and a header received when they are disabled is an error.

A bare message has no room for a header and so a message that has one is
always sent in an envelope, in a single one, with its total equal to its
size, if it fits. The header is accounted for in the budget as the rest
of the envelope is: it is part of what is written before the payload,
and is of known size. MinFragmentSizeWithHeader is the least fragment size
that guarantees progress when the largest header is sent.

# Forwarding

A router, or proxy, that moves messages between connections needs neither to
hold the whole of a message nor to decode it. ReadFragment returns a frame
as it is, a Fragment, with the sequence number, the total on the first, the
user header and the payload still escaped, along with the number of bytes
that it stands for once unescaped and whether it is the last of its message,
and WriteFragment writes one to another Messager. Memory use is that of one
frame, whatever the size of the message. The frames are checked in just the
way that ReadMessage checks them, by the same code, as is what WriteFragment
is given, so that what is forwarded is always a valid stream, and everything
that applies to what is read, in particular that an error is permanent,
applies. A frame is not refragmented: it is written as it is and must fit
in the frame size of the writer. The user header of a frame that is written
is the one it is given, which allows a header to be added, for example,
or the routing information to be changed, between reading and writing, and
so a router that is forwarding to a receiver that needs a header on every
fragment can supply it. Reading a message with ReadMessage part way through
reading its frames with ReadFragment, or writing one with WriteMessage
between its frames written with WriteFragment, is an error since it would
mix up the frames of two messages.

# Reading

ReadMessage returns a message complete: a fragmented message is reassembled
before it is returned. Everything read is untrusted, and the reader is
strict. A fragment must have exactly the form given above, written by
this package or by JSON.stringify of {"~":[...],p:chunk} in a browser:
any variation, such as white space, another member or a different key order,
is an error, as is anything out of order or inconsistent. Specifically:
the first fragment must have seq 0 and declare total, which must be at
least 1 and no more than the limit set by WithMaxFragmentedMessageSize,
a check made before anything is allocated for it; seq must increase by one;
a payload must be non-empty, correctly escaped and valid UTF-8; the payloads
must not add up to more than total; and the message is complete exactly
when they add up to total, so a frame that is not a fragment, or has seq 0,
arriving part way through is an error. A frame is rejected as soon as its
length is read if it is larger than WithMaxSize, and what a frame declares
never causes memory to be allocated in advance of the data to fill it
arriving.

After any error from ReadMessage, other than io.EOF between frames,
the Messager can no longer be used to read: the position in the stream is
not known, and attempting to carry on would risk treating the contents of
one frame as the start of the next. All subsequent calls return the same
error and the connection should be closed. Similarly, after an error writing
to the underlying writer WriteMessage returns that error to all later calls,
since a partially written frame or message cannot be completed or undone.
Errors found before anything is written, such as a message that is too
large, have no such effect.

# Browser extension

The extension must reassemble fragmented messages that it receives from
the host (the limit on those is 1 MB, it is the direction in which it
is needed), and may send them, but need not, since the limit to the
host is much higher. A reassembler in JavaScript is a few lines; see
testdata/fragments.js, which is also used to test this package against an
independent implementation:

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

The user header, if any, is m.h of the first fragment, and of the others
if the host repeats it; a sender in JavaScript puts it in the object,
between "~" and "p", and adds its length in bytes, JSON.stringify(h) encoded
as UTF-8, to the numbers of the header.

An ordinary message is distinguished by the same test as the Go reader uses,
except that this relies on the key "~" and an array, which a message that
starts with {"~":[ is guaranteed to have been wrapped as a fragment by the
writer. A JavaScript writer must split on code points and not UTF-16 code
units so that a surrogate pair is never divided between two fragments.

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



## Variables
### ErrMessageTooLarge, ErrInvalidFrame
```go
// ErrMessageTooLarge is returned, or wrapped by the error returned, when a
// message or frame exceeds the limits in effect.
ErrMessageTooLarge = errors.New("jsonmsgs: message too large")
// ErrInvalidFrame is returned, or wrapped by the error returned, when the
// data read is not a valid frame or sequence of fragments, or when a
// message to be written cannot be framed: it is empty, is not valid UTF-8
// when it needs to be fragmented, or begins with the bytes reserved for a
// fragment but fragmentation is not enabled.
ErrInvalidFrame = errors.New("jsonmsgs: invalid frame")

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
for messages that have no user header.

### Func MinFragmentSizeWithHeader
```go
func MinFragmentSizeWithHeader(maxFragmentedMessageSize, maxHeaderSize uint32) uint32
```
MinFragmentSizeWithHeader is MinFragmentSize for a Messager that sends user
headers of up to maxHeaderSize bytes, see WithMaxHeaderSize. The largest
header a fragment needs is that of the first, which has a user header of
that size as well as the total.



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

### Methods

```go
func (d *Decoder) Header() jsontext.Value
```
Header returns the user header that was sent with the message,
see Messager.WriteMessageWithHeader, or nil if it was sent without one.
It is valid until the Decoder is released.




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


### Type Fragment
```go
type Fragment struct {
	// Bare is true for a message that was sent as a frame of its own, with no
	// envelope, in which case Payload is the message and Header is nil.
	Bare bool

	// Seq is the sequence number of the fragment within its message, which
	// starts at 0.
	Seq uint64

	// Total is the size in bytes of the whole message, once unescaped and
	// reassembled, and is set on the first fragment, Seq 0, only, as that is
	// the only one that carries it. It is the size of the message if Bare.
	Total uint64

	// Header is the user header, as sent by WriteMessageWithHeader, a JSON
	// value that is nil if the fragment does not carry one. It is on the
	// first fragment of a message and, if the sender sets WithRepeatHeader, on
	// every other.
	Header []byte

	// Payload is the part of the message that the fragment carries, as it is
	// in the fragment: the contents of a JSON string, ie. escaped, unless Bare.
	Payload []byte

	// Len is the number of bytes of the message that Payload stands for once
	// unescaped. It is set by ReadFragment, and ignored by WriteFragment which
	// works it out.
	Len int

	// Last is true if this is the last fragment of the message, or Bare.
	Last bool
}
```
Fragment is a frame as read by ReadFragment, and as written by
WriteFragment: a fragment of a message, or a whole message that was sent as
a frame of its own, that has not been reassembled or, if it is a fragment,
unescaped. This allows a frame to be routed, and forwarded, without
buffering the message that it is a part of.


### Type Messager
```go
type Messager struct {
	// contains filtered or unexported fields
}
```
Messager reads and writes messages, see the package documentation. Reads and
writes may be made concurrently, and a message is read or written whole by
any one call, but the calls to either are serialized.

### Functions

```go
func NewMessager(rd io.ReadCloser, wr io.Writer, opts ...Option) *Messager
```
NewMessager creates a new Messager with the given readCloser and writer.
If maxSize is not specified via WithMaxSize, DefaultMaxNativeMessageSize is
used. If fragmentSize is not specified via WithFragmentSize, it defaults
to maxSize. NewMessager panics if maxSize is less than 2 or exceeds
math.MaxInt32, if the maximum fragmented message size exceeds it, or if
fragmentation is enabled and the fragment size is less than MinFragmentSize
for the maximum fragmented message size.



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
func (m *Messager) ReadFragment() (*Fragment, error)
```
ReadFragment reads the next frame from the underlying reader and returns
it as it is, without reassembling or unescaping a fragmented message, so
that it can be routed, using its Header, and forwarded, using WriteFragment
on another Messager, one at a time without holding the message in memory.
The frames of a message are returned in order, ending with one for which
Last is true, with the same checks as ReadMessage makes of them, and the
same consequences: any error other than io.EOF between messages is returned
by all subsequent calls. ReadMessage cannot be called part way through
a message, and fails if it is. The Fragment, and the slices it holds,
are valid only until the next call to ReadFragment, and must not be retained
or modified.


```go
func (m *Messager) ReadMessage() (*Decoder, error)
```
ReadMessage reads a message from the underlying reader, returning a Decoder
that can be used to decode the message. If the message was fragmented,
all of its fragments are read and reassembled into a single Decoder,
see the package documentation. The Decoder must be released by calling
ReleaseDecoder when no longer needed. ReadMessage will block until a
complete message is read or an error occurs. io.EOF is returned if the
stream ends between frames; any other error, including ErrInvalidFrame and
ErrMessageTooLarge, is permanent: it is returned by all subsequent calls,
since what follows it in the stream can no longer be told from the contents
of a frame.


```go
func (m *Messager) ReleaseDecoder(dec *Decoder)
```
ReleaseDecoder returns a Decoder obtained from ReadMessage, which cannot be
used again. It is safe to call with a Decoder that did not come from this
Messager, which is ignored.


```go
func (m *Messager) ReleaseEncoder(enc *Encoder)
```
ReleaseEncoder should only be called if WriteMessage will not be called,
for example if there is an error during encoding that will cause the message
to be discarded.


```go
func (m *Messager) WriteFragment(f *Fragment) error
```
WriteFragment writes a frame, as returned by ReadFragment, to the underlying
writer, to forward it. The frame is written as it is, rebuilt from its
fields, and not refragmented: it must fit in the fragment size, or the
maximum size if fragmentation is not enabled, of this Messager, and what
it carries is checked with the same rules as are applied to what is read.
In particular, the Header is sent if it is set, whether or not it was set
in the frame that was read, so that a sender that is not repeating headers
may add them. A message is the frames from Seq 0 to one that is Last,
which must be written one after the other, and no other message may be
written, by WriteMessage or otherwise, between them, and fails if it is.
Unlike an error writing a frame, an error that is found before writing,
such as one in the sequence, does not stop later calls, but a message part
way through is then incomplete and so is poisoned.


```go
func (m *Messager) WriteMessage(enc *Encoder) error
```
WriteMessage writes the message encoded by enc, which is returned to the
pool whatever the outcome and must not be used again. A message that fits in
a frame of at most the maximum size, or the fragment size if fragmentation
is enabled, is written as that frame, with a single call to Write,
and otherwise, if fragmentation is enabled, is fragmented, see the package
documentation, each fragment being written with a single call to Write.
ErrMessageTooLarge is returned if it is not enabled, or if the message
is larger than the maximum fragmented message size. The newline that the
encoder appends to a message is not written. A message is written whole or,
if the underlying writer fails, not completely, in which case the error is
returned by all subsequent calls.


```go
func (m *Messager) WriteMessageWithHeader(enc *Encoder, header jsontext.Value) error
```
WriteMessageWithHeader is like WriteMessage but sends header, which is
opaque to this package, with the message, as the "h" member of the envelope
that carries its first chunk, and of every envelope if WithRepeatHeader
is set, so that the message can be routed without decoding it, see the
package documentation. The header must be a valid JSON value, as is checked,
of at most the size set by WithMaxHeaderSize, and requires fragmentation
to be enabled. A message with a header is always sent in an envelope,
even if it would fit in a bare frame, since a bare frame has no room for it;
it is not sent in an additional frame of its own. As with WriteMessage, enc
is returned to the pool whatever the outcome, and must not be used again.




### Type Option
```go
type Option func(*options)
```
Option represents an option for configuring a Messager.

### Functions

```go
func WithDecoderOptions(opts jsontext.Options) Option
```
WithDecoderOptions sets the options for the decoders returned by
ReadMessage.


```go
func WithEncoderOptions(opts jsontext.Options) Option
```
WithEncoderOptions sets the options for the encoders returned by NewEncoder.


```go
func WithFragmentSize(fragmentSize uint32) Option
```
WithFragmentSize sets the maximum size in bytes of a frame written when
fragmentation is enabled, including a message sent whole: a message that
does not fit is fragmented into frames of at most this size. It defaults
to the maximum size, and if 0 or larger is set to it. If fragmentation is
enabled, NewMessager panics if fragmentSize is less than MinFragmentSize for
the maximum fragmented message size.


```go
func WithFragmentation(enable bool) Option
```
WithFragmentation controls whether automatic message fragmentation
is enabled. The default is false. When false, WriteMessage returns
ErrMessageTooLarge if a message exceeds the maximum size, and ReadMessage
returns an error if a fragment is received. When true, messages exceeding
the fragment size (which defaults to the maximum size) are automatically
fragmented into multiple frames. Both ends must have it enabled for
fragmented messages to be exchanged, but a message that is not fragmented
is the same either way, apart from the reserved prefix, see the package
documentation.


```go
func WithMaxFragmentedMessageSize(maxSize uint32) Option
```
WithMaxFragmentedMessageSize sets the maximum total size of a message
that can be fragmented in bytes, which is to say the largest that
can be written and that will be accepted when reassembled. If 0,
DefaultMaxFragmentedMessageSize (16MB) is used. WriteMessage returns
ErrMessageTooLarge for a message that exceeds it and ReadMessage for one
that declares a total that does, before reading any of it. It protects
receivers from unbounded memory growth, and prevents deadlocks when writing
over buffered channels that could fill up before a complete message is sent.
NewMessager panics if it exceeds math.MaxInt32.


```go
func WithMaxHeaderSize(maxHeaderSize uint32) Option
```
WithMaxHeaderSize enables user headers, see WriteMessageWithHeader, of up
to maxHeaderSize bytes, which is also the largest accepted by ReadMessage
and ReadFragment: a frame with a header is an error without it. It requires
fragmentation, which is what carries a header, and increases the smallest
fragment size that NewMessager accepts, see MinFragmentSizeWithHeader.


```go
func WithMaxSize(maxSize uint32) Option
```
WithMaxSize sets the maximum size of a single frame in bytes: no frame
larger is accepted by ReadMessage, which rejects it on seeing its length,
and, if fragmentation is not enabled, no message larger is written by
WriteMessage. If maxSize is 0, DefaultMaxNativeMessageSize is used. It
should be at least as large as the largest frame a peer will send; the limit
of a browser on frames to the host is much larger than that on frames from
it. NewMessager panics if maxSize is less than 2 or exceeds math.MaxInt32.


```go
func WithRepeatHeader(repeat bool) Option
```
WithRepeatHeader sends the user header of a message on each of its fragments
rather than only on the first, so that a receiver can route a fragment
without having seen the first. It costs the size of the header in each
fragment.







