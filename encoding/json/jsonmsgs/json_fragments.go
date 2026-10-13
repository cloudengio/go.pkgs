// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"unicode/utf8"
)

// fragState tracks the fragments of the message that is being read, a frame at
// a time, and holds the rules that fragments must follow, see the package
// documentation. ReadMessage and ReadFragments both use it so that they accept
// and reject the same streams.
type fragState struct {
	active bool   // a message has begun and has not yet been completed
	next   uint64 // the sequence number of the next fragment
	got    uint64 // the bytes of the message received so far
	total  uint64 // the bytes of the message, as declared by its first fragment
	single bool   // fragmentation is not enabled: a message must be in a single envelope
}

// begin checks the sequence number of the next fragment of a message against
// that of the message so far, or the total of the first fragment against the
// limit on the size of a message, before anything is allocated for it.
func (s *fragState) begin(seq, total, maxTotal uint64) error {
	if !s.active {
		if seq != 0 {
			return fmt.Errorf("%w: the first fragment has sequence number %d, not 0", ErrInvalidFrame, seq)
		}
		if total > maxTotal {
			return fmt.Errorf("%w: message size %d exceeds maximum message size %d", ErrMessageTooLarge, total, maxTotal)
		}
		s.active, s.next, s.got, s.total = true, 0, 0, total
		return nil
	}
	if seq != s.next {
		return fmt.Errorf("%w: fragment %d is out of order, expected %d", ErrInvalidFrame, seq, s.next)
	}
	return nil
}

// add accounts for a fragment, accepted by begin, that carries n bytes of the
// message, and reports whether it completes the message.
func (s *fragState) add(n int) (last bool, err error) {
	if s.got+uint64(n) > s.total {
		return false, fmt.Errorf("%w: fragments add up to more than the declared %d bytes", ErrInvalidFrame, s.total)
	}
	s.got += uint64(n)
	s.next++
	if s.single && s.got < s.total {
		return false, fmt.Errorf("%w: a message in more than one envelope requires fragmentation to be enabled", ErrInvalidFrame)
	}
	if s.got == s.total {
		s.active = false
		return true, nil
	}
	return false, nil
}

// Fragments are the frames that make up a single message, as read by
// Reader.ReadFragments and written by Writer.WriteFragments. Fragments
// is intended for use by message routers that do not need access to the
// body of the message, just the metadata.
type Fragments struct {
	bare    bool   // payload is the message itself, not escaped, in a single frame
	total   int    // the size of the message, once unescaped
	header  []byte // the user header, or nil
	payload []byte // the payloads of the frames, escaped, one after the other, or the message if bare
	ends    []int  // the end of the payload of each frame in payload
}

// reset empties f for reading into, keeping its memory unless it is larger than
// retain.
func (f *Fragments) reset(retain int) {
	f.bare, f.total = false, 0
	f.header = f.header[:0]
	f.payload = f.payload[:0]
	f.ends = f.ends[:0]
	if cap(f.payload) > retain {
		f.payload = nil
	}
	if cap(f.header) > retain {
		f.header = nil
	}
}

// valid reports whether f holds a message.
func (f *Fragments) valid() bool {
	return len(f.payload) > 0 && len(f.ends) > 0 && f.ends[len(f.ends)-1] == len(f.payload) && f.total > 0
}

// Header returns the user header that was sent with the message, see
// Writer.WriteMessageWithHeader, or nil if it was sent without one. It is valid
// until f is next used.
func (f *Fragments) Header() jsontext.Value {
	if f == nil || len(f.header) == 0 {
		return nil
	}
	return f.header
}

// Len returns the size in bytes of the message once the frames are reassembled
// and unescaped, ie. the size that Unescape of Payload gives, or that of
// Payload if the message is Bare.
func (f *Fragments) Len() int {
	if f == nil {
		return 0
	}
	return f.total
}

// Bare reports whether Payload is the message itself, and not escaped, as it is
// if the message was read from a frame that was not an envelope, or was created
// using Set.
func (f *Fragments) Bare() bool {
	if f == nil {
		return false
	}
	return f.bare
}

// Frames returns the number of frames that the message was read from.
func (f *Fragments) Frames() int {
	if f == nil {
		return 0
	}
	return len(f.ends)
}

// Payload returns the payloads of the frames one after the other, which are the
// contents of JSON strings that Unescape decodes to the message. If the message
// is Bare it is the message, and is not escaped. It must not be modified, and is
// valid until f is next used.
func (f *Fragments) Payload() []byte {
	if f == nil {
		return nil
	}
	return f.payload
}

// NewFragments returns Fragments for a message that is created, rather than
// read, see Set.
func NewFragments(header, message any) (*Fragments, error) {
	f := &Fragments{}
	if err := f.Set(header, message); err != nil {
		return nil, err
	}
	return f, nil
}

// Set replaces what f holds with the user header to send with a message, if there
// is one, and the message, that it marshals as JSON, so that a router, say, can
// respond to a message that is addressed to it using Writer.WriteFragments, as it
// forwards the messages that are not, without encoding the message itself. Each
// is marshalled as the standard library's json.Marshal does, other than that
// the characters <, > and & are not escaped, and so can be of any type that it
// can marshal, including a jsontext.Value or json.RawMessage that holds JSON,
// but not a []byte, which is marshalled as a base64 string. The header is nil if
// there is none. As both are of type any, the order of header and message is not
// checked by the compiler. It is an error, that wraps ErrInvalidFrame and the error from
// marshalling, if either cannot be marshalled, or if the message is nil. If there
// is an error f is empty. The message is not checked against the limits of a
// Writer, which are those of WriteFragments. f is reused, like
// the Fragments that are passed to ReadFragments, and can be written, as can
// any other, to any number of Writers.
func (f *Fragments) Set(header, message any) error {
	f.reset(DefaultMaxRetainedBuffer)
	if message == nil {
		return fmt.Errorf("%w: no message", ErrInvalidFrame)
	}
	var err error
	if f.payload, err = marshalJSON(f.payload, message); err != nil {
		f.reset(DefaultMaxRetainedBuffer)
		return fmt.Errorf("%w: marshalling the message: %w", ErrInvalidFrame, err)
	}
	if len(f.payload) == 0 {
		f.reset(DefaultMaxRetainedBuffer)
		return fmt.Errorf("%w: empty message", ErrInvalidFrame)
	}
	if header != nil {
		if f.header, err = marshalJSON(f.header, header); err != nil {
			f.reset(DefaultMaxRetainedBuffer)
			return fmt.Errorf("%w: marshalling the header: %w", ErrInvalidFrame, err)
		}
		if len(f.header) == 0 {
			f.reset(DefaultMaxRetainedBuffer)
			return fmt.Errorf("%w: empty header", ErrInvalidFrame)
		}
	}
	// A type that marshals itself can return what is not valid UTF-8.
	if !utf8.Valid(f.payload) || !utf8.Valid(f.header) {
		f.reset(DefaultMaxRetainedBuffer)
		return fmt.Errorf("%w: the message and header must be valid UTF-8", ErrInvalidFrame)
	}
	f.bare, f.total = true, len(f.payload)
	f.ends = append(f.ends[:0], len(f.payload))
	return nil
}

// marshalJSON appends the JSON for v, without a newline, to dst[:0].
func marshalJSON(dst []byte, v any) ([]byte, error) {
	buf := bytes.NewBuffer(dst[:0])
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return dst[:0], err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// Reader returns an io.Reader for the message, with the escapes in the payloads
// decoded as they are read, which is for a router that is the destination of a
// message and not merely forwarding it, to decode the message, using a
// jsontext.Decoder or the standard library's, for example, without first
// assembling it in memory. Frames are decoded one at a time, into the buffer
// that Read is given if it is large enough and otherwise into one that the Reader
// keeps, which is no larger than a frame. A message that is Bare is read as it
// is. The Reader is valid until f is next used, and several may be in use at a
// time, but each by one goroutine.
func (f *Fragments) Reader() io.Reader {
	if f == nil {
		return bytes.NewReader(nil)
	}
	return &fragmentsReader{f: f}
}

// fragmentsReader is the io.Reader for Fragments.
type fragmentsReader struct {
	f       *Fragments
	frame   int    // the next frame to be decoded
	start   int    // where its payload begins
	pending []byte // what has been decoded and not read
	scratch []byte // for a frame that Read's buffer cannot hold
}

// Read implements io.Reader.
func (r *fragmentsReader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		if len(r.pending) > 0 {
			c := copy(p[n:], r.pending)
			r.pending = r.pending[c:]
			n += c
			continue
		}
		if r.frame >= len(r.f.ends) {
			break
		}
		end := r.f.ends[r.frame]
		chunk := r.f.payload[r.start:end]
		r.frame++
		r.start = end
		switch {
		case r.f.bare:
			r.pending = chunk
		case len(p)-n >= len(chunk):
			// Decoding never lengthens what it decodes, and so the frame is
			// decoded where it is to be read.
			out, err := Unescape(p[n:n], chunk)
			if err != nil {
				return n, err
			}
			n += len(out)
		default:
			var err error
			if r.scratch, err = Unescape(r.scratch[:0], chunk); err != nil {
				return n, err
			}
			r.pending = r.scratch
		}
	}
	if n == 0 && len(p) > 0 {
		return 0, io.EOF
	}
	return n, nil
}

// Unescape appends the bytes of the message that payload stands for to dst and
// returns the extended slice. payload is the contents of a JSON string, as the
// Payload of Fragments is, and is decoded as described under "Escaping" in the
// package documentation, accepting every form of escape that a JSON encoder
// may produce. This is for code that needs to look at the message that
// Fragments carry, for example to route on it; the Payload of Fragments that
// are Bare is already the message and is not escaped. An error that wraps
// ErrInvalidFrame is returned if payload is not the valid contents of a JSON
// string, or is not valid UTF-8 once decoded, in which case the length of dst
// is unchanged. payload is not modified.
func Unescape(dst, payload []byte) ([]byte, error) {
	start := len(dst)
	// unescapeChunk decodes in place, so that the payload is first copied to
	// where the result is to be, which is no more than that.
	dst = append(dst, payload...)
	end, err := unescapeChunk(dst, start, len(dst), start)
	if err != nil {
		return dst[:start], err
	}
	return dst[:end], nil
}
