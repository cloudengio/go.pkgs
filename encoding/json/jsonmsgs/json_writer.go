// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonmsgs

import (
	"bytes"
	"encoding/binary"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"sync"
	"unicode/utf8"
)

// Writer writes messages to a stream, fragmenting them if need be, see the
// package documentation, and Option for the options that apply to it. A Writer
// is safe for concurrent use: a message is written whole by any one call and
// calls are serialized.
type Writer struct {
	wr                       io.Writer
	maxSize                  uint32
	fragmentSize             uint32
	maxFragmentedMessageSize uint32
	maxHeaderSize            uint32
	repeatHeader             bool
	fragmentation            bool
	encPool                  sync.Pool

	wmu     sync.Mutex
	werr    error     // the error that failed a write, returned by all later writes; guarded by wmu
	fragBuf []byte    // the frame being built by writeFragments and writeFragment; guarded by wmu
	wst     fragState // the message being forwarded by WriteFragment; guarded by wmu
}

// NewWriter creates a new Writer that writes to wr, which may be nil to discard
// what is written. It panics if the options are invalid, see the individual
// options.
func NewWriter(wr io.Writer, opts ...Option) *Writer {
	o := resolveOptions(opts)
	if least := MinFragmentSizeWithHeader(o.maxFragmentedMessageSize, o.maxHeaderSize); o.fragmentation && o.fragmentSize < least {
		panic(fmt.Sprintf("jsonmsgs: fragment size %d must be at least %d to fragment messages of up to %d bytes",
			o.fragmentSize, least, o.maxFragmentedMessageSize))
	}
	if wr == nil {
		wr = io.Discard
	}
	w := &Writer{
		wr:                       wr,
		maxSize:                  o.maxSize,
		fragmentSize:             o.fragmentSize,
		maxFragmentedMessageSize: o.maxFragmentedMessageSize,
		maxHeaderSize:            o.maxHeaderSize,
		repeatHeader:             o.repeatHeader,
		fragmentation:            o.fragmentation,
	}
	w.wst.single = !o.fragmentation
	w.encPool = sync.Pool{
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
	return w
}

// Encoder captures the state to encode and send a single message.
// It must be obtained using Writer.NewEncoder. It will be reclaimed by
// Writer.WriteMessage after which it cannot be used again.
type Encoder struct {
	*jsontext.Encoder
	opts   jsontext.Options
	buffer *bytes.Buffer
}

// NewEncoder creates a new Encoder for encoding a single message.
func (w *Writer) NewEncoder() *Encoder {
	enc := w.encPool.Get().(*Encoder)
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
func (w *Writer) ReleaseEncoder(enc *Encoder) {
	if enc == nil || enc.buffer == nil {
		return
	}
	w.putEncoder(enc)
}

// putEncoder returns enc to the pool, first dropping a buffer that has grown
// large so that a single large message does not pin its memory in the pool.
func (w *Writer) putEncoder(enc *Encoder) {
	if enc.buffer.Cap() > maxRetainedBuffer && uint64(enc.buffer.Cap()) > uint64(w.maxSize)*4 {
		enc.buffer = bytes.NewBuffer(make([]byte, 0, 1024))
	}
	w.encPool.Put(enc)
}

// WriteMessage writes the message encoded by enc, as a bare frame or in
// envelopes as described in the package documentation, each frame being written
// with a single call to Write. The newline that the encoder appends to a message
// is not written. enc is returned to the pool whatever the outcome and must not
// be used again. See "Errors" in the package documentation for the errors that
// can be retried.
func (w *Writer) WriteMessage(enc *Encoder) error {
	return w.writeMessage(enc, nil)
}

// WriteMessageWithHeader is like WriteMessage but sends header with the message,
// see "User header" in the package documentation. The header must be a valid
// JSON value of at most the size set by WithMaxHeaderSize, which is checked.
func (w *Writer) WriteMessageWithHeader(enc *Encoder, header jsontext.Value) error {
	if len(header) == 0 {
		// Return enc to the pool, as writeMessage does for every other outcome,
		// since the caller has handed it over and will not use or release it.
		w.ReleaseEncoder(enc)
		return fmt.Errorf("%w: empty header", ErrInvalidFrame)
	}
	return w.writeMessage(enc, header)
}

// checkHeader checks a user header that is to be sent.
func (w *Writer) checkHeader(h []byte) error {
	switch {
	case w.maxHeaderSize == 0:
		return fmt.Errorf("%w: headers are not enabled, see WithMaxHeaderSize", ErrInvalidFrame)
	case uint64(len(h)) > uint64(w.maxHeaderSize):
		return fmt.Errorf("%w: header size %d exceeds maximum %d", ErrMessageTooLarge, len(h), w.maxHeaderSize)
	case !jsontext.Value(h).IsValid():
		return fmt.Errorf("%w: header is not valid JSON", ErrInvalidFrame)
	}
	return nil
}

func (w *Writer) writeMessage(enc *Encoder, header []byte) error {
	if enc == nil || enc.buffer == nil {
		return errors.New("nil encoder")
	}
	w.wmu.Lock()
	defer w.wmu.Unlock()
	defer w.putEncoder(enc)
	if w.werr != nil {
		return w.werr
	}
	if w.wst.active {
		return fmt.Errorf("%w: a message is being forwarded with WriteFragment", ErrInvalidFrame)
	}
	if header != nil {
		if err := w.checkHeader(header); err != nil {
			return err
		}
	}
	data, err := w.encodedMessage(enc)
	if err != nil {
		return err
	}
	return w.sendMessage(data, header)
}

// encodedMessage returns the frame, with room for its length, that holds the
// message encoded by enc after checking that it is not too large or empty.
func (w *Writer) encodedMessage(enc *Encoder) ([]byte, error) {
	data := enc.buffer.Bytes()
	if len(data) < 4 {
		return nil, fmt.Errorf("%w: buffer too small to write length prefix", ErrInvalidFrame)
	}
	if err := w.checkRawSize(uint64(len(data) - 4)); err != nil {
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
// single bare frame, or in one or more envelopes, as described by WriteMessage.
func (w *Writer) sendMessage(data, header []byte) error {
	size := uint64(len(data) - 4)
	// A message that has a header, or that would be mistaken for an envelope,
	// is sent in an envelope, which the reader unwraps, whether or not
	// fragmentation is enabled.
	if header == nil && !bytes.HasPrefix(data[4:], []byte(envelopePrefix)) {
		if size <= w.frameLimit() {
			return w.writeFrame(data, size)
		}
		if !w.fragmentation {
			return fmt.Errorf("%w: message size %d exceeds maximum %d", ErrMessageTooLarge, size, w.maxSize)
		}
	}
	if size > w.maxMessage() {
		return fmt.Errorf("%w: message size %d exceeds maximum message size %d", ErrMessageTooLarge, size, w.maxMessage())
	}
	return w.writeFragments(data[4:], header)
}

// frameLimit is the largest frame that is written: the fragment size if
// fragmentation is enabled and otherwise the maximum size.
func (w *Writer) frameLimit() uint64 {
	if w.fragmentation {
		return uint64(w.fragmentSize)
	}
	return uint64(w.maxSize)
}

// maxMessage is the largest message that is written or accepted: the maximum
// fragmented message size if fragmentation is enabled, and otherwise that of
// a frame.
func (w *Writer) maxMessage() uint64 {
	if w.fragmentation {
		return uint64(w.maxFragmentedMessageSize)
	}
	return uint64(w.maxSize)
}

// checkRawSize rejects an encoded message of n bytes, which may include the
// newline that is trimmed from it, that is too large to be written, before any
// of it is looked at.
func (w *Writer) checkRawSize(n uint64) error {
	limit := w.maxMessage()
	if n > limit+1 {
		return fmt.Errorf("%w: message size %d exceeds maximum %d", ErrMessageTooLarge, n, limit)
	}
	return nil
}

// writeFrame writes data, which is a frame with room for its length at its
// start, and size bytes of body, as a single frame.
func (w *Writer) writeFrame(data []byte, size uint64) error {
	binary.LittleEndian.PutUint32(data, uint32(size))
	if err := writeFull(w.wr, data); err != nil {
		// Part of the frame may have been written.
		w.werr = permanent(err)
		return w.werr
	}
	return nil
}

// writeFragments writes text, a message of any length, in envelopes: see the
// package documentation for the format and for the argument that every
// envelope fits in the frame limit, which is the fragment size if
// fragmentation is enabled and otherwise the maximum size, in which case the
// message must fit in one. The text must be valid UTF-8 for it to be sent as
// a series of JSON strings, and is checked in its entirety before anything is
// written so that a message is never abandoned part way through.
func (w *Writer) writeFragments(text, header []byte) error {
	if !utf8.Valid(text) {
		return fmt.Errorf("%w: a message must be valid UTF-8 to be sent in an envelope", ErrInvalidFrame)
	}
	total := uint64(len(text))
	budget := int(w.frameLimit())
	buf := slices.Grow(w.fragBuf[:0], 4+budget)
	defer func() {
		// Keep the buffer for the next message unless it is large.
		if cap(buf) <= maxRetainedBuffer {
			w.fragBuf = buf
		} else {
			w.fragBuf = nil
		}
	}()

	for off, seq := 0, uint64(0); off < len(text); seq++ {
		// Room for the length, then the header of this fragment, exactly.
		buf = append(buf[:0], 0, 0, 0, 0)
		var h []byte
		if seq == 0 || w.repeatHeader {
			h = header
		}
		buf = appendFragmentHeader(buf, seq, total, h)

		// What remains of the budget, once the header and the closing "}, which
		// are known, are accounted for, is exactly what the payload may use.
		room := budget - (len(buf) - 4) - len(envelopeSuffix)
		var n int
		buf, n = appendEscaped(buf, text[off:], room)
		if !w.fragmentation && n < len(text) {
			// Without fragmentation a message must fit in a single envelope,
			// which is known before anything is written.
			return fmt.Errorf("%w: message of %d bytes does not fit in an envelope of at most %d bytes", ErrMessageTooLarge, len(text), budget)
		}
		if n == 0 {
			// Not possible given MinFragmentSize, and so a bug if it happens;
			// fail rather than loop. If part of the message has been written the
			// peer is left with an incomplete message.
			err := fmt.Errorf("%w: no progress fragmenting message at offset %d with fragment size %d", ErrInvalidFrame, off, budget)
			if off > 0 {
				w.werr = permanent(err)
				return w.werr
			}
			return err
		}
		off += n
		buf = append(buf, envelopeSuffix...)
		if err := w.writeFrame(buf, uint64(len(buf)-4)); err != nil {
			return err
		}
	}
	return nil
}

// WriteFragment writes a frame, as returned by ReadFragment, as it is, see
// "Forwarding" in the package documentation. It is not refragmented and so must
// fit in the frame size of this Writer. The Header is sent if it is set,
// whether or not it was set in the frame that was read. The frames of a message
// must be written in order, from Seq 0 to the one that is Last, with no other
// message written between them. See "Errors" in the package documentation for
// the errors that can be retried; a nil Fragment is always rejected, with
// ErrInvalidFrame, without any effect.
func (w *Writer) WriteFragment(f *Fragment) error {
	if f == nil {
		return fmt.Errorf("%w: nil fragment", ErrInvalidFrame)
	}
	w.wmu.Lock()
	defer w.wmu.Unlock()
	if w.werr != nil {
		return w.werr
	}
	midMessage := w.wst.active
	err := w.writeFragment(f)
	if err != nil && w.werr == nil {
		if midMessage {
			// The peer has the start of a message, that can no longer be completed.
			err = permanent(err)
			w.werr = err
		} else {
			w.wst.active = false
		}
	}
	return err
}

func (w *Writer) writeFragment(f *Fragment) error {
	buf := w.fragBuf[:0]
	buf = append(buf, 0, 0, 0, 0)
	defer func() {
		if cap(buf) <= maxRetainedBuffer {
			w.fragBuf = buf
		} else {
			w.fragBuf = nil
		}
	}()
	limit := w.frameLimit()
	if f.Bare {
		buf = append(buf, f.Payload...)
		return w.writeBare(buf, limit)
	}
	var err error
	if buf, err = w.writeEnvelope(buf, f, limit); err != nil {
		return err
	}
	return w.writeFrame(buf, uint64(len(buf)-4))
}

// writeBare writes buf, a frame with room for its length, as a message that is
// not a fragment.
func (w *Writer) writeBare(buf []byte, limit uint64) error {
	size := uint64(len(buf) - 4)
	switch {
	case w.wst.active:
		return fmt.Errorf("%w: a message that is not a fragment was written part way through one", ErrInvalidFrame)
	case size == 0:
		return fmt.Errorf("%w: empty message", ErrInvalidFrame)
	case size > limit:
		return fmt.Errorf("%w: message size %d exceeds maximum %d", ErrMessageTooLarge, size, limit)
	case bytes.HasPrefix(buf[4:], []byte(envelopePrefix)):
		return fmt.Errorf("%w: a message may not begin with %s", ErrInvalidFrame, envelopePrefix)
	}
	return w.writeFrame(buf, size)
}

// writeEnvelope checks the fragment f and, if it is acceptable, appends it
// to buf, a frame with room for its length, and accounts for it in the state
// of the message that is being written.
func (w *Writer) writeEnvelope(buf []byte, f *Fragment, limit uint64) ([]byte, error) {
	if len(f.Header) != 0 {
		if err := w.checkHeader(f.Header); err != nil {
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
	if err := w.wst.begin(f.Seq, f.Total, f.Header, w.maxMessage()); err != nil {
		return buf, err
	}
	_, err = w.wst.add(l)
	return buf, err
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
