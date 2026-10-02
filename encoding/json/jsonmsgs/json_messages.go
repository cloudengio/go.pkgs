// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package jsonmsgs provides support for efficient encoding and decoding
// arbitrary json messages over a stream, ie. an arbitrary io.Reader or
// io.Writer etc., using the framing used by browser native messaging, and
// transparently fragmenting messages that are too large for a single frame.
//
// # Framing
//
// Every frame is a 4 byte little endian length followed by exactly that many
// bytes (the body), which are one complete JSON value:
//
//	[len:4 LE][one complete JSON value]
//
// This is the framing used by Chrome and Firefox to talk to a native
// messaging host: "each message is serialized using JSON, UTF-8 encoded and is
// preceded with 32-bit message length in native byte order". A message that
// fits in a single frame is sent as its bare JSON, byte for byte what a
// browser sends and receives, with nothing added to it: no flags, version or
// padding. Browsers limit a message from the host to 1 MB, and one to the host
// to 64 MiB (Chrome) or 4 GB (Firefox); see DefaultMaxNativeMessageSize.
// Byte order is always little endian, which is the native order of every
// platform that browsers run on.
//
// Both frames and messages must be valid JSON: a browser parses every frame it
// receives as a single JSON value, so nothing may be wrapped around, or
// interleaved with, the JSON of a frame. That rules out the usual approach to
// fragmentation of splitting a message's bytes across frames, since the pieces
// are not themselves JSON; fragmentation is therefore done inside the JSON, as
// follows.
//
// # Wire format
//
// The body of a frame is one of two things, a bare message or an envelope, and
// which it is is decided by its first bytes alone:
//
//	frame    = length body
//	length   = the number of bytes in body, as 4 bytes, little endian
//	body     = bare | envelope
//	bare     = a complete JSON value that does not begin with `{"~":[`
//	envelope = `{"~":[` seq [ `,` total ] [ `,` hlen ] `]` [ `,"h":` header ] `,"p":"` payload `"}`
//
// A bare message is the message and nothing else. An envelope is a JSON object
// of exactly the form above, in that order, with no white space. Its parts
// are:
//
//	seq      the sequence number of the envelope within its message, from 0
//	total    only if seq is 0: the length in bytes of the whole message
//	hlen     only if there is a user header: its length in bytes
//	header   only if there is a user header: a JSON value, see "User header"
//	payload  a chunk of the message, as the contents of a JSON string, escaped
//	         as described under "Escaping"; it is the whole message if the
//	         chunk, once unescaped, is total bytes long
//
// All numbers are decimal, with no sign or leading zeros, and of at most 10
// digits. There is deliberately no field saying whether more envelopes follow,
// no message identifier and no version: the last envelope of a message is the
// one that brings the bytes received up to total, a single message is in
// flight in each direction at a time, and a future version of this format
// would use a key other than "~". The envelope costs 19 bytes on the first
// envelope of a message and 15 to 20 on the others, plus the user header if
// there is one, which is of no consequence when a frame is of the order of a
// megabyte; it is the encoding of the payload that determines the size of a
// message.
//
// # Fragmentation
//
// If fragmentation is enabled (see WithFragmentation) a message that does not
// fit in one frame is sent in a series of envelopes, each carrying a
// fragment, a chunk, of it, as in these examples, in which the second envelope
// has no total and the first has no user header:
//
//	{"~":[0,61],"p":"{\"key\":\"a value long enough to need fragmenting"}
//	{"~":[1],"p":"\",\"n\":[1,2,3]}"}
//
// The payloads of the envelopes, once unescaped and concatenated, are the
// original message exactly. A message that fits in a frame is sent bare unless
// it has a user header, see below, or begins with the bytes that mark an
// envelope, {"~":[. Neither side ever ignores or reinterprets any other bytes,
// so there is no ambiguity. To keep that true of messages that happen to begin
// with those bytes, a writer with fragmentation enabled sends such a message
// in an envelope, with a total that is the size of the message, in which case
// the reader unwraps it and delivers it unchanged, and one without
// fragmentation enabled fails to write it, with ErrInvalidFrame.
//
// # Escaping
//
// The payload is escaped exactly as JSON.stringify does, so that a browser
// recovers it with JSON.parse and nothing more, no base64:
//
//	"                                    \"     2 bytes
//	\                                    \\     2 bytes
//	backspace, form feed, newline,       \b \f \n \r \t   2 bytes
//	carriage return, tab
//	any other character below U+0020     \u00xx (lower case hex)   6 bytes
//	everything else, including U+007F,   unchanged   1 to 4 bytes
//	U+2028/9 and all non-ASCII
//
// A message must therefore be valid UTF-8, as JSON requires, and is split only
// on rune boundaries so that every payload is valid UTF-8 in its own right. A
// JSON message produced by this package never contains a character of the 6
// byte kind, so the payload of one is at most twice the size of the message,
// and in practice a few tens of percent at most for typical JSON, whose
// escapes are mostly the quotes around its strings and none at all for
// numbers and text. Base64 would add a third.
//
// # Fitting the size budget
//
// A frame is never larger than the fragment size, see WithFragmentSize, and
// this is enforced, not estimated, while the frame is built. The writer first
// writes the header of the fragment, the size of which is known exactly since
// the sequence number and total are known, and sets aside the 2 bytes of the
// closing "} to leave room for the payload. It then takes the message one rune
// at a time, computes the exact length of that rune once escaped, and stops
// at the first rune that would not fit, emitting a rune only if it, and so
// its escape sequence, fits entirely. The frame is therefore within the
// budget for any message, including one whose characters all escape to 6
// bytes, and is as full as it can be, since it ends only when the next rune
// does not fit. Every fragment holds at least one rune, so the message is
// always consumed, provided the fragment size is at least MinFragmentSize,
// which NewMessager requires. The same pass scans, escapes and copies, into a
// single buffer that is written with a single call to Write.
//
// # User header
//
// Code that sits above this package may need to know where a message is to go,
// or what it is, without decoding it, or even reassembling it. For that a
// message may be sent with a user header, see WriteMessageWithHeader: any JSON
// value, that is opaque to this package, that is a member of the envelope, "h",
// between the numbers and the payload, as in the wire format above, along with
// its length in bytes, which is the last of the numbers:
//
//	{"~":[0,61,17],"h":{"to":7,"op":"x"},"p":"{\"key\":\"a value long enough"}
//	{"~":[1],"p":"\",\"n\":[1,2,3]}"}
//
// The header is part of the envelope that carries the first chunk of the
// message, and not an envelope of its own, so that it is all that is needed to
// route the message, since a single message is in flight in each direction. If
// WithRepeatHeader is set it is also part of every other envelope, as in
// {"~":[1,17],"h":{"to":7,"op":"x"},"p":"…"}, for a receiver that cannot rely
// on having seen the first. The length means that a receiver finds the end of
// the header without parsing it, and there is nothing to be confused by in what
// the header contains: the reader checks that it is a valid JSON value, of no
// more than WithMaxHeaderSize bytes, followed by exactly the bytes that begin
// the payload, and that a header that is repeated is that of the first
// envelope. User headers are disabled unless WithMaxHeaderSize is set, and a
// header received when they are disabled is an error.
//
// A bare message has no room for a header and so a message that has one is
// always sent in an envelope, in a single one, with its total equal to its
// size, if it fits. The header is accounted for in the budget as the rest of
// the envelope is: it is part of what is written before the payload, and is of
// known size. MinFragmentSizeWithHeader is the least fragment size that
// guarantees progress when the largest header is sent.
//
// # Forwarding
//
// A router, or proxy, that moves messages between connections needs neither to
// hold the whole of a message nor to decode it. ReadFragment returns a frame
// as it is, a Fragment, with the sequence number, the total on the first, the
// user header and the payload still escaped, along with the number of bytes
// that it stands for once unescaped and whether it is the last of its message,
// and WriteFragment writes one to another Messager. Memory use is that of one
// frame, whatever the size of the message. The frames are checked in just
// the way that ReadMessage checks them, by the same code, as is what
// WriteFragment is given, so that what is forwarded is always a valid
// stream, and everything that applies to what is read, in particular that an
// error is permanent, applies. A frame is not refragmented: it is
// written as it is and must fit in the frame size of the writer. The user header
// of a frame that is written is the one it is given, which allows a header to
// be added, for example, or the routing information to be changed, between
// reading and writing, and so a router that is forwarding to a receiver that
// needs a header on every fragment can supply it. Reading a message with
// ReadMessage part way through reading its frames with ReadFragment, or
// writing one with WriteMessage between its frames written with WriteFragment,
// is an error since it would mix up the frames of two messages.
//
// # Reading
//
// ReadMessage returns a message complete: a fragmented message is reassembled
// before it is returned. Everything read is untrusted, and the reader is
// strict. A fragment must have exactly the form given above, written by this
// package or by JSON.stringify of {"~":[...],p:chunk} in a browser: any
// variation, such as white space, another member or a different key order, is an
// error, as is anything out of order or inconsistent. Specifically: the
// first fragment must have seq 0 and declare total, which must be at least 1
// and no more than the limit set by WithMaxFragmentedMessageSize, a check made
// before anything is allocated for it; seq must increase by one; a payload must
// be non-empty, correctly escaped and valid UTF-8; the payloads must not add up
// to more than total; and the message is complete exactly when they add up
// to total, so a frame that is not a fragment, or has seq 0, arriving part way
// through is an error. A frame is rejected as soon as its length is read if it
// is larger than WithMaxSize, and what a frame declares never causes memory to
// be allocated in advance of the data to fill it arriving.
//
// After any error from ReadMessage, other than io.EOF between frames, the
// Messager can no longer be used to read: the position in the stream is not
// known, and attempting to carry on would risk treating the contents of one
// frame as the start of the next. All subsequent calls return the same error
// and the connection should be closed. Similarly, after an error writing to
// the underlying writer WriteMessage returns that error to all later calls,
// since a partially written frame or message cannot be completed or undone.
// Errors found before anything is written, such as a message that is too large,
// have no such effect.
//
// # Browser extension
//
// The extension must reassemble fragmented messages that it receives from the
// host (the limit on those is 1 MB, it is the direction in which it is needed),
// and may send them, but need not, since the limit to the host is much higher.
// A reassembler in JavaScript is a few lines; see testdata/fragments.js, which
// is also used to test this package against an independent implementation:
//
//	let parts = [], total = 0, got = 0;
//	function onMessage(m) {  // m is the parsed JSON of a frame
//	  const h = (m !== null && typeof m === "object") ? m["~"] : undefined;
//	  if (!Array.isArray(h)) return m;           // an ordinary message
//	  if (h[0] !== parts.length) throw new Error("bad fragment");
//	  if (h[0] === 0) total = h[1];
//	  parts.push(m.p); got += new TextEncoder().encode(m.p).length;
//	  if (got < total) return undefined;         // more to come
//	  const text = parts.join(""); parts = []; got = 0;
//	  return JSON.parse(text);
//	}
//
// The user header, if any, is m.h of the first fragment, and of the others if
// the host repeats it; a sender in JavaScript puts it in the object, between
// "~" and "p", and adds its length in bytes, JSON.stringify(h) encoded as
// UTF-8, to the numbers of the header.
//
// An ordinary message is distinguished by the same test as the Go reader uses,
// except that this relies on the key "~" and an array, which a message that
// starts with {"~":[ is guaranteed to have been wrapped as a fragment by the
// writer. A JavaScript writer must split on code points and not UTF-16 code
// units so that a surrogate pair is never divided between two fragments.
package jsonmsgs

import (
	"bytes"
	"encoding/binary"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"strconv"
	"sync"
	"unicode/utf8"
)

// DefaultMaxNativeMessageSize is the default maximum size of a single frame,
// in bytes: 1,000,000 bytes, which is below Chrome's and Firefox's limit of
// 1 MB on a message from a native messaging host whether that is taken to be
// 10^6 or 2^20 bytes. The limit applies to the length of the frame, ie. to
// the length of the message when it is not fragmented, and to the length of
// each fragment, including its header and its escaped payload, when it is.
const DefaultMaxNativeMessageSize = 1_000_000

// DefaultMaxFragmentedMessageSize is the default maximum total size of a
// fragmented/reassembled message, in bytes (16MB).
const DefaultMaxFragmentedMessageSize = 16 * 1024 * 1024 // 16MB

// maxRetainedBuffer is the capacity above which a buffer is dropped, rather
// than kept for reuse, when it is returned to a pool or finished with, so that
// one very large message does not pin that memory for the life of the Messager.
const maxRetainedBuffer = 4 * 1024 * 1024

// The pieces of a fragment, see the package documentation. Every fragment is
// envelopePrefix, the sequence number and, for the first, ",<total>", then
// envelopeMid, the escaped payload and envelopeSuffix.
const (
	envelopePrefix = `{"~":[`
	envelopeFirst  = `{"~":[0,`
	envelopeMid    = `],"p":"`
	// With a user header the closing ] of the numbers is followed by the header
	// and the payload is preceded by envelopeHdrEnd rather than envelopeMid.
	envelopeHdrMid = `],"h":`
	envelopeHdrEnd = `,"p":"`
	envelopeSuffix = `"}`

	// maxEscapedLen is the length of the longest escape sequence, \u00xx.
	maxEscapedLen = 6

	// maxDigits is the most digits accepted in a sequence number or total.
	maxDigits = 10
)

var (
	// ErrMessageTooLarge is returned, or wrapped by the error returned, when a
	// message or frame exceeds the limits in effect.
	ErrMessageTooLarge = errors.New("jsonmsgs: message too large")

	// ErrInvalidFrame is returned, or wrapped by the error returned, when the
	// data read is not a valid frame or sequence of fragments, or when a
	// message to be written cannot be framed: it is empty, is not valid UTF-8
	// when it needs to be fragmented, or begins with the bytes reserved for a
	// fragment but fragmentation is not enabled.
	ErrInvalidFrame = errors.New("jsonmsgs: invalid frame")
)

// MinFragmentSize returns the smallest fragment size, see WithFragmentSize,
// with which a message of up to maxFragmentedMessageSize bytes can always be
// fragmented. It is the largest header that such a message needs, the 2 bytes
// that close a fragment and the longest escape sequence, so that every
// fragment can carry at least one character. It is MinFragmentSizeWithHeader
// for messages that have no user header.
func MinFragmentSize(maxFragmentedMessageSize uint32) uint32 {
	return MinFragmentSizeWithHeader(maxFragmentedMessageSize, 0)
}

// MinFragmentSizeWithHeader is MinFragmentSize for a Messager that sends user
// headers of up to maxHeaderSize bytes, see WithMaxHeaderSize. The largest
// header a fragment needs is that of the first, which has a user header of
// that size as well as the total.
func MinFragmentSizeWithHeader(maxFragmentedMessageSize, maxHeaderSize uint32) uint32 {
	n := len(envelopeFirst) + len(strconv.FormatUint(uint64(maxFragmentedMessageSize), 10)) + len(envelopeSuffix) + maxEscapedLen
	if maxHeaderSize == 0 {
		return uint32(n + len(envelopeMid))
	}
	n += len(",") + len(strconv.FormatUint(uint64(maxHeaderSize), 10)) + len(envelopeHdrMid) + len(envelopeHdrEnd)
	return uint32(n) + maxHeaderSize
}

type options struct {
	// maxSize specifies the maximum size of a single frame in bytes, the
	// largest that will be read, and the largest message written whole.
	maxSize uint32

	// fragmentSize specifies the maximum size of a fragment frame in bytes.
	// It defaults to maxSize, but can be configured to be smaller.
	fragmentSize uint32

	// maxFragmentedMessageSize specifies the maximum size of a message that can
	// be fragmented in bytes. If 0, DefaultMaxFragmentedMessageSize (16MB) is used.
	maxFragmentedMessageSize uint32

	// maxHeaderSize is the largest user header, in bytes, that may be sent or
	// received. 0, the default, disables user headers.
	maxHeaderSize uint32

	// repeatHeader sends the user header on every fragment of a message.
	repeatHeader bool

	// fragmentation enables automatic message fragmentation.
	// Defaults to false.
	fragmentation bool

	encoderOptions jsontext.Options
	decoderOptions jsontext.Options
}

// Option represents an option for configuring a Messager.
type Option func(*options)

// WithMaxSize sets the maximum size of a single frame in bytes: no frame
// larger is accepted by ReadMessage, which rejects it on seeing its length,
// and, if fragmentation is not enabled, no message larger is written by
// WriteMessage. If maxSize is 0, DefaultMaxNativeMessageSize is used. It
// should be at least as large as the largest frame a peer will send;
// the limit of a browser on frames to the host is much larger than that on
// frames from it. NewMessager panics if maxSize is less than 2 or exceeds
// math.MaxInt32.
func WithMaxSize(maxSize uint32) Option {
	return func(opts *options) {
		opts.maxSize = maxSize
	}
}

// WithFragmentSize sets the maximum size in bytes of a frame written when
// fragmentation is enabled, including a message sent whole: a message that
// does not fit is fragmented into frames of at most this size. It defaults to
// the maximum size, and if 0 or larger is set to it. If fragmentation is
// enabled, NewMessager panics if fragmentSize is less than MinFragmentSize
// for the maximum fragmented message size.
func WithFragmentSize(fragmentSize uint32) Option {
	return func(opts *options) {
		opts.fragmentSize = fragmentSize
	}
}

// WithMaxFragmentedMessageSize sets the maximum total size of a message that can
// be fragmented in bytes, which is to say the largest that can be written and
// that will be accepted when reassembled. If 0, DefaultMaxFragmentedMessageSize
// (16MB) is used. WriteMessage returns ErrMessageTooLarge for a message that
// exceeds it and ReadMessage for one that declares a total that does, before
// reading any of it. It protects receivers from unbounded memory growth, and
// prevents deadlocks when writing over buffered channels that could fill up
// before a complete message is sent. NewMessager panics if it exceeds
// math.MaxInt32.
func WithMaxFragmentedMessageSize(maxSize uint32) Option {
	return func(opts *options) {
		opts.maxFragmentedMessageSize = maxSize
	}
}

// WithFragmentation controls whether automatic message fragmentation is enabled.
// The default is false. When false, WriteMessage returns ErrMessageTooLarge if
// a message exceeds the maximum size, and ReadMessage returns an error if a
// fragment is received. When true, messages exceeding the fragment size (which
// defaults to the maximum size) are automatically fragmented into multiple
// frames. Both ends must have it enabled for fragmented messages to be
// exchanged, but a message that is not fragmented is the same either way, apart
// from the reserved prefix, see the package documentation.
func WithFragmentation(enable bool) Option {
	return func(opts *options) {
		opts.fragmentation = enable
	}
}

// WithMaxHeaderSize enables user headers, see WriteMessageWithHeader, of up to
// maxHeaderSize bytes, which is also the largest accepted by ReadMessage and
// ReadFragment: a frame with a header is an error without it. It requires
// fragmentation, which is what carries a header, and increases the smallest
// fragment size that NewMessager accepts, see MinFragmentSizeWithHeader.
func WithMaxHeaderSize(maxHeaderSize uint32) Option {
	return func(opts *options) {
		opts.maxHeaderSize = maxHeaderSize
	}
}

// WithRepeatHeader sends the user header of a message on each of its
// fragments rather than only on the first, so that a receiver can route a
// fragment without having seen the first. It costs the size of the header
// in each fragment.
func WithRepeatHeader(repeat bool) Option {
	return func(opts *options) {
		opts.repeatHeader = repeat
	}
}

// WithEncoderOptions sets the options for the encoders returned by NewEncoder.
func WithEncoderOptions(opts jsontext.Options) Option {
	return func(o *options) {
		o.encoderOptions = opts
	}
}

// WithDecoderOptions sets the options for the decoders returned by ReadMessage.
func WithDecoderOptions(opts jsontext.Options) Option {
	return func(o *options) {
		o.decoderOptions = opts
	}
}

// Encoder captures the state to encode and send a single message.
// It must be obtained using Messager.NewEncoder. It will be reclaimed by
// Messager.WriteMessage after which it cannot be used again.
type Encoder struct {
	*jsontext.Encoder
	opts   jsontext.Options
	buffer *bytes.Buffer
}

// Decoder captures the state to decode a single message.
// It is created and returned by Messager.ReadMessage and must released by
// calling Messager.ReleaseDecoder after which it cannot be used again.
type Decoder struct {
	*jsontext.Decoder
	opts   jsontext.Options
	buf    *bytes.Buffer
	buffer []byte
	header []byte
}

// Header returns the user header that was sent with the message, see
// Messager.WriteMessageWithHeader, or nil if it was sent without one. It is
// valid until the Decoder is released.
func (d *Decoder) Header() jsontext.Value {
	if len(d.header) == 0 {
		return nil
	}
	return d.header
}

// Messager reads and writes messages, see the package documentation. Reads
// and writes may be made concurrently, and a message is read or written
// whole by any one call, but the calls to either are serialized.
type Messager struct {
	rd                       io.ReadCloser
	wr                       io.Writer
	maxSize                  uint32
	fragmentSize             uint32
	maxFragmentedMessageSize uint32
	fragmentation            bool
	maxHeaderSize            uint32
	repeatHeader             bool
	encPool                  sync.Pool
	decPool                  sync.Pool

	wmu     sync.Mutex
	werr    error     // the error that failed a write, returned by all later writes; guarded by wmu
	fragBuf []byte    // the frame being built by writeFragments; guarded by wmu
	wst     fragState // the message being forwarded by WriteFragment; guarded by wmu

	rmu   sync.Mutex
	rerr  error     // the error that failed a read, returned by all later reads; guarded by rmu
	hdr   [4]byte   // scratch for frame lengths, here so that it does not escape to the heap; guarded by rmu
	rst   fragState // the message being read; guarded by rmu
	fbody []byte    // the frame returned by ReadFragment; guarded by rmu
	frag  Fragment  // returned by ReadFragment; guarded by rmu
}

// NewMessager creates a new Messager with the given readCloser and writer.
// If maxSize is not specified via WithMaxSize, DefaultMaxNativeMessageSize is
// used. If fragmentSize is not specified via WithFragmentSize, it defaults to
// maxSize. NewMessager panics if maxSize is less than 2 or exceeds
// math.MaxInt32, if the maximum fragmented message size exceeds it, or if
// fragmentation is enabled and the fragment size is less than MinFragmentSize
// for the maximum fragmented message size.
func NewMessager(rd io.ReadCloser, wr io.Writer, opts ...Option) *Messager {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if o.maxSize == 0 {
		o.maxSize = DefaultMaxNativeMessageSize
	}
	if o.maxSize < 2 || o.maxSize > math.MaxInt32 {
		panic(fmt.Sprintf("jsonmsgs: maxSize %d must be between 2 and %d", o.maxSize, math.MaxInt32))
	}
	if o.fragmentSize == 0 || o.fragmentSize > o.maxSize {
		o.fragmentSize = o.maxSize
	}
	if o.maxFragmentedMessageSize == 0 {
		o.maxFragmentedMessageSize = DefaultMaxFragmentedMessageSize
	}
	if o.maxFragmentedMessageSize > math.MaxInt32 {
		panic(fmt.Sprintf("jsonmsgs: maxFragmentedMessageSize %d must not exceed %d", o.maxFragmentedMessageSize, math.MaxInt32))
	}
	if o.maxHeaderSize > math.MaxInt32 {
		panic(fmt.Sprintf("jsonmsgs: maxHeaderSize %d must not exceed %d", o.maxHeaderSize, math.MaxInt32))
	}
	if least := MinFragmentSizeWithHeader(o.maxFragmentedMessageSize, o.maxHeaderSize); o.fragmentation && o.fragmentSize < least {
		panic(fmt.Sprintf("jsonmsgs: fragment size %d must be at least %d to fragment messages of up to %d bytes",
			o.fragmentSize, least, o.maxFragmentedMessageSize))
	}
	if wr == nil {
		wr = io.Discard
	}
	if rd == nil {
		rd = io.NopCloser(bytes.NewReader(nil))
	}
	nm := &Messager{
		wr:                       wr,
		rd:                       rd,
		maxSize:                  o.maxSize,
		fragmentSize:             o.fragmentSize,
		maxFragmentedMessageSize: o.maxFragmentedMessageSize,
		fragmentation:            o.fragmentation,
		maxHeaderSize:            o.maxHeaderSize,
		repeatHeader:             o.repeatHeader,
	}
	nm.encPool = sync.Pool{
		New: func() any {
			buf := bytes.NewBuffer(make([]byte, 0, 1024))
			buf.Write([]byte{0, 0, 0, 0})
			return &Encoder{
				Encoder: jsontext.NewEncoder(buf, o.encoderOptions),
				opts:    o.encoderOptions,
				buffer:  buf,
			}
		},
	}
	nm.decPool = sync.Pool{
		New: func() any {
			buf := bytes.NewBuffer(make([]byte, 0, 256))
			return &Decoder{
				Decoder: jsontext.NewDecoder(buf, o.decoderOptions),
				buf:     buf,
				opts:    o.decoderOptions,
			}
		},
	}
	return nm
}

// NewEncoder creates a new Encoder for encoding a single message.
func (m *Messager) NewEncoder() *Encoder {
	enc := m.encPool.Get().(*Encoder)
	enc.buffer.Reset()
	// Room for the frame length, which WriteMessage fills in, so that the
	// frame is contiguous and can be written with a single call.
	enc.buffer.Write([]byte{0, 0, 0, 0})
	enc.Reset(enc.buffer, enc.opts)
	return enc
}

// ReleaseEncoder should only be called if WriteMessage will not be called,
// for example if there is an error during encoding that will cause the message
// to be discarded.
func (m *Messager) ReleaseEncoder(enc *Encoder) {
	if enc == nil || enc.buffer == nil {
		return
	}
	m.putEncoder(enc)
}

// putEncoder returns enc to the pool, first dropping a buffer that has grown
// large so that a single large message does not pin its memory in the pool.
func (m *Messager) putEncoder(enc *Encoder) {
	if enc.buffer.Cap() > maxRetainedBuffer && enc.buffer.Cap() > int(m.maxSize)*4 {
		enc.buffer = bytes.NewBuffer(make([]byte, 0, 1024))
	}
	m.encPool.Put(enc)
}

// ReleaseDecoder returns a Decoder obtained from ReadMessage, which cannot be
// used again. It is safe to call with a Decoder that did not come from this
// Messager, which is ignored.
func (m *Messager) ReleaseDecoder(dec *Decoder) {
	if dec == nil || dec.buf == nil {
		return
	}
	if cap(dec.buffer) > int(m.maxSize)*4 && cap(dec.buffer) > maxRetainedBuffer {
		dec.buffer = nil
		// dec.buf aliases the old dec.buffer's backing array (bytes.NewBuffer
		// does not copy), so it must be replaced too, or the large array
		// stays reachable via dec.buf until this decoder is next reused.
		dec.buf = bytes.NewBuffer(nil)
	}
	m.decPool.Put(dec)
}

// Close closes the underlying reader of the Messager causing a pending
// ReadMessage to return.
func (m *Messager) Close() error {
	if m.rd == nil {
		return nil
	}
	return m.rd.Close()
}

func writeFull(w io.Writer, data []byte) error {
	for off := 0; off < len(data); {
		n, err := w.Write(data[off:])
		if n > 0 {
			off += n
		}
		if err != nil {
			return fmt.Errorf("failed to write complete message: %w", err)
		}
		if n == 0 {
			return fmt.Errorf("failed to write complete message: %w", io.ErrShortWrite)
		}
	}
	return nil
}

// WriteMessage writes the message encoded by enc, which is returned to the
// pool whatever the outcome and must not be used again. A message that fits
// in a frame of at most the maximum size, or the fragment size if
// fragmentation is enabled, is written as that frame, with a single call to
// Write, and otherwise, if fragmentation is enabled, is fragmented, see the
// package documentation, each fragment being written with a single call to
// Write. ErrMessageTooLarge is returned if it is not enabled, or if the
// message is larger than the maximum fragmented message size. The newline that
// the encoder appends to a message is not written. A message is written
// whole or, if the underlying writer fails, not completely, in which case the
// error is returned by all subsequent calls.
func (m *Messager) WriteMessage(enc *Encoder) error {
	return m.writeMessage(enc, nil)
}

// WriteMessageWithHeader is like WriteMessage but sends header, which is
// opaque to this package, with the message, as the "h" member of the envelope
// that carries its first chunk, and of every envelope if WithRepeatHeader is
// set, so that the message can be routed without decoding it, see the package
// documentation. The header must be a valid JSON value, as is checked, of at
// most the size set by WithMaxHeaderSize, and requires fragmentation to be
// enabled. A message with a header is always sent in an envelope, even if it
// would fit in a bare frame, since a bare frame has no room for it; it is
// not sent in an additional frame of its own. As with WriteMessage, enc is
// returned to the pool whatever the outcome, and must not be used again.
func (m *Messager) WriteMessageWithHeader(enc *Encoder, header jsontext.Value) error {
	if len(header) == 0 {
		// Return enc to the pool, as writeMessage does for every other outcome,
		// since the caller has handed it over and will not use or release it.
		m.ReleaseEncoder(enc)
		return fmt.Errorf("%w: empty header", ErrInvalidFrame)
	}
	return m.writeMessage(enc, header)
}

// checkHeader checks a user header that is to be sent.
func (m *Messager) checkHeader(h []byte) error {
	switch {
	case !m.fragmentation:
		return fmt.Errorf("%w: a header requires fragmentation to be enabled", ErrInvalidFrame)
	case m.maxHeaderSize == 0:
		return fmt.Errorf("%w: headers are not enabled, see WithMaxHeaderSize", ErrInvalidFrame)
	case uint64(len(h)) > uint64(m.maxHeaderSize):
		return fmt.Errorf("%w: header size %d exceeds maximum %d", ErrMessageTooLarge, len(h), m.maxHeaderSize)
	case !jsontext.Value(h).IsValid():
		return fmt.Errorf("%w: header is not valid JSON", ErrInvalidFrame)
	}
	return nil
}

func (m *Messager) writeMessage(enc *Encoder, header []byte) error {
	if enc == nil || enc.buffer == nil {
		return errors.New("nil encoder")
	}
	m.wmu.Lock()
	defer m.wmu.Unlock()
	defer m.putEncoder(enc)
	if m.werr != nil {
		return m.werr
	}
	if m.wst.active {
		return fmt.Errorf("%w: a message is being forwarded with WriteFragment", ErrInvalidFrame)
	}
	if header != nil {
		if err := m.checkHeader(header); err != nil {
			return err
		}
	}
	data, err := m.encodedMessage(enc)
	if err != nil {
		return err
	}
	return m.sendMessage(data, header)
}

// encodedMessage returns the frame, with room for its length, that holds the
// message encoded by enc after checking that it is not too large or empty.
func (m *Messager) encodedMessage(enc *Encoder) ([]byte, error) {
	data := enc.buffer.Bytes()
	if len(data) < 4 {
		return nil, fmt.Errorf("%w: buffer too small to write length prefix", ErrInvalidFrame)
	}
	if err := m.checkRawSize(uint64(len(data) - 4)); err != nil {
		return nil, err
	}
	// The encoder appends a newline to every value it writes. It is white
	// space to JSON, but not part of the message, is not what a browser
	// sends, and would make the length of the message not that of its text.
	if data[len(data)-1] == '\n' && len(data) > 4 {
		data = data[:len(data)-1]
	}
	if len(data) == 4 {
		return nil, fmt.Errorf("%w: empty message", ErrInvalidFrame)
	}
	return data, nil
}

// sendMessage writes the frame data, as returned by encodedMessage, as a
// single frame or as fragments, as described by WriteMessage.
func (m *Messager) sendMessage(data, header []byte) error {
	size := uint64(len(data) - 4)
	reserved := bytes.HasPrefix(data[4:], []byte(envelopePrefix))
	switch {
	case header != nil:
		// Always as fragments, since only they have room for a header.
	case !m.fragmentation:
		if size > uint64(m.maxSize) {
			return fmt.Errorf("%w: message size %d exceeds maximum %d", ErrMessageTooLarge, size, m.maxSize)
		}
		if reserved {
			return fmt.Errorf("%w: a message may not begin with %s unless fragmentation is enabled", ErrInvalidFrame, envelopePrefix)
		}
		return m.writeFrame(data, size)
	case size <= uint64(m.fragmentSize) && !reserved:
		// With fragmentation enabled, a message sent whole is held to the same
		// size as a fragment. One that would be mistaken for a fragment is
		// sent as one, which the reader unwraps.
		return m.writeFrame(data, size)
	}
	if size > uint64(m.maxFragmentedMessageSize) {
		return fmt.Errorf("%w: message size %d exceeds maximum fragmented message size %d", ErrMessageTooLarge, size, m.maxFragmentedMessageSize)
	}
	return m.writeFragments(data[4:], header)
}

// checkRawSize rejects an encoded message of n bytes, which may include the
// newline that is trimmed from it, that is too large to be written, before any
// of it is looked at.
func (m *Messager) checkRawSize(n uint64) error {
	limit := uint64(m.maxSize)
	if m.fragmentation {
		limit = uint64(m.maxFragmentedMessageSize)
	}
	if n > limit+1 {
		return fmt.Errorf("%w: message size %d exceeds maximum %d", ErrMessageTooLarge, n, limit)
	}
	return nil
}

// writeFrame writes data, which is a frame with room for its length at its
// start, and size bytes of body, as a single frame.
func (m *Messager) writeFrame(data []byte, size uint64) error {
	binary.LittleEndian.PutUint32(data, uint32(size))
	if err := writeFull(m.wr, data); err != nil {
		m.werr = err
		return err
	}
	return nil
}

// escLen is the length in bytes of each ASCII character once escaped for a
// JSON string, see the table in the package documentation.
var escLen = func() (t [utf8.RuneSelf]uint8) {
	for i := range t {
		t[i] = 1
	}
	for i := range 0x20 {
		t[i] = maxEscapedLen
	}
	for _, c := range []byte{'\b', '\f', '\n', '\r', '\t', '"', '\\'} {
		t[c] = 2
	}
	return t
}()

// appendEscape appends the escape sequence for the ASCII character c, which
// has an escape sequence, ie. escLen[c] is not 1.
func appendEscape(dst []byte, c byte) []byte {
	switch c {
	case '"', '\\':
		return append(dst, '\\', c)
	case '\b':
		return append(dst, '\\', 'b')
	case '\f':
		return append(dst, '\\', 'f')
	case '\n':
		return append(dst, '\\', 'n')
	case '\r':
		return append(dst, '\\', 'r')
	case '\t':
		return append(dst, '\\', 't')
	}
	const hex = "0123456789abcdef"
	return append(dst, '\\', 'u', '0', '0', hex[c>>4], hex[c&0xf])
}

// appendEscaped appends the characters of the valid UTF-8 text src to dst,
// escaped as the contents of a JSON string, for as long as the escaped form
// fits in room bytes, and returns the extended slice and the number of bytes
// of src it consumed. A character, and so its escape sequence, is written
// whole or not at all: it stops before the first that does not fit in what
// remains of room. This is what guarantees that a fragment fits in the
// budget, see the package documentation.
//
// Runs of characters that need no escape are copied in one operation, rather
// than character by character; start is the beginning of the run not yet
// copied.
func appendEscaped(dst, src []byte, room int) (out []byte, consumed int) {
	used, i, start := 0, 0, 0
	for i < len(src) {
		// Fast path for a run of characters that are themselves, each of which
		// is a single byte, bounded by what remains of room.
		if end := min(len(src), i+room-used); i < end {
			j := i
			for j < end && src[j] < utf8.RuneSelf && escLen[src[j]] == 1 {
				j++
			}
			used += j - i
			i = j
			if i == len(src) {
				break
			}
		}
		c := src[i]
		var width, esc int
		if c < utf8.RuneSelf {
			width, esc = 1, int(escLen[c])
		} else {
			// The text is valid UTF-8, which the caller has checked, so
			// the width of a character is that of its leading byte.
			switch {
			case c < 0xE0:
				width = 2
			case c < 0xF0:
				width = 3
			default:
				width = 4
			}
			esc = width
		}
		if used+esc > room {
			break
		}
		used += esc
		if c < utf8.RuneSelf && esc != 1 {
			dst = append(dst, src[start:i]...)
			dst = appendEscape(dst, c)
			start = i + 1
		}
		i += width
	}
	return append(dst, src[start:i]...), i
}

// appendFragmentHeader appends to buf everything in a fragment that precedes
// its payload: see the package documentation.
func appendFragmentHeader(buf []byte, seq, total uint64, header []byte) []byte {
	if seq == 0 {
		buf = append(buf, envelopeFirst...)
		buf = strconv.AppendUint(buf, total, 10)
	} else {
		buf = append(buf, envelopePrefix...)
		buf = strconv.AppendUint(buf, seq, 10)
	}
	if len(header) == 0 {
		return append(buf, envelopeMid...)
	}
	buf = append(buf, ',')
	buf = strconv.AppendUint(buf, uint64(len(header)), 10)
	buf = append(buf, envelopeHdrMid...)
	buf = append(buf, header...)
	return append(buf, envelopeHdrEnd...)
}

// writeFragments writes text, a valid message of any length, as fragments:
// see the package documentation for the format and for the argument
// that every fragment fits in the fragment size. The text must be valid
// UTF-8 for it to be sent as a series of JSON strings, and is checked in its
// entirety before anything is written so that a message is never abandoned
// part way through.
func (m *Messager) writeFragments(text, header []byte) error {
	if !utf8.Valid(text) {
		return fmt.Errorf("%w: a message must be valid UTF-8 to be fragmented", ErrInvalidFrame)
	}
	total := uint64(len(text))
	budget := int(m.fragmentSize)
	buf := slices.Grow(m.fragBuf[:0], 4+budget)
	defer func() {
		// Keep the buffer for the next message unless it is large.
		if cap(buf) <= maxRetainedBuffer {
			m.fragBuf = buf
		} else {
			m.fragBuf = nil
		}
	}()

	for off, seq := 0, uint64(0); off < len(text); seq++ {
		// Room for the length, then the header of this fragment, exactly.
		buf = append(buf[:0], 0, 0, 0, 0)
		var h []byte
		if seq == 0 || m.repeatHeader {
			h = header
		}
		buf = appendFragmentHeader(buf, seq, total, h)

		// What remains of the budget, once the header and the closing "}, which
		// are known, are accounted for, is exactly what the payload may use.
		room := budget - (len(buf) - 4) - len(envelopeSuffix)
		var n int
		buf, n = appendEscaped(buf, text[off:], room)
		if n == 0 {
			// Not possible given MinFragmentSize, and so a bug if it happens;
			// fail rather than loop.
			return fmt.Errorf("%w: no progress fragmenting message at offset %d with fragment size %d", ErrInvalidFrame, off, budget)
		}
		off += n
		buf = append(buf, envelopeSuffix...)
		if err := m.writeFrame(buf, uint64(len(buf)-4)); err != nil {
			return err
		}
	}
	return nil
}

// ReadMessage reads a message from the underlying reader, returning a
// Decoder that can be used to decode the message. If the message was
// fragmented, all of its fragments are read and reassembled into a single
// Decoder, see the package documentation. The Decoder must be released by
// calling ReleaseDecoder when no longer needed. ReadMessage will block until
// a complete message is read or an error occurs. io.EOF is returned if the
// stream ends between frames; any other error, including ErrInvalidFrame and
// ErrMessageTooLarge, is permanent: it is returned by all subsequent calls,
// since what follows it in the stream can no longer be told from the
// contents of a frame.
func (m *Messager) ReadMessage() (*Decoder, error) {
	m.rmu.Lock()
	defer m.rmu.Unlock()
	if m.rerr != nil {
		return nil, m.rerr
	}
	if m.rst.active {
		return nil, fmt.Errorf("%w: a message is being forwarded with ReadFragment", ErrInvalidFrame)
	}
	dec, err := m.readMessage()
	if err != nil {
		if err != io.EOF { //nolint:errorlint // io.EOF is returned unwrapped, and only by the first read.
			m.rerr = err
		}
		return nil, err
	}
	return dec, nil
}

// readFull reads len(buf) bytes of a frame, converting a clean EOF, which is
// an error in the middle of a frame, to the error that says so.
func (m *Messager) readFull(buf []byte) error {
	_, err := io.ReadFull(m.rd, buf)
	if err == io.EOF { //nolint:errorlint // ReadFull returns io.EOF itself.
		return io.ErrUnexpectedEOF
	}
	return err
}

// readFrameLen reads the length of the next frame and checks it against the
// limits before anything is read or allocated on the strength of it. At the
// start of a message, atStart, the stream may end cleanly, which is io.EOF.
func (m *Messager) readFrameLen(atStart bool) (int, error) {
	if _, err := io.ReadFull(m.rd, m.hdr[:]); err != nil {
		if err == io.EOF && !atStart { //nolint:errorlint // ReadFull returns io.EOF itself.
			err = io.ErrUnexpectedEOF
		}
		return 0, err
	}
	n := binary.LittleEndian.Uint32(m.hdr[:])
	if n == 0 {
		return 0, fmt.Errorf("%w: empty frame", ErrInvalidFrame)
	}
	if n > m.maxSize {
		return 0, fmt.Errorf("%w: frame size %d exceeds maximum %d", ErrMessageTooLarge, n, m.maxSize)
	}
	return int(n), nil
}

// readBody reads n bytes, the body of a frame, into buf, reusing it if it is
// large enough. Otherwise the buffer is grown as the data arrives, doubling,
// rather than allocated at the size the frame declares, so that a peer cannot
// have this process allocate memory in advance of sending the data to fill it.
func (m *Messager) readBody(buf []byte, n int) ([]byte, error) {
	if cap(buf) >= n {
		buf = buf[:n]
		return buf, m.readFull(buf)
	}
	const firstChunk = 64 * 1024
	buf = buf[:0]
	for len(buf) < n {
		want := min(n-len(buf), max(firstChunk, len(buf)))
		buf = slices.Grow(buf, want)
		end := len(buf) + want
		err := m.readFull(buf[len(buf):end])
		if err != nil {
			return buf, err
		}
		buf = buf[:end]
	}
	return buf, nil
}

func (m *Messager) readMessage() (*Decoder, error) {
	n, err := m.readFrameLen(true)
	if err != nil {
		return nil, err
	}
	dec := m.decPool.Get().(*Decoder)
	dec.header = dec.header[:0]
	dec.buffer, err = m.readBody(dec.buffer, n)
	if err != nil {
		m.ReleaseDecoder(dec)
		return nil, err
	}
	if bytes.HasPrefix(dec.buffer, []byte(envelopePrefix)) {
		if !m.fragmentation {
			m.ReleaseDecoder(dec)
			return nil, fmt.Errorf("%w: a fragment was received but fragmentation is not enabled", ErrInvalidFrame)
		}
		if err := m.reassemble(dec); err != nil {
			m.ReleaseDecoder(dec)
			return nil, err
		}
	}
	*dec.buf = *bytes.NewBuffer(dec.buffer)
	dec.Reset(dec.buf, dec.opts)
	return dec, nil
}

// reassemble reads the remaining fragments of the message that begins with
// the frame in dec.buffer and leaves the message in dec.buffer. Each fragment
// is read into the free space after those already assembled and unescaped
// where it lies, so that no fragment is copied. The user header, if any, is
// copied out first since unescaping overwrites it.
func (m *Messager) reassemble(dec *Decoder) error {
	st := &m.rst
	msg := dec.buffer[:0]
	defer func() { dec.buffer = msg }()
	body := dec.buffer
	for last := false; !last; {
		fh, err := parseFragmentHeader(body, m.maxHeaderSize)
		if err != nil {
			return err
		}
		if err := st.begin(fh.seq, fh.total, fh.hdr, uint64(m.maxFragmentedMessageSize)); err != nil {
			return err
		}
		if fh.seq == 0 {
			dec.header = append(dec.header[:0], st.hdr...)
		}
		w, err := unescapeChunk(body, fh.payloadAt, len(body)-len(envelopeSuffix), 0)
		if err != nil {
			return err
		}
		if last, err = st.add(w); err != nil {
			return err
		}
		msg = msg[:len(msg)+w]
		if last {
			break
		}
		n, err := m.readFrameLen(false)
		if err != nil {
			return err
		}
		// Room for the whole of the next frame, which is at least as large as
		// what it decodes to, after the message so far, but no more than
		// that can ever be needed.
		if need := len(msg) + n; cap(msg) < need {
			grown := make([]byte, len(msg), min(max(need, 2*cap(msg)), int(st.total)+int(m.maxSize)))
			copy(grown, msg)
			msg = grown
		}
		body = msg[len(msg) : len(msg)+n]
		if err := m.readFull(body); err != nil {
			return err
		}
	}
	return nil
}

// parseUint parses the decimal number that starts at b[i], with no sign and no
// leading zeros as JSON requires, and of at most maxDigits digits, returning
// it and the index of the byte after it.
func parseUint(b []byte, i int) (v uint64, next int, ok bool) {
	start := i
	for i < len(b) && b[i] >= '0' && b[i] <= '9' {
		if i-start >= maxDigits {
			return 0, 0, false
		}
		v = v*10 + uint64(b[i]-'0')
		i++
	}
	if i == start || (b[start] == '0' && i-start > 1) {
		return 0, 0, false
	}
	return v, i, true
}

// fragmentHeader is what precedes the payload of a fragment.
type fragmentHeader struct {
	seq, total uint64 // total is zero except for the first fragment
	hdr        []byte // the user header, a slice of the fragment, or nil
	payloadAt  int    // the index at which the escaped payload begins
}

// parseFragmentHeader parses the header of the fragment in body, which is
// known to begin with envelopePrefix, and checks the rest of its form, which
// is fixed. The payload ends 2 bytes before the end of body. No variation of
// the form is accepted: see the package documentation. A user header is
// accepted only if maxHeader is not zero and it is not larger than that.
func parseFragmentHeader(body []byte, maxHeader uint32) (fh fragmentHeader, err error) {
	bad := func(what string) (fragmentHeader, error) {
		return fragmentHeader{}, fmt.Errorf("%w: malformed fragment: %s", ErrInvalidFrame, what)
	}
	if !bytes.HasPrefix(body, []byte(envelopePrefix)) {
		return bad("does not begin with " + envelopePrefix)
	}
	seq, i, ok := parseUint(body, len(envelopePrefix))
	if !ok {
		return bad("invalid sequence number")
	}
	fh.seq = seq
	if seq == 0 {
		if i >= len(body) || body[i] != ',' {
			return bad("the first fragment has no total")
		}
		fh.total, i, ok = parseUint(body, i+1)
		if !ok || fh.total == 0 {
			return bad("invalid total")
		}
	}
	if fh.hdr, i, err = parseUserHeader(body, i, maxHeader); err != nil {
		return fragmentHeader{}, err
	}
	// A payload is at least one byte, then the closing "}.
	if len(body)-i < 1+len(envelopeSuffix) || !bytes.HasSuffix(body, []byte(envelopeSuffix)) {
		return bad("empty payload or missing closing")
	}
	fh.payloadAt = i
	return fh, nil
}

// parseUserHeader parses what follows the numbers of a fragment's header,
// which begins at body[i]: either the end of the header and the start of the
// payload, or the length of a user header, then the header, then the start of
// the payload. It returns the user header, or nil, and the index of the start
// of the payload.
func parseUserHeader(body []byte, i int, maxHeader uint32) (hdr []byte, payloadAt int, err error) {
	bad := func(what string) ([]byte, int, error) {
		return nil, 0, fmt.Errorf("%w: malformed fragment: %s", ErrInvalidFrame, what)
	}
	if i >= len(body) || body[i] != ',' {
		if !bytes.HasPrefix(body[i:], []byte(envelopeMid)) {
			return bad("invalid header")
		}
		return nil, i + len(envelopeMid), nil
	}
	if maxHeader == 0 {
		return bad("a user header was received but they are not enabled")
	}
	hlen, j, ok := parseUint(body, i+1)
	if !ok || hlen == 0 || hlen > uint64(maxHeader) {
		return bad("invalid user header length")
	}
	if !bytes.HasPrefix(body[j:], []byte(envelopeHdrMid)) {
		return bad("invalid header")
	}
	j += len(envelopeHdrMid)
	if uint64(len(body)-j) < hlen {
		return bad("user header cut short")
	}
	hdr = body[j : j+int(hlen)]
	j += int(hlen)
	if !bytes.HasPrefix(body[j:], []byte(envelopeHdrEnd)) {
		return bad("invalid header")
	}
	if !jsontext.Value(hdr).IsValid() {
		return bad("user header is not valid JSON")
	}
	return hdr, j + len(envelopeHdrEnd), nil
}

// hexValue returns the value of the hex digit c, or -1.
func hexValue(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	}
	return -1
}

// hex4 returns the value of the 4 hex digits at b[i:], or -1.
func hex4(b []byte, i int) rune {
	if i+4 > len(b) {
		return -1
	}
	var v rune
	for _, c := range b[i : i+4] {
		d := hexValue(c)
		if d < 0 {
			return -1
		}
		v = v<<4 | rune(d)
	}
	return v
}

// unescapeChunk decodes the contents of the JSON string at buf[from:to] and
// writes the result to buf at w, returning the index after what was written.
// The destination may be, and is, the same slice as the source, with w no
// greater than from: decoding never lengthens the data, so what is written is
// never ahead of what is read, and the data is decoded where it lies. It
// accepts every form of escape that a JSON encoder may produce, and rejects
// anything that is not a valid JSON string or is not valid UTF-8 once decoded
// (including a surrogate that is not part of a pair), a quote or control
// character that is not escaped, and an escape that is cut short.
func unescapeChunk(buf []byte, from, to, w int) (int, error) {
	start := w
	// The common case, no escapes: validate and move.
	plain := true
	for _, c := range buf[from:to] {
		if c < 0x20 {
			return 0, errPayload("unescaped control character")
		}
		if c == '\\' {
			plain = false
		}
		// A quote is escaped, and so preceded by a backslash, or is an error.
		if c == '"' && plain {
			return 0, errPayload("unescaped quote")
		}
	}
	if plain {
		w += copy(buf[w:], buf[from:to])
	} else {
		var err error
		if w, err = unescapeEscapes(buf, from, to, w); err != nil {
			return 0, err
		}
	}
	if !utf8.Valid(buf[start:w]) {
		return 0, errPayload("not valid UTF-8")
	}
	return w, nil
}

func errPayload(what string) error {
	return fmt.Errorf("%w: malformed fragment payload: %s", ErrInvalidFrame, what)
}

// simpleEscapes maps the character following a backslash to the character it
// stands for, for the escapes that are a single character, and is zero for
// any other.
var simpleEscapes = func() (t [256]byte) {
	for k, v := range map[byte]byte{'"': '"', '\\': '\\', '/': '/', 'b': '\b', 'f': '\f', 'n': '\n', 'r': '\r', 't': '\t'} {
		t[k] = v
	}
	return t
}()

// unescapeEscapes is unescapeChunk for a payload that contains a backslash.
func unescapeEscapes(buf []byte, from, to, w int) (int, error) {
	for r := from; r < to; {
		c := buf[r]
		if c == '"' {
			return 0, errPayload("unescaped quote")
		}
		if c != '\\' {
			buf[w] = c
			w++
			r++
			continue
		}
		r++
		if r >= to {
			return 0, errPayload("escape cut short")
		}
		if e := simpleEscapes[buf[r]]; e != 0 {
			buf[w] = e
			w++
			r++
			continue
		}
		if buf[r] != 'u' {
			return 0, errPayload("invalid escape")
		}
		v, next, err := decodeUnicodeEscape(buf[:to], r+1)
		if err != nil {
			return 0, err
		}
		w += utf8.EncodeRune(buf[w:], v)
		r = next
	}
	return w, nil
}

// decodeUnicodeEscape decodes the hex digits of a \u escape that begin at b[i],
// ie. just after the u, and, if they are the first half of a surrogate pair,
// the second half that must follow. It returns the character and the index
// of the byte after the escape or pair.
func decodeUnicodeEscape(b []byte, i int) (rune, int, error) {
	bad := func(what string) (rune, int, error) {
		return 0, 0, errPayload(what)
	}
	v := hex4(b, i)
	if v < 0 {
		return bad("invalid \\u escape")
	}
	i += 4
	switch {
	case v >= 0xDC00 && v <= 0xDFFF:
		return bad("unpaired low surrogate")
	case v >= 0xD800 && v <= 0xDBFF:
		// The first half of a pair; the second must follow.
		if i+2 >= len(b) || b[i] != '\\' || b[i+1] != 'u' {
			return bad("unpaired high surrogate")
		}
		lo := hex4(b, i+2)
		if lo < 0xDC00 || lo > 0xDFFF {
			return bad("unpaired high surrogate")
		}
		return 0x10000 + (v-0xD800)<<10 + (lo - 0xDC00), i + 6, nil
	}
	return v, i, nil
}
