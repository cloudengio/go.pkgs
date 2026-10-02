// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package jsonmsgs provides support for the efficient encoding and decoding of
// arbitrary JSON messages over a stream. It uses the same framing as browser
// native messaging since that is both minimal (4 bytes of length + payload)
// and can be used to communicate with browser extensions. Different
// browsers and other environments impose different limits on message sizes and
// hence support for fragmentation and reassembly is useful to avoid the
// need for users of this package to vary their behaviour depending on which
// environments they are running in. In addition, support is provided for
// efficiently forwarding/proxying messages through intermediaries.
// Support for proxies takes the form of a user supplied header that can be
// decoded independently of the payload to avoid the unnecessary overhead of
// decoding the entire payload. A payload can be opaque to intermediaries,
// for example if it is encrypted by the sender. Note that a message that is
// sent in an envelope must be valid UTF-8 and hence a sender that encrypts must
// encode its ciphertext appropriately. The user header is always visible to
// intermediaries.
//
// # Reader and Writer
//
// A Reader reads messages and a Writer writes them, each with its own options,
// so that the two directions of a connection, or the two sides of a proxy, can
// have different limits, in particular different frame sizes: a Reader accepts
// frames of up to its maximum size, see WithMaxSize, whatever the fragment size
// that the Writer at the other end of the stream uses, and a Writer writes
// frames of at most its own size. Reader and Writer accept the same Options,
// and ignore those that do not apply to them; the documentation of each
// option says which it applies to. A Messager is a Reader and a Writer with the
// same options, for convenience.
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
// Both frames and messages must be valid JSON since a browser parses every
// frame it receives as a single JSON value. Fragmentation is therefore
// done inside the JSON, with the fragments of a message carried as JSON strings,
// and must be UTF-8 aware to avoid splitting runes.
//
// # Wire format
//
// The body of a frame is one of two things, a bare message or an envelope,
// and which it is is determined by its first bytes:
//
//	frame    = length body
//	length   = the number of bytes in body, as 4 bytes, little endian
//	body     = bare | envelope
//	bare     = a complete JSON value that does not begin with `{"~":[`
//	envelope = `{"~":[` <seq> [ `,` <total> ] [ `,` <hlen> ] `]` [ `,"h":` <header> ] `,"p":"` <payload> `"}`
//
// A bare message is the message and nothing else. An envelope is a JSON object
// of exactly the form above, in that order, with no white space, and with up to
// three members: "~", an array of numbers that describes the envelope, "h", the
// user header, if there is one, and "p", the payload. The names in the grammar
// are not members: they stand for the elements of the "~" array, in order, and
// for the values of "h" and "p":
//
//	seq      "~" element: the sequence number of the envelope within its
//	         message, from 0
//	total    "~" element, only if seq is 0: the length in bytes of the whole
//	         message
//	hlen     "~" element, only if there is a user header: its length in bytes
//	header   the value of "h", only if there is a user header: a JSON value,
//	         see "User header"
//	payload  the value of "p": a chunk of the message, as the contents of a
//	         JSON string, escaped as described under "Escaping"; it is never
//	         empty, and, if it is that of the first envelope and is total
//	         bytes long once unescaped, is the whole message
//
// All numbers are decimal, with no sign or leading zeros, and of at most 10
// digits. There is deliberately no member saying whether more envelopes follow,
// no message identifier and no version: the last envelope of a message is the
// one that brings the bytes received up to total, a single message is in
// flight in each direction at a time, and a future version of this format
// would use a key other than "~". An envelope costs 17 bytes plus the digits
// of total on the first envelope of a message and 15 bytes plus the digits of
// seq on the others, plus the user header and its length if there is one,
// which is of no consequence when a frame is of the order of a megabyte; it is
// the encoding of the payload that determines the size of a message.
//
// # Fragmentation
//
// If fragmentation is enabled (see WithFragmentation) a message that does not
// fit in one frame is sent in a series of envelopes, each carrying a
// fragment, a chunk, of it, as in these examples, neither of which has a user
// header and only the first of which has a total:
//
//	{"~":[0,61],"p":"{\"key\":\"a value long enough to need fragmenting"}
//	{"~":[1],"p":"\",\"n\":[1,2,3]}"}
//
// The payloads of the envelopes, once unescaped and concatenated, are the
// original message exactly. A message that fits in a frame is sent bare unless
// it has a user header, see below, or begins with the bytes that mark an
// envelope, {"~":[. Neither side ever ignores or reinterprets any other bytes.
// A message that would fit into a bare frame, but coincidentally begins with
// the bytes that mark an envelope, is sent in a single envelope, whether or not
// fragmentation is enabled, and the reader unwraps it. Envelopes are
// independent of fragmentation: fragmentation controls only whether a message
// that does not fit in a frame may be split across envelopes. Without it a
// message must fit in a single envelope. A reader returns ErrMessageTooLarge
// for a message that declares a total larger than the maximum size, and
// ErrInvalidFrame for one that is in more than one envelope but is not too
// large, since it is the peer's use of fragmentation that is not accepted.
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
// A frame is never larger than the fragment size, see WithFragmentSize, or than
// the maximum size, see WithMaxSize, if fragmentation is not enabled, and this
// is enforced while the frame is built. The writer first writes everything
// that precedes the payload, that is the "~" array and the user header, if any,
// which are of known size since the sequence number, total and length of the
// header are known, and sets aside the 2 bytes of the closing "} to leave room
// for the payload. It then takes the message one rune at a time, computes the
// exact length of that rune once escaped, and stops at the first rune that
// would not fit, emitting a rune only if it, and so its escape sequence, fits
// entirely. The frame is therefore within the budget for any message, including
// one whose characters all escape to 6 bytes, and is as full as it can be,
// since it ends only when the next rune does not fit. Every envelope holds at
// least one rune, so the message is always consumed, provided the fragment size
// is at least MinFragmentSizeWithHeader, which NewWriter requires. Each
// envelope is built in a single buffer and is written with a single call to
// Write. Without fragmentation the message must be in a single envelope and
// ErrMessageTooLarge is returned if it does not fit, which is known before
// anything is written.
//
// # User header
//
// Code that sits above this package may need to route a message without decoding
// or reassembling it. The user header is intended to be used for this purpose,
// see WriteMessageWithHeader. It is any JSON value, that is opaque to this
// package, and is carried as the "h" member of an envelope, with its length in
// bytes as the last element of the "~" array:
//
//	{"~":[0,61,17],"h":{"to":7,"op":"x"},"p":"{\"key\":\"a value long enough"}
//	{"~":[1],"p":"\",\"n\":[1,2,3]}"}
//
// When fragmentation is enabled the header is part of the envelope that carries
// the first chunk of the message, and not an envelope of its own, and is
// optionally repeated in subsequent envelopes, see WithRepeatHeader, as
// {"~":[1,17],"h":{"to":7,"op":"x"},"p":"…"}. If fragmentation is not enabled
// the message is sent in a single envelope which, with its overhead, the user
// header and the escaped payload, must fit in the maximum size, see
// WithMaxSize.
//
// User headers are disabled unless WithMaxHeaderSize is set, and a header
// received when they are disabled is an error.
//
// # Forwarding
//
// A router, or proxy, that moves messages between connections needs neither to
// hold the whole of a message nor to decode it. ReadFragment returns a frame
// as it is, a Fragment, with the sequence number, the total on the first, the
// user header and the payload still escaped, along with the number of bytes
// that it stands for once unescaped and whether it is the last of its message.
// WriteFragment writes a received message using a Writer. This requires
// that the next hop fragment size is at least as large as that of the
// received message. In the case where the next hop has a smaller max fragment
// size than the receiver, the only option is for the proxy to buffer the
// whole message in order to refragment it according to the next hop's maximum
// fragment size.
//
// # Reading
//
// ReadMessage returns a complete message: a fragmented message is reassembled
// before it is returned, and its user header, if any, is available from
// Decoder.Header.
//
// # Errors
//
// An error that matches ErrPermanent, tested with errors.Is, means that the
// Reader or Writer that returned it can no longer be used and the connection
// should be closed: all later calls return the same error. This is the case for
// any error reading, other than io.EOF between messages, since the position in
// the stream is then not known, and for an error writing to the underlying
// writer, since a partially written frame or message cannot be completed or
// undone. Any other error, such as ErrMessageTooLarge or ErrInvalidFrame for a
// message to be written, is found before anything is read or written, leaves
// the Reader or Writer as it was, and may be retried, with another message for
// example.
//
// When forwarding, an error writing a frame that is not the first of its
// message is also permanent, since the peer has been sent the start of a message
// that can no longer be completed, but an error in the first frame of a message,
// or in a bare frame, can be retried with a corrected Fragment. Reading with
// ReadMessage part way through a message that is being read with ReadFragment,
// or writing with WriteMessage part way through one that is being written with
// WriteFragment, fails with ErrInvalidFrame, which can be retried once the
// message is complete.
//
// # Browser extension
//
// The extension must reassemble fragmented messages that it receives from the
// host (the limit on those is 1 MB, it is the direction in which it is needed),
// and may send them, but need not, since the limit to the host is much higher.
// It must also unwrap an envelope that holds a whole message, which is how a
// message with a user header, or one that begins with the bytes that mark an
// envelope, is sent. A reassembler in JavaScript is a few lines; see
// testdata/fragments.js, which is also used to test this package against an
// independent implementation:
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
// The user header, if any, is m.h of the first envelope, and of the others if
// the host repeats it. A sender in JavaScript puts it in the object, between
// "~" and "p", and adds its length in bytes, JSON.stringify(h) encoded as
// UTF-8, to the "~" array of each envelope that carries it.
//
// An extension sees only the parsed JSON of a frame and so can tell an envelope
// from an ordinary message only by its structure, as above, whereas the Go
// reader tests for the exact bytes {"~":[. The two agree on everything that
// this package writes, since a message that begins with those bytes is always
// sent in an envelope, but not on an ordinary message that has the same
// structure written differently, such as { "~": [1] }, which a Go writer sends
// bare and an extension would take for an envelope. Messages sent to an
// extension should therefore not have a top-level "~" member. A JavaScript
// writer must split on code points and not UTF-16 code units so that a
// surrogate pair is never divided between two fragments.
package jsonmsgs

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"
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
// one very large message does not pin that memory for the life of the Reader
// or Writer.
const maxRetainedBuffer = 4 * 1024 * 1024

// The pieces of an envelope, see the package documentation. An envelope is
// envelopePrefix, or envelopeFirst for the first which includes its sequence
// number of 0, the sequence number and, for the first, the total, and then
// either envelopeMid, or, if there is a user header, a "," and the length of the
// header followed by envelopeHdrMid, the header and envelopeHdrEnd, and finally
// the escaped payload and envelopeSuffix.
const (
	envelopePrefix = `{"~":[`
	envelopeFirst  = `{"~":[0,`
	envelopeMid    = `],"p":"` // ends the "~" array if there is no user header
	envelopeHdrMid = `],"h":`  // ends the "~" array and begins the user header
	envelopeHdrEnd = `,"p":"`  // ends the user header
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
	// retried, with a different message or Fragment, or later. io.EOF, returned
	// between messages, is neither.
	ErrPermanent = errors.New("jsonmsgs: permanent error")
)

// permanentError marks the error that it wraps as one that matches
// ErrPermanent, without changing its text.
type permanentError struct{ err error }

func (e *permanentError) Error() string        { return e.err.Error() }
func (e *permanentError) Unwrap() error        { return e.err }
func (e *permanentError) Is(target error) bool { return target == ErrPermanent }

// permanent returns err marked as permanent, unless it already is.
func permanent(err error) error {
	if err == nil || errors.Is(err, ErrPermanent) {
		return err
	}
	return &permanentError{err}
}

// MinFragmentSize returns the smallest fragment size, see WithFragmentSize,
// with which a message of up to maxFragmentedMessageSize bytes can always be
// fragmented. It is the largest header that such a message needs, the 2 bytes
// that close a fragment and the longest escape sequence, so that every
// fragment can carry at least one character. It is MinFragmentSizeWithHeader
// with a maximum header size of 0, ie. for messages that have no user header.
func MinFragmentSize(maxFragmentedMessageSize uint32) uint32 {
	return MinFragmentSizeWithHeader(maxFragmentedMessageSize, 0)
}

// MinFragmentSizeWithHeader is MinFragmentSize for a Writer that sends user
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

// Option represents an option for configuring a Reader or a Writer, which
// ignore the options that do not apply to them: a Reader uses WithMaxSize,
// WithMaxFragmentedMessageSize, WithFragmentation, WithMaxHeaderSize and
// WithDecoderOptions; a Writer uses WithMaxSize, WithFragmentSize,
// WithMaxFragmentedMessageSize, WithFragmentation, WithMaxHeaderSize,
// WithRepeatHeader and WithEncoderOptions.
type Option func(*options)

// WithMaxSize sets the maximum size of a single frame in bytes: no frame
// larger is accepted by a Reader, which rejects it on seeing its length, and,
// if fragmentation is not enabled, no frame larger is written by a Writer, so
// that a message that is sent in an envelope, because it has a user header for
// example, must be smaller by the size of the envelope. If maxSize is 0,
// DefaultMaxNativeMessageSize is used. A Reader's should be at least as large as
// the largest frame a peer will send; the limit of a browser on frames to the
// host is much larger than that on frames from it. NewReader and NewWriter
// panic if maxSize is less than 2 or exceeds math.MaxInt32.
func WithMaxSize(maxSize uint32) Option {
	return func(opts *options) {
		opts.maxSize = maxSize
	}
}

// WithFragmentSize sets the maximum size in bytes of a frame written by a
// Writer when fragmentation is enabled, including a message sent whole: a
// message that does not fit is fragmented into frames of at most this size. It
// defaults to the maximum size, and is set to it if it is 0 or larger than it.
// It is ignored if fragmentation is not enabled, when the maximum size is the
// limit, and by a Reader, which accepts frames of any size up to its maximum.
// If fragmentation is enabled, NewWriter panics if fragmentSize is less than
// MinFragmentSizeWithHeader for the maximum fragmented message size and header
// size.
func WithFragmentSize(fragmentSize uint32) Option {
	return func(opts *options) {
		opts.fragmentSize = fragmentSize
	}
}

// WithMaxFragmentedMessageSize sets the maximum total size of a message that can
// be fragmented in bytes, which is to say the largest that can be written and
// that will be accepted when reassembled. If 0, DefaultMaxFragmentedMessageSize
// (16MB) is used. It applies only if fragmentation is enabled, otherwise a
// message is limited by the maximum size, see WithMaxSize. A Writer returns
// ErrMessageTooLarge for a message that exceeds it and a Reader for one that
// declares a total that does, before reading any of it. It protects receivers
// from unbounded memory growth, and prevents deadlocks when writing over
// buffered channels that could fill up before a complete message is sent.
// NewReader and NewWriter panic if it exceeds math.MaxInt32.
func WithMaxFragmentedMessageSize(maxSize uint32) Option {
	return func(opts *options) {
		opts.maxFragmentedMessageSize = maxSize
	}
}

// WithFragmentation controls whether automatic message fragmentation is enabled.
// The default is false. When false, a Writer returns ErrMessageTooLarge if a
// message exceeds the maximum size, and a Reader returns an error if a message
// in more than one envelope is received. When true, messages exceeding the
// fragment size (which defaults to the maximum size) are automatically
// fragmented into multiple frames. The Reader and the Writer at the two ends
// of a stream must both have it enabled for fragmented messages to be
// exchanged, but a message that is not fragmented is the same either way. It
// does not affect the use of envelopes, which carry a user header, see
// WithMaxHeaderSize, or a message that begins with the bytes that mark one: when
// disabled such a message must fit in a single envelope.
func WithFragmentation(enable bool) Option {
	return func(opts *options) {
		opts.fragmentation = enable
	}
}

// WithMaxHeaderSize enables user headers, see WriteMessageWithHeader, of up to
// maxHeaderSize bytes, which is also the largest accepted by a Reader: a frame
// with a header is an error unless this option is set.
// It does not require fragmentation, in which case a message with a header must
// fit in a single envelope, and if fragmentation is enabled it increases the
// smallest fragment size that NewWriter accepts, see
// MinFragmentSizeWithHeader. NewReader and NewWriter panic if maxHeaderSize
// exceeds math.MaxInt32.
func WithMaxHeaderSize(maxHeaderSize uint32) Option {
	return func(opts *options) {
		opts.maxHeaderSize = maxHeaderSize
	}
}

// WithRepeatHeader sends the user header of a message on each of its
// fragments rather than only on the first, so that a receiver can route a
// fragment without having seen the first. It costs the size of the header, and
// of its length, in each fragment. It has no effect if fragmentation is not
// enabled, since a message is then in a single envelope.
func WithRepeatHeader(repeat bool) Option {
	return func(opts *options) {
		opts.repeatHeader = repeat
	}
}

// WithEncoderOptions sets the options for the encoders returned by
// Writer.NewEncoder.
func WithEncoderOptions(opts jsontext.Options) Option {
	return func(o *options) {
		o.encoderOptions = opts
	}
}

// WithDecoderOptions sets the options for the decoders returned by
// Reader.ReadMessage.
func WithDecoderOptions(opts jsontext.Options) Option {
	return func(o *options) {
		o.decoderOptions = opts
	}
}

// resolveOptions applies opts and checks and defaults the settings that Reader
// and Writer share.
func resolveOptions(opts []Option) options {
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
	return o
}

// Messager is a Reader and a Writer, for convenience when the same options
// apply to both directions of a connection: NewMessager(rd, wr, opts...) is
// NewReader(rd, opts...) and NewWriter(wr, opts...), and its methods are theirs.
// Use a Reader and a Writer when each direction needs its own options, for
// example a proxy that reads frames of one size and writes those of another.
type Messager struct {
	*Reader
	*Writer
}

// NewMessager creates a Messager that reads from rd and writes to wr, see
// NewReader and NewWriter.
func NewMessager(rd io.ReadCloser, wr io.Writer, opts ...Option) *Messager {
	return &Messager{NewReader(rd, opts...), NewWriter(wr, opts...)}
}
