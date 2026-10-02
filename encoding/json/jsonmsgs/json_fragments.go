// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs

import (
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"
)

// fragState tracks the fragments of the message that is being read, or
// written, a frame at a time, and holds the rules that fragments must follow
// whichever way they travel, see the package documentation. The reassembling
// reader, ReadMessage, and the forwarding reader and writer, ReadFragment and
// WriteFragment, all use it, so that they accept and reject the same streams.
type fragState struct {
	active bool   // a message has begun and has not yet been completed
	next   uint64 // the sequence number of the next fragment
	got    uint64 // the bytes of the message received so far
	total  uint64 // the bytes of the message, as declared by its first fragment
	hdr    []byte // the user header of the first fragment
}

// begin checks the sequence number, total and user header of the next fragment
// of a message against those of the message so far, or of the first fragment
// against the limit on the size of a message, before anything is allocated
// for it. hdr may be the user header of the message or empty: a header that
// is repeated must be that of the first fragment.
func (s *fragState) begin(seq, total uint64, hdr []byte, maxTotal uint64) error {
	if !s.active {
		if seq != 0 {
			return fmt.Errorf("%w: the first fragment has sequence number %d, not 0", ErrInvalidFrame, seq)
		}
		if total > maxTotal {
			return fmt.Errorf("%w: message size %d exceeds maximum fragmented message size %d", ErrMessageTooLarge, total, maxTotal)
		}
		s.active, s.next, s.got, s.total = true, 0, 0, total
		s.hdr = append(s.hdr[:0], hdr...)
		return nil
	}
	if seq != s.next {
		return fmt.Errorf("%w: fragment %d is out of order, expected %d", ErrInvalidFrame, seq, s.next)
	}
	if len(hdr) != 0 && !bytes.Equal(hdr, s.hdr) {
		return fmt.Errorf("%w: fragment %d has a different user header to the first", ErrInvalidFrame, seq)
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
	if s.got == s.total {
		s.active = false
		return true, nil
	}
	return false, nil
}

// Fragment is a frame as read by ReadFragment, and as written by WriteFragment:
// a fragment of a message, or a whole message that was sent as a frame of
// its own, that has not been reassembled or, if it is a fragment, unescaped.
// This allows a frame to be routed, and forwarded, without
// buffering the message that it is a part of.
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

// ReadFragment reads the next frame from the underlying reader and returns it
// as it is, without reassembling or unescaping a fragmented message, so that
// it can be routed, using its Header, and forwarded, using WriteFragment on
// another Messager, one at a time without holding the message in memory.
// The frames of a message are returned in order, ending with one for which Last
// is true, with the same checks as ReadMessage makes of them, and the same
// consequences: any error other than io.EOF between messages is returned by
// all subsequent calls. ReadMessage cannot be called part way through a
// message, and fails if it is. The Fragment, and the slices it holds, are valid
// only until the next call to ReadFragment, and must not be retained or
// modified.
func (m *Messager) ReadFragment() (*Fragment, error) {
	m.rmu.Lock()
	defer m.rmu.Unlock()
	if m.rerr != nil {
		return nil, m.rerr
	}
	f, err := m.readFragment()
	if err != nil {
		if err != io.EOF { //nolint:errorlint // io.EOF is returned unwrapped, and only by the first read.
			m.rerr = err
		}
		return nil, err
	}
	return f, nil
}

func (m *Messager) readFragment() (*Fragment, error) {
	if cap(m.fbody) > maxRetainedBuffer {
		m.fbody = nil
	}
	n, err := m.readFrameLen(!m.rst.active)
	if err != nil {
		return nil, err
	}
	m.fbody, err = m.readBody(m.fbody, n)
	if err != nil {
		return nil, err
	}
	body := m.fbody
	f := &m.frag
	*f = Fragment{}
	if !bytes.HasPrefix(body, []byte(envelopePrefix)) {
		if m.rst.active {
			return nil, fmt.Errorf("%w: a message that is not a fragment arrived part way through one", ErrInvalidFrame)
		}
		f.Bare, f.Payload, f.Total, f.Len, f.Last = true, body, uint64(n), n, true
		return f, nil
	}
	if !m.fragmentation {
		return nil, fmt.Errorf("%w: a fragment was received but fragmentation is not enabled", ErrInvalidFrame)
	}
	fh, err := parseFragmentHeader(body, m.maxHeaderSize)
	if err != nil {
		return nil, err
	}
	if err := m.rst.begin(fh.seq, fh.total, fh.hdr, uint64(m.maxFragmentedMessageSize)); err != nil {
		return nil, err
	}
	payload := body[fh.payloadAt : len(body)-len(envelopeSuffix)]
	l, err := unescapedLen(payload)
	if err != nil {
		return nil, err
	}
	last, err := m.rst.add(l)
	if err != nil {
		return nil, err
	}
	f.Seq, f.Total, f.Header, f.Payload, f.Len, f.Last = fh.seq, fh.total, fh.hdr, payload, l, last
	return f, nil
}

// WriteFragment writes a frame, as returned by ReadFragment, to the underlying
// writer, to forward it. The frame is written as it is, rebuilt from its
// fields, and not refragmented: it must fit in the fragment size, or the maximum
// size if fragmentation is not enabled, of this Messager, and what it carries
// is checked with the same rules as are applied to what is read. In
// particular, the Header is sent if it is set, whether or not it was set in
// the frame that was read, so that a sender that is not repeating headers
// may add them. A message is the frames from Seq 0 to one that is Last, which
// must be written one after the other, and no other message may be written, by
// WriteMessage or otherwise, between them, and fails if it is. Unlike an error
// writing a frame, an error that is found before writing, such as one in the
// sequence, does not stop later calls, but a message part way through is
// then incomplete and so is poisoned.
func (m *Messager) WriteFragment(f *Fragment) error {
	if f == nil {
		return fmt.Errorf("%w: nil fragment", ErrInvalidFrame)
	}
	m.wmu.Lock()
	defer m.wmu.Unlock()
	if m.werr != nil {
		return m.werr
	}
	midMessage := m.wst.active
	err := m.writeFragment(f)
	if err != nil && m.werr == nil {
		if midMessage {
			// The peer has the start of a message, that can no longer be completed.
			m.werr = err
		} else {
			m.wst.active = false
		}
	}
	return err
}

func (m *Messager) writeFragment(f *Fragment) error {
	buf := m.fragBuf[:0]
	buf = append(buf, 0, 0, 0, 0)
	defer func() {
		if cap(buf) <= maxRetainedBuffer {
			m.fragBuf = buf
		} else {
			m.fragBuf = nil
		}
	}()
	limit := uint64(m.maxSize)
	if m.fragmentation {
		limit = uint64(m.fragmentSize)
	}
	if f.Bare {
		buf = append(buf, f.Payload...)
		return m.writeBare(buf, limit)
	}
	var err error
	if buf, err = m.writeEnvelope(buf, f, limit); err != nil {
		return err
	}
	return m.writeFrame(buf, uint64(len(buf)-4))
}

// writeBare writes buf, a frame with room for its length, as a message that is
// not a fragment.
func (m *Messager) writeBare(buf []byte, limit uint64) error {
	size := uint64(len(buf) - 4)
	switch {
	case m.wst.active:
		return fmt.Errorf("%w: a message that is not a fragment was written part way through one", ErrInvalidFrame)
	case size == 0:
		return fmt.Errorf("%w: empty message", ErrInvalidFrame)
	case size > limit:
		return fmt.Errorf("%w: message size %d exceeds maximum %d", ErrMessageTooLarge, size, limit)
	case bytes.HasPrefix(buf[4:], []byte(envelopePrefix)):
		return fmt.Errorf("%w: a message may not begin with %s", ErrInvalidFrame, envelopePrefix)
	}
	return m.writeFrame(buf, size)
}

// writeEnvelope checks the fragment f and, if it is acceptable, appends it
// to buf, a frame with room for its length, and accounts for it in the state
// of the message that is being written.
func (m *Messager) writeEnvelope(buf []byte, f *Fragment, limit uint64) ([]byte, error) {
	if !m.fragmentation {
		return buf, fmt.Errorf("%w: fragmentation is not enabled", ErrInvalidFrame)
	}
	if len(f.Header) != 0 {
		if err := m.checkHeader(f.Header); err != nil {
			return buf, err
		}
	}
	if len(f.Payload) == 0 {
		return buf, fmt.Errorf("%w: empty fragment payload", ErrInvalidFrame)
	}
	l, err := unescapedLen(f.Payload)
	if err != nil {
		return buf, err
	}
	buf = appendFragmentHeader(buf, f.Seq, f.Total, f.Header)
	buf = append(buf, f.Payload...)
	buf = append(buf, envelopeSuffix...)
	if size := uint64(len(buf) - 4); size > limit {
		return buf, fmt.Errorf("%w: fragment size %d exceeds maximum %d", ErrMessageTooLarge, size, limit)
	}
	if err := m.wst.begin(f.Seq, f.Total, f.Header, uint64(m.maxFragmentedMessageSize)); err != nil {
		return buf, err
	}
	_, err = m.wst.add(l)
	return buf, err
}

// unescapedLen validates the contents of a JSON string, that is the payload of
// a fragment, as unescapeChunk does, but without decoding it, and returns the
// number of bytes that it stands for.
func unescapedLen(b []byte) (int, error) {
	if !utf8.Valid(b) {
		return 0, errPayload("not valid UTF-8")
	}
	n := 0
	for r := 0; r < len(b); {
		c := b[r]
		switch {
		case c < 0x20:
			return 0, errPayload("unescaped control character")
		case c == '"':
			return 0, errPayload("unescaped quote")
		case c != '\\':
			n++
			r++
			continue
		}
		r++
		if r >= len(b) {
			return 0, errPayload("escape cut short")
		}
		if simpleEscapes[b[r]] != 0 {
			n++
			r++
			continue
		}
		if b[r] != 'u' {
			return 0, errPayload("invalid escape")
		}
		v, next, err := decodeUnicodeEscape(b, r+1)
		if err != nil {
			return 0, err
		}
		n += utf8.RuneLen(v)
		r = next
	}
	return n, nil
}
