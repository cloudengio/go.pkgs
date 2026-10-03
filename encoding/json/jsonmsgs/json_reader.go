// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs

import (
	"bytes"
	"encoding/binary"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"math"
	"slices"
	"sync"
)

// Reader reads the messages written by a Writer, or by a browser, from a
// stream, see the package documentation, and Option for the options that apply
// to it. A Reader is safe for concurrent use: a message is read whole by any one
// call and calls are serialized.
type Reader struct {
	rd                       io.ReadCloser
	maxSize                  uint32
	maxFragmentedMessageSize uint32
	maxHeaderSize            uint32
	fragmentation            bool
	decPool                  sync.Pool

	rmu   sync.Mutex
	rerr  error     // the error that failed a read, returned by all later reads; guarded by rmu
	hdr   [4]byte   // scratch for frame lengths, here so that it does not escape to the heap; guarded by rmu
	rst   fragState // the message being read; guarded by rmu
	fbody []byte    // the frame returned by ReadFragment; guarded by rmu
	frag  Fragment  // returned by ReadFragment; guarded by rmu
}

// NewReader creates a new Reader that reads from rd, which may be nil for a
// stream that is empty. It panics if the options are invalid, see the
// individual options.
func NewReader(rd io.ReadCloser, opts ...Option) *Reader {
	o := resolveOptions(opts)
	if rd == nil {
		rd = io.NopCloser(bytes.NewReader(nil))
	}
	r := &Reader{
		rd:                       rd,
		maxSize:                  o.maxSize,
		maxFragmentedMessageSize: o.maxFragmentedMessageSize,
		maxHeaderSize:            o.maxHeaderSize,
		fragmentation:            o.fragmentation,
	}
	r.rst.single = !o.fragmentation
	r.decPool = sync.Pool{
		New: func() any {
			buf := bytes.NewBuffer(make([]byte, 0, 256))
			return &Decoder{
				Decoder: jsontext.NewDecoder(buf, o.decoderOptions),
				buf:     buf,
				opts:    o.decoderOptions,
			}
		},
	}
	return r
}

// maxMessage is the largest message that is accepted: the maximum fragmented
// message size if fragmentation is enabled, and otherwise that of a frame.
func (r *Reader) maxMessage() uint64 {
	if r.fragmentation {
		return uint64(r.maxFragmentedMessageSize)
	}
	return uint64(r.maxSize)
}

// Decoder captures the state to decode a single message.
// It is created and returned by Reader.ReadMessage and must be released by
// calling Reader.ReleaseDecoder after which it cannot be used again.
type Decoder struct {
	*jsontext.Decoder
	opts   jsontext.Options
	buf    *bytes.Buffer
	buffer []byte
	header []byte
}

// Header returns the user header that was sent with the message, see
// Writer.WriteMessageWithHeader, or nil if it was sent without one. It is
// valid until the Decoder is released.
func (d *Decoder) Header() jsontext.Value {
	if len(d.header) == 0 {
		return nil
	}
	return d.header
}

// ReleaseDecoder returns a Decoder obtained from ReadMessage, which cannot be
// used again. A nil Decoder, or one that has no buffer, is ignored.
func (r *Reader) ReleaseDecoder(dec *Decoder) {
	if dec == nil || dec.buf == nil {
		return
	}
	if cap(dec.buffer) > maxRetainedBuffer && uint64(cap(dec.buffer)) > uint64(r.maxSize)*4 {
		dec.buffer = nil
		// dec.buf aliases the old dec.buffer's backing array (bytes.NewBuffer
		// does not copy), so it must be replaced too, or the large array
		// stays reachable via dec.buf until this decoder is next reused.
		dec.buf = bytes.NewBuffer(nil)
	}
	if cap(dec.header) > maxRetainedBuffer {
		dec.header = nil
	}
	r.decPool.Put(dec)
}

// Close closes the underlying reader of the Reader causing a pending
// ReadMessage to return.
func (r *Reader) Close() error {
	if r.rd == nil {
		return nil
	}
	return r.rd.Close()
}

// ReadMessage reads the next message, reassembling it if it was fragmented, and
// returns a Decoder for it, which must be released using ReleaseDecoder when no
// longer needed. The Decoder's Header returns the user header, if any. It blocks
// until a complete message is read or an error occurs, and returns io.EOF if
// the stream ends between messages. See "Errors" in the package documentation
// for the errors that can be retried.
func (r *Reader) ReadMessage() (*Decoder, error) {
	r.rmu.Lock()
	defer r.rmu.Unlock()
	if r.rerr != nil {
		return nil, r.rerr
	}
	if r.rst.active {
		return nil, fmt.Errorf("%w: a message is being forwarded with ReadFragment", ErrInvalidFrame)
	}
	dec, err := r.readMessage()
	if err != nil {
		if err != io.EOF { //nolint:errorlint // io.EOF is returned unwrapped, and only by the first read.
			err = permanent(err)
			r.rerr = err
		}
		return nil, err
	}
	return dec, nil
}

// readFull reads len(buf) bytes of a frame, converting a clean EOF, which is
// an error in the middle of a frame, to the error that says so.
func (r *Reader) readFull(buf []byte) error {
	_, err := io.ReadFull(r.rd, buf)
	if err == io.EOF { //nolint:errorlint // ReadFull returns io.EOF itself.
		return io.ErrUnexpectedEOF
	}
	return err
}

// readFrameLen reads the length of the next frame and checks it against the
// limits before anything is read or allocated on the strength of it. At the
// start of a message, atStart, the stream may end cleanly, which is io.EOF.
func (r *Reader) readFrameLen(atStart bool) (int, error) {
	if _, err := io.ReadFull(r.rd, r.hdr[:]); err != nil {
		if err == io.EOF && !atStart { //nolint:errorlint // ReadFull returns io.EOF itself.
			err = io.ErrUnexpectedEOF
		}
		return 0, err
	}
	n := binary.LittleEndian.Uint32(r.hdr[:])
	if n == 0 {
		return 0, fmt.Errorf("%w: empty frame", ErrInvalidFrame)
	}
	if n > r.maxSize {
		return 0, fmt.Errorf("%w: frame size %d exceeds maximum %d", ErrMessageTooLarge, n, r.maxSize)
	}
	return int(n), nil
}

// readBody reads n bytes, the body of a frame, into buf, reusing it if it is
// large enough. Otherwise the buffer is grown as the data arrives, doubling,
// rather than allocated at the size the frame declares, so that a peer cannot
// have this process allocate memory in advance of sending the data to fill it.
func (r *Reader) readBody(buf []byte, n int) ([]byte, error) {
	if cap(buf) >= n {
		buf = buf[:n]
		return buf, r.readFull(buf)
	}
	const firstChunk = 64 * 1024
	buf = buf[:0]
	for len(buf) < n {
		want := min(n-len(buf), max(firstChunk, len(buf)))
		buf = slices.Grow(buf, want)
		end := len(buf) + want
		err := r.readFull(buf[len(buf):end])
		if err != nil {
			return buf, err
		}
		buf = buf[:end]
	}
	return buf, nil
}

func (r *Reader) readMessage() (*Decoder, error) {
	n, err := r.readFrameLen(true)
	if err != nil {
		return nil, err
	}
	dec := r.decPool.Get().(*Decoder)
	dec.header = dec.header[:0]
	dec.buffer, err = r.readBody(dec.buffer, n)
	if err != nil {
		r.ReleaseDecoder(dec)
		return nil, err
	}
	if bytes.HasPrefix(dec.buffer, []byte(envelopePrefix)) {
		if err := r.reassemble(dec); err != nil {
			r.ReleaseDecoder(dec)
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
func (r *Reader) reassemble(dec *Decoder) error {
	st := &r.rst
	msg := dec.buffer[:0]
	defer func() { dec.buffer = msg }()
	body := dec.buffer
	for last := false; !last; {
		fh, err := parseFragmentHeader(body, r.maxHeaderSize)
		if err != nil {
			return err
		}
		if err := st.begin(fh.seq, fh.total, fh.hdr, r.maxMessage()); err != nil {
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
		n, err := r.readFrameLen(false)
		if err != nil {
			return err
		}
		// The next frame is read into the space after the message so far, and
		// is at least as large as what it decodes to, but no more than that
		// can ever be needed.
		if msg, body, err = r.readAppend(msg, n, st.total+uint64(r.maxSize)); err != nil {
			return err
		}
	}
	return nil
}

// readAppend reads the n byte body of a frame into the free space after msg,
// which it returns, with the same length but, possibly, a larger capacity, and
// the body. Like readBody, it grows msg as the data arrives, and not to hold
// all n bytes in advance, so that a peer cannot have this process allocate
// memory by declaring the size of a frame that it does not send. maxCap is the
// largest capacity that can ever be needed.
func (r *Reader) readAppend(msg []byte, n int, maxCap uint64) (grown, body []byte, err error) {
	if uint64(len(msg))+uint64(n) > uint64(math.MaxInt) {
		return msg, nil, ErrMessageTooLarge
	}
	const firstChunk = 64 * 1024
	start := len(msg)
	for got := 0; got < n; {
		want := min(n-got, max(firstChunk, got))
		end := start + got + want
		if cap(msg) < end {
			newCap := min(max(uint64(end), 2*uint64(cap(msg))), maxCap, uint64(math.MaxInt))
			bigger := make([]byte, start+got, int(newCap))
			copy(bigger, msg[:start+got])
			msg = bigger[:start]
		}
		if err := r.readFull(msg[:end][start+got:]); err != nil {
			return msg, nil, err
		}
		got += want
	}
	return msg, msg[start : start+n], nil
}

// ReadFragment reads the next frame and returns it as it is, without
// reassembling or unescaping a fragmented message, so that it can be routed
// using its Header and forwarded using Writer.WriteFragment, see "Forwarding" in
// the package documentation. It returns io.EOF if the stream ends between
// messages, and otherwise the errors that ReadMessage does. The Fragment, and
// the slices it holds, are valid only until the next call to ReadFragment, and
// must not be retained or modified.
func (r *Reader) ReadFragment() (*Fragment, error) {
	r.rmu.Lock()
	defer r.rmu.Unlock()
	if r.rerr != nil {
		return nil, r.rerr
	}
	f, err := r.readFragment()
	if err != nil {
		if err != io.EOF { //nolint:errorlint // io.EOF is returned unwrapped, and only by the first read.
			err = permanent(err)
			r.rerr = err
		}
		return nil, err
	}
	return f, nil
}

func (r *Reader) readFragment() (*Fragment, error) {
	if cap(r.fbody) > maxRetainedBuffer {
		r.fbody = nil
	}
	n, err := r.readFrameLen(!r.rst.active)
	if err != nil {
		return nil, err
	}
	r.fbody, err = r.readBody(r.fbody, n)
	if err != nil {
		return nil, err
	}
	body := r.fbody
	f := &r.frag
	*f = Fragment{}
	if !bytes.HasPrefix(body, []byte(envelopePrefix)) {
		if r.rst.active {
			return nil, fmt.Errorf("%w: a message that is not a fragment arrived part way through one", ErrInvalidFrame)
		}
		f.Bare, f.Payload, f.Total, f.Len, f.Last = true, body, uint64(n), n, true
		return f, nil
	}
	fh, err := parseFragmentHeader(body, r.maxHeaderSize)
	if err != nil {
		return nil, err
	}
	if err := r.rst.begin(fh.seq, fh.total, fh.hdr, r.maxMessage()); err != nil {
		return nil, err
	}
	payload := body[fh.payloadAt : len(body)-len(envelopeSuffix)]
	l, err := unescapedLen(payload)
	if err != nil {
		return nil, err
	}
	last, err := r.rst.add(l)
	if err != nil {
		return nil, err
	}
	f.Seq, f.Total, f.Header, f.Payload, f.Len, f.Last = fh.seq, fh.total, fh.hdr, payload, l, last
	f.verified = newVerified(payload, fh.hdr, l)
	return f, nil
}
