// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs

import (
	"bytes"
	"fmt"
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
	single bool   // fragmentation is not enabled: a message must be in a single envelope
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
			return fmt.Errorf("%w: message size %d exceeds maximum message size %d", ErrMessageTooLarge, total, maxTotal)
		}
		s.active, s.next, s.got, s.total = true, 0, 0, total
		if cap(s.hdr) > maxRetainedBuffer {
			s.hdr = nil
		}
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
	if s.single && s.got < s.total {
		return false, fmt.Errorf("%w: a message in more than one envelope requires fragmentation to be enabled", ErrInvalidFrame)
	}
	if s.got == s.total {
		s.active = false
		if cap(s.hdr) > maxRetainedBuffer {
			s.hdr = nil
		}
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
	// Neither it nor Header may be modified in place, see Writer.WriteFragment.
	Payload []byte

	// Len is the number of bytes of the message that Payload stands for once
	// unescaped. It is set by ReadFragment, and ignored by WriteFragment which
	// works it out.
	Len int

	// Last is true if this is the last fragment of the message, or Bare.
	Last bool

	// verified is what ReadFragment checked, see WriteFragment.
	verified verified
}

// verified records the payload and header that ReadFragment checked, and the
// number of bytes that the payload stands for, so that WriteFragment, which
// would otherwise check them again, can rely on that for a Fragment that has not
// been changed. A slice has no identity, so a payload or header is the same as
// the one that was checked if it starts at the same byte and has the same
// length: one that has been replaced, or cut short, is checked afresh.
type verified struct {
	payload, header       *byte
	payloadLen, headerLen int
	unescaped             int
	ok                    bool
}

func firstByte(b []byte) *byte {
	if len(b) == 0 {
		return nil
	}
	return &b[0]
}

func newVerified(payload, header []byte, unescaped int) verified {
	return verified{firstByte(payload), firstByte(header), len(payload), len(header), unescaped, true}
}

// holds reports whether f has the payload and header that were checked.
func (v *verified) holds(f *Fragment) bool {
	return v.ok && !f.Bare &&
		firstByte(f.Payload) == v.payload && len(f.Payload) == v.payloadLen &&
		firstByte(f.Header) == v.header && len(f.Header) == v.headerLen
}
