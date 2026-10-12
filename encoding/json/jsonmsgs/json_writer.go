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
	"math"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

// Writer writes messages to a stream, fragmenting them if need be, see the
// package documentation, and Option for the options that apply to it. A Writer
// is safe for concurrent use: a message is written whole by any one call and
// calls are serialized. Close stops it writing, and closes the io.Writer if it
// is an io.Closer.
type Writer struct {
	wr                       io.Writer
	maxSize                  uint32
	fragmentSize             uint32
	maxFragmentedMessageSize uint32
	maxHeaderSize            uint32
	retain                   int // the capacity above which a buffer is not kept
	fragmentation            bool
	encPool                  sync.Pool
	closed                   atomic.Bool // Close has been called
	closeOnce                sync.Once
	closeErr                 error          // the error from closing wr, set by closeOnce
	deadliner                writeDeadliner // the connection that writes are timed out on, or nil
	writeTimeout             time.Duration

	wmu     sync.Mutex
	werr    error  // the error that failed a write, returned by all later writes; guarded by wmu
	fragBuf []byte // the frame being built by writeFragments and writeFragment; guarded by wmu
	scratch []byte // the message being refragmented by WriteFragments; guarded by wmu
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
		retain:                   o.retain,
		fragmentation:            o.fragmentation,
	}
	if d, ok := wr.(writeDeadliner); ok && o.writeTimeout > 0 {
		w.deadliner, w.writeTimeout = d, o.writeTimeout
	}
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
	if enc.buffer.Cap() > w.retain {
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
// If valid is true, h is already known to be valid JSON.
func (w *Writer) checkHeader(h []byte, valid bool) error {
	switch {
	case w.maxHeaderSize == 0:
		return fmt.Errorf("%w: headers are not enabled, see WithMaxHeaderSize", ErrInvalidFrame)
	case uint64(len(h)) > uint64(w.maxHeaderSize):
		return fmt.Errorf("%w: header size %d exceeds maximum %d", ErrMessageTooLarge, len(h), w.maxHeaderSize)
	case !valid && !jsontext.Value(h).IsValid():
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
	if err := w.usable(); err != nil {
		return err
	}
	if header != nil {
		if err := w.checkHeader(header, false); err != nil {
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

// writeDeadliner is implemented by a net.Conn, and anything else with a write
// deadline.
type writeDeadliner interface {
	SetWriteDeadline(t time.Time) error
}

// errClosed is returned by writes after Close, and is permanent.
var errClosed = permanent(ErrClosed)

// usable returns the error that all writes return, if there is one: that which
// failed an earlier write, or that the Writer was closed.
func (w *Writer) usable() error {
	if w.werr != nil {
		return w.werr
	}
	if w.closed.Load() {
		return errClosed
	}
	return nil
}

// Close closes the underlying writer if it is an io.Closer, such as a net.Conn,
// which causes a write that is blocked on it to return, with an error, and
// stops the Writer from writing: all later writes return an error that wraps
// ErrClosed and matches ErrPermanent, as does a message that is part way through
// being written, which is then not completed. Close does not wait for a write
// that is in progress. It returns the error from closing the underlying writer,
// if any, and closes it only once, however many times it is called, from any
// number of goroutines. An underlying writer that is not an io.Closer is left
// as it is.
func (w *Writer) Close() error {
	w.closeOnce.Do(func() {
		w.closed.Store(true)
		if c, ok := w.wr.(io.Closer); ok {
			w.closeErr = c.Close()
		}
	})
	return w.closeErr
}

// write writes data, within the write timeout if there is one.
func (w *Writer) write(data []byte) error {
	if w.closed.Load() {
		// A message that is part way through is not completed.
		return ErrClosed
	}
	if w.deadliner == nil {
		return writeFull(w.wr, data)
	}
	if err := w.deadliner.SetWriteDeadline(time.Now().Add(w.writeTimeout)); err != nil {
		return fmt.Errorf("failed to set the write deadline: %w", err)
	}
	// The deadline is for this frame only, and is not left to affect anything
	// else that writes to the connection.
	defer func() { _ = w.deadliner.SetWriteDeadline(time.Time{}) }()
	return writeFull(w.wr, data)
}

// writeFrame writes data, which is a frame with room for its length at its
// start, and size bytes of body, as a single frame.
func (w *Writer) writeFrame(data []byte, size uint64) error {
	binary.LittleEndian.PutUint32(data, uint32(size))
	if err := w.write(data); err != nil {
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
	limit := w.frameLimit()
	if limit > uint64(math.MaxInt)-4 {
		return fmt.Errorf("%w: frame limit %d exceeds maximum allocatable size", ErrMessageTooLarge, limit)
	}
	budget := int(limit)
	buf := slices.Grow(w.fragBuf[:0], 4+budget)
	defer func() {
		// Keep the buffer for the next message unless it is large.
		if cap(buf) <= w.retain {
			w.fragBuf = buf
		} else {
			w.fragBuf = nil
		}
	}()

	for off, seq := 0, uint64(0); off < len(text); seq++ {
		// Room for the length, then the header of this fragment, exactly.
		buf = append(buf[:0], 0, 0, 0, 0)
		var h []byte
		if seq == 0 {
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

// WriteFragments writes the message in f, as read by Reader.ReadFragments, to
// the underlying writer, see "Forwarding" in the package documentation. It is
// not checked again, other than that its user header is acceptable to this
// Writer, since it was checked when it was read. It is written in frames of at
// most the size of this Writer's, as WriteMessage would: the frames that it was
// read from are used as they are, other than that they are put together, to send
// fewer frames, if they are smaller than this Writer's, and that any that are
// larger are split. A message that was not Bare and is written with
// fragmentation not enabled must fit in a single frame. The user header is sent
// with the first frame if there is one. f is not modified, and may be written again. See "Errors" in
// the package documentation for the errors that can be retried.
func (w *Writer) WriteFragments(f *Fragments) error {
	if f == nil || !f.valid() {
		return fmt.Errorf("%w: no fragments", ErrInvalidFrame)
	}
	w.wmu.Lock()
	defer w.wmu.Unlock()
	if err := w.usable(); err != nil {
		return err
	}
	if uint64(f.total) > w.maxMessage() {
		return fmt.Errorf("%w: message size %d exceeds maximum %d", ErrMessageTooLarge, f.total, w.maxMessage())
	}
	if len(f.header) != 0 {
		// It was checked when it was read, or set.
		if err := w.checkHeader(f.header, true); err != nil {
			return err
		}
	}
	if f.bare {
		return w.writeBareFragments(f)
	}
	return w.writeEnvelopes(f)
}

// writeBareFragments writes a message that was read as a single frame that was
// not an envelope, as a frame of its own if it has no user header and fits, and
// otherwise as WriteMessage does.
func (w *Writer) writeBareFragments(f *Fragments) error {
	msg := f.payload
	if len(f.header) == 0 && uint64(len(msg)) <= w.frameLimit() {
		buf := w.fragBuf[:0]
		buf = append(buf, 0, 0, 0, 0)
		buf = append(buf, msg...)
		w.keepFragBuf(buf)
		return w.writeFrame(buf, uint64(len(msg)))
	}
	if !w.fragmentation && len(f.header) == 0 {
		return fmt.Errorf("%w: message size %d exceeds maximum %d", ErrMessageTooLarge, len(msg), w.maxSize)
	}
	// A message with a user header is in an envelope, whatever its size.
	return w.writeFragments(msg, f.header)
}

// keepFragBuf keeps buf, which was built from fragBuf, for the next frame unless
// it is large.
func (w *Writer) keepFragBuf(buf []byte) {
	if cap(buf) <= w.retain {
		w.fragBuf = buf
	} else {
		w.fragBuf = nil
	}
}

// writeEnvelopes writes a message read as envelopes, using their payloads as
// they are where they fit.
func (w *Writer) writeEnvelopes(f *Fragments) error {
	p := &repacker{w: w, f: f, limit: int(w.frameLimit()), buf: w.fragBuf[:0]}
	defer func() {
		w.keepFragBuf(p.buf)
		if cap(w.scratch) > w.retain {
			w.scratch = nil
		}
	}()
	p.begin()
	if !w.fragmentation && p.room < len(f.payload) {
		// Without fragmentation a message must be in a single envelope, which is
		// known before anything is written.
		return fmt.Errorf("%w: message of %d bytes does not fit in an envelope of at most %d bytes", ErrMessageTooLarge, f.total, p.limit)
	}
	start := 0
	for _, end := range f.ends {
		if err := p.add(f.payload[start:end]); err != nil {
			return p.fail(err)
		}
		start = end
	}
	if p.used() > 0 {
		return p.fail(p.flush())
	}
	return nil
}

// repacker builds the frames of a message that has been read from frames, to be
// written in frames that may be larger or smaller.
type repacker struct {
	w       *Writer
	f       *Fragments
	limit   int    // the size of the largest frame
	buf     []byte // the frame being built, with room at its start for its length
	seq     uint64 // the sequence number of the frame being built
	hdrEnd  int    // the end of what precedes the payload in buf
	room    int    // the room in the frame being built for the payload
	written bool   // a frame has been written
}

// begin starts the frame that is next, as seq.
func (p *repacker) begin() {
	p.buf = append(p.buf[:0], 0, 0, 0, 0)
	var header []byte
	if p.seq == 0 {
		header = p.f.header
	}
	p.buf = appendFragmentHeader(p.buf, p.seq, uint64(p.f.total), header)
	p.hdrEnd = len(p.buf)
	p.room = p.limit - (len(p.buf) - 4) - len(envelopeSuffix)
}

// used is the length of the payload in the frame being built.
func (p *repacker) used() int {
	return len(p.buf) - p.hdrEnd
}

// flush writes the frame being built and starts the next.
func (p *repacker) flush() error {
	p.buf = append(p.buf, envelopeSuffix...)
	if err := p.w.writeFrame(p.buf, uint64(len(p.buf)-4)); err != nil {
		return err
	}
	p.written = true
	p.seq++
	p.begin()
	return nil
}

// fail returns err, and if part of the message has been written, which can not
// then be completed, makes it permanent.
func (p *repacker) fail(err error) error {
	if err != nil && p.written && p.w.werr == nil {
		err = permanent(err)
		p.w.werr = err
	}
	return err
}

// add adds the escaped payload of a frame, that was valid when it was read, to
// the frame being built, or, if it does not fit, to the next, or, if it is
// larger than a frame, splits it.
func (p *repacker) add(chunk []byte) error {
	if len(chunk) <= p.room-p.used() {
		p.buf = append(p.buf, chunk...)
		return nil
	}
	if p.used() > 0 {
		if err := p.flush(); err != nil {
			return err
		}
		if len(chunk) <= p.room {
			p.buf = append(p.buf, chunk...)
			return nil
		}
	}
	return p.split(chunk)
}

// split fills frames with as much as will fit of the payload, which is first
// unescaped and then escaped again to fit exactly, leaving the last partly
// filled for what follows.
func (p *repacker) split(chunk []byte) error {
	msg, err := Unescape(p.w.scratch[:0], chunk)
	p.w.scratch = msg
	if err != nil {
		return err
	}
	for off := 0; off < len(msg); {
		var n int
		p.buf, n = appendEscaped(p.buf, msg[off:], p.room-p.used())
		if n == 0 {
			// Not possible given MinFragmentSizeWithHeader, since each frame is
			// empty when it is begun.
			return fmt.Errorf("%w: no progress splitting a frame at offset %d with frame size %d", ErrInvalidFrame, off, p.limit)
		}
		off += n
		if off < len(msg) {
			if err := p.flush(); err != nil {
				return err
			}
		}
	}
	return nil
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
