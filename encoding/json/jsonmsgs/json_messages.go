// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package jsonmsgs provides support for efficient encoding and decoding
// arbitrary json messages over a stream, ie. an arbitrary io.Reader or io.Writer
// etc. The message format has a 5-byte header overhead: a 4-byte little-endian length
// (covering the 1-byte flags plus payload) followed by a 1-byte flags field, followed
// by the encoded json data.
package jsonmsgs

import (
	"bytes"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"io"
	"sync"
)

// DefaultMaxNativeMessageSize is the default maximum size of a single frame, in bytes.
const DefaultMaxNativeMessageSize = 1024 * 1024 // 1MB

// DefaultMaxFragmentedMessageSize is the default maximum total size of a fragmented/reassembled message, in bytes (16MB).
const DefaultMaxFragmentedMessageSize = 16 * 1024 * 1024 // 16MB

const (
	// FlagFragment indicates that the frame is a fragment of a larger message.
	// It is a bit flag in the 1-byte header flags field.
	FlagFragment byte = 1 << 0

	// FlagMore indicates that more fragments follow for the current message.
	// It is a bit flag in the 1-byte header flags field.
	FlagMore byte = 1 << 1
)

var (
	ErrMessageTooLarge = errors.New("jsonmsgs: message too large")
)

type options struct {
	// MaxSize specifies the maximum size of a single frame in bytes.
	// If MaxSize is 0, the default maximum size of 1MB is used.
	maxSize uint32

	// fragmentSize specifies the maximum size of a fragment frame in bytes.
	// It defaults to MaxSize, but can be configured to be smaller.
	fragmentSize uint32

	// maxFragmentedMessageSize specifies the maximum size of a message that can
	// be fragmented in bytes. If 0, DefaultMaxFragmentedMessageSize (16MB) is used.
	maxFragmentedMessageSize uint32

	// fragmentation enables automatic message fragmentation.
	// Defaults to false.
	fragmentation bool

	encoderOptions jsontext.Options
	decoderOptions jsontext.Options
}

// Option represents an option for configuring a Messager.
type Option func(*options)

// WithMaxSize sets the maximum size of a single frame body (1-byte flags + payload)
// in bytes. If maxSize is 0, DefaultMaxNativeMessageSize (1MB) is used.
// NewMessager panics if maxSize is less than 2.
func WithMaxSize(maxSize uint32) Option {
	return func(opts *options) {
		opts.maxSize = maxSize
	}
}

// WithFragmentSize sets the maximum size of a fragment frame body (1-byte flags +
// chunk payload) in bytes. It defaults to MaxSize, but can be configured to be
// smaller. If fragmentSize is 0 or exceeds MaxSize, it is set to MaxSize.
// NewMessager panics if fragmentSize is less than 2.
func WithFragmentSize(fragmentSize uint32) Option {
	return func(opts *options) {
		opts.fragmentSize = fragmentSize
	}
}

// WithMaxFragmentedMessageSize sets the maximum total size of a message that can
// be fragmented in bytes. If 0, DefaultMaxFragmentedMessageSize (16MB) is used.
// If set (> 0), WriteMessage returns ErrMessageTooLarge if a message to be fragmented
// exceeds this size, and ReadMessage returns an error if the total reassembled size
// exceeds this limit. This prevents deadlocks when writing over buffered channels
// that could fill up before a complete request is sent, as well as protecting
// receivers from unbounded memory growth.
func WithMaxFragmentedMessageSize(maxSize uint32) Option {
	return func(opts *options) {
		opts.maxFragmentedMessageSize = maxSize
	}
}

// WithFragmentation controls whether automatic message fragmentation is enabled.
// The default is false. When false, WriteMessage returns ErrMessageTooLarge if
// a message exceeds MaxSize, and ReadMessage returns an error if a fragment frame
// is received. When true, messages exceeding the fragment size (which defaults to
// MaxSize) are automatically fragmented into multiple frames.
func WithFragmentation(enable bool) Option {
	return func(opts *options) {
		opts.fragmentation = enable
	}
}

func WithEncoderOptions(opts jsontext.Options) Option {
	return func(o *options) {
		o.encoderOptions = opts
	}
}

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
}

type Messager struct {
	rd                       io.ReadCloser
	wr                       io.Writer
	maxSize                  uint32
	fragmentSize             uint32
	maxFragmentedMessageSize uint32
	fragmentation            bool
	encPool                  sync.Pool
	decPool                  sync.Pool
	wmu                      sync.Mutex
	rmu                      sync.Mutex
	fragScratch              []byte // scratch buffer for non-first fragment frames; guarded by wmu
}

// NewMessager creates a new Messager with the given readCloser and writer.
// If maxSize is not specified via WithMaxSize, DefaultMaxNativeMessageSize (1MB) is used.
// If fragmentSize is not specified via WithFragmentSize, it defaults to maxSize.
// NewMessager panics if maxSize or fragmentSize is less than 2.
func NewMessager(rd io.ReadCloser, wr io.Writer, opts ...Option) *Messager {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if o.maxSize == 0 {
		o.maxSize = DefaultMaxNativeMessageSize
	}
	if o.maxSize < 2 {
		panic(fmt.Sprintf("jsonmsgs: maxSize %d must be at least 2", o.maxSize))
	}
	if o.fragmentSize == 0 || o.fragmentSize > o.maxSize {
		o.fragmentSize = o.maxSize
	}
	if o.fragmentSize < 2 {
		panic(fmt.Sprintf("jsonmsgs: fragmentSize %d must be at least 2", o.fragmentSize))
	}
	if o.maxFragmentedMessageSize == 0 {
		o.maxFragmentedMessageSize = DefaultMaxFragmentedMessageSize
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
	}
	nm.encPool = sync.Pool{
		New: func() any {
			buf := bytes.NewBuffer(make([]byte, 0, 1024))
			buf.Write([]byte{0, 0, 0, 0, 0})
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
	enc.buffer.Write([]byte{0, 0, 0, 0, 0})
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
	m.encPool.Put(enc)
}

func (m *Messager) ReleaseDecoder(dec *Decoder) {
	if dec == nil || dec.buf == nil {
		return
	}
	if cap(dec.buffer) > int(m.maxSize)*4 && cap(dec.buffer) > 4*1024*1024 {
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

// WriteMessage writes a message to the underlying writer with a 5-byte header
// (a 4-byte little-endian length prefix followed by a 1-byte flags field).
// The length prefix encodes 1 + payload size (covering the flags byte and the payload).
// If fragmentation is enabled (via WithFragmentation) and the message body exceeds
// the configured fragment size (which defaults to MaxSize), it is transparently
// fragmented into multiple frames of at most fragment size bytes.
// If fragmentation is disabled (the default) and the message exceeds MaxSize,
// ErrMessageTooLarge is returned. The encoder is returned to the pool after use
// regardless of error.
func (m *Messager) WriteMessage(enc *Encoder) error {
	if enc == nil || enc.buffer == nil {
		return errors.New("nil encoder")
	}
	m.wmu.Lock()
	defer m.wmu.Unlock()
	defer m.encPool.Put(enc)
	data := enc.buffer.Bytes()
	if len(data) < 5 {
		return fmt.Errorf("buffer too small to write header")
	}
	payloadSize := len(data) - 5
	frameLen := uint64(payloadSize) + 1

	if (!m.fragmentation && frameLen <= uint64(m.maxSize)) || (m.fragmentation && frameLen <= uint64(m.fragmentSize)) {
		data[0] = byte(frameLen)
		data[1] = byte(frameLen >> 8)
		data[2] = byte(frameLen >> 16)
		data[3] = byte(frameLen >> 24)
		data[4] = 0 // unfragmented: no flags
		return writeFull(m.wr, data)
	}

	if !m.fragmentation {
		return fmt.Errorf("%w: message size %d exceeds maximum %d", ErrMessageTooLarge, frameLen, m.maxSize)
	}

	if err := m.checkFragmentedLimit(uint64(payloadSize)); err != nil {
		return err
	}

	return m.writeFragmentedMessage(data, data[5:])
}

func (m *Messager) writeFragmentedMessage(data []byte, payload []byte) error {
	maxChunk := int(m.fragmentSize) - 1
	for off := 0; off < len(payload); off += maxChunk {
		chunkLen := min(maxChunk, len(payload)-off)
		isLast := (off+chunkLen == len(payload))
		frameLen := uint32(1 + chunkLen)
		flags := FlagFragment
		if !isLast {
			flags |= FlagMore
		}
		if off == 0 {
			data[0] = byte(frameLen)
			data[1] = byte(frameLen >> 8)
			data[2] = byte(frameLen >> 16)
			data[3] = byte(frameLen >> 24)
			data[4] = flags
			if err := writeFull(m.wr, data[:5+chunkLen]); err != nil {
				return err
			}
			continue
		}
		// Combine the header and payload into a single contiguous write so
		// that unbuffered writers (e.g. a net.Conn) don't incur two Write
		// syscalls per fragment.
		need := 5 + chunkLen
		if cap(m.fragScratch) < need {
			m.fragScratch = make([]byte, need)
		} else {
			m.fragScratch = m.fragScratch[:need]
		}
		m.fragScratch[0] = byte(frameLen)
		m.fragScratch[1] = byte(frameLen >> 8)
		m.fragScratch[2] = byte(frameLen >> 16)
		m.fragScratch[3] = byte(frameLen >> 24)
		m.fragScratch[4] = flags
		copy(m.fragScratch[5:], payload[off:off+chunkLen])
		if err := writeFull(m.wr, m.fragScratch); err != nil {
			return err
		}
	}
	return nil
}

// ReadMessage reads a message from the underlying reader, returning a
// Decoder that can be used to decode the message. If the message was fragmented,
// all fragments are read and reassembled into a single Decoder.
// The Decoder must be released by calling ReleaseDecoder when no longer needed.
// ReadMessage will block until a complete message is read or an error occurs.
func (m *Messager) ReadMessage() (*Decoder, error) {
	m.rmu.Lock()
	defer m.rmu.Unlock()
	// Read the 5-byte header: 4-byte little-endian length followed by 1-byte flags.
	var hdr [5]byte
	if _, err := io.ReadFull(m.rd, hdr[:]); err != nil {
		return nil, err
	}
	frameLen := uint32(hdr[0]) |
		uint32(hdr[1])<<8 |
		uint32(hdr[2])<<16 |
		uint32(hdr[3])<<24
	flags := hdr[4]

	isFragment := (flags & FlagFragment) != 0
	hasMore := (flags & FlagMore) != 0

	// FlagMore says that continuation fragments belong to this message, which
	// only makes sense for a frame that is itself a fragment. Accepting it as
	// a complete unfragmented message would leave its continuations to be read
	// as messages of their own, so malformed input could move message
	// boundaries. Reject it before the payload is read, whether or not
	// fragmentation is enabled.
	if hasMore && !isFragment {
		return nil, fmt.Errorf("jsonmsgs: invalid frame header: FlagMore set without FlagFragment")
	}

	if flags & ^(FlagFragment|FlagMore) != 0 {
		return nil, fmt.Errorf("jsonmsgs: invalid frame header: unknown flags %#02x", flags)
	}

	if frameLen == 0 {
		return nil, fmt.Errorf("jsonmsgs: invalid frame length 0")
	}

	if frameLen > m.maxSize {
		return nil, fmt.Errorf("%w: message size %d exceeds maximum %d", ErrMessageTooLarge, frameLen, m.maxSize)
	}

	payloadLen := frameLen - 1
	dec := m.decPool.Get().(*Decoder)

	if !isFragment {
		if cap(dec.buffer) < int(payloadLen) {
			dec.buffer = make([]byte, payloadLen)
		} else {
			dec.buffer = dec.buffer[:payloadLen]
		}
		if _, err := io.ReadFull(m.rd, dec.buffer); err != nil {
			m.decPool.Put(dec)
			return nil, err
		}
		*dec.buf = *bytes.NewBuffer(dec.buffer)
		dec.Reset(dec.buf, dec.opts)
		return dec, nil
	}

	if err := m.readFragmentedMessage(dec, payloadLen, hasMore); err != nil {
		m.decPool.Put(dec)
		return nil, err
	}

	*dec.buf = *bytes.NewBuffer(dec.buffer)
	dec.Reset(dec.buf, dec.opts)
	return dec, nil
}

func (m *Messager) checkFragmentedLimit(totalLen uint64) error {
	if totalLen > uint64(m.maxFragmentedMessageSize) {
		return fmt.Errorf("%w: message size %d exceeds maximum fragmented message size %d", ErrMessageTooLarge, totalLen, m.maxFragmentedMessageSize)
	}
	return nil
}

func (m *Messager) readFragmentedMessage(dec *Decoder, firstPayloadLen uint32, hasMore bool) error {
	if !m.fragmentation {
		return fmt.Errorf("%w: received fragmented message when fragmentation is disabled", ErrMessageTooLarge)
	}

	totalLen := uint64(firstPayloadLen)
	if err := m.checkFragmentedLimit(totalLen); err != nil {
		return err
	}

	if cap(dec.buffer) < int(firstPayloadLen) {
		dec.buffer = make([]byte, firstPayloadLen)
	} else {
		dec.buffer = dec.buffer[:firstPayloadLen]
	}
	if _, err := io.ReadFull(m.rd, dec.buffer); err != nil {
		return err
	}

	for hasMore {
		fragPayloadLen, more, err := m.readFragmentHeader()
		if err != nil {
			return err
		}
		hasMore = more
		totalLen += uint64(fragPayloadLen)
		if err := m.checkFragmentedLimit(totalLen); err != nil {
			return err
		}
		if err := m.appendChunk(dec, fragPayloadLen); err != nil {
			return err
		}
	}
	return nil
}

func (m *Messager) readFragmentHeader() (fragPayloadLen uint32, hasMore bool, err error) {
	var hdr [5]byte
	if _, err := io.ReadFull(m.rd, hdr[:]); err != nil {
		return 0, false, err
	}
	frameLen := uint32(hdr[0]) |
		uint32(hdr[1])<<8 |
		uint32(hdr[2])<<16 |
		uint32(hdr[3])<<24
	flags := hdr[4]

	if (flags & FlagFragment) == 0 {
		return 0, false, fmt.Errorf("jsonmsgs: expected fragment frame, got unfragmented frame")
	}
	if flags & ^(FlagFragment|FlagMore) != 0 {
		return 0, false, fmt.Errorf("jsonmsgs: invalid fragment frame header: unknown flags %#02x", flags)
	}
	if frameLen == 0 {
		return 0, false, fmt.Errorf("jsonmsgs: invalid frame length 0")
	}
	if frameLen > m.maxSize {
		return 0, false, fmt.Errorf("%w: fragment size %d exceeds maximum %d", ErrMessageTooLarge, frameLen, m.maxSize)
	}
	fragPayloadLen = frameLen - 1
	if fragPayloadLen == 0 {
		return 0, false, fmt.Errorf("jsonmsgs: zero-length fragment")
	}
	return fragPayloadLen, (flags & FlagMore) != 0, nil
}

func (m *Messager) appendChunk(dec *Decoder, fragLen uint32) error {
	currLen := len(dec.buffer)
	newLen := currLen + int(fragLen)
	if cap(dec.buffer) < newLen {
		growCap := min(uint64(newLen)*2, uint64(m.maxFragmentedMessageSize))
		newBuf := make([]byte, newLen, growCap)
		copy(newBuf, dec.buffer)
		dec.buffer = newBuf
	} else {
		dec.buffer = dec.buffer[:newLen]
	}
	_, err := io.ReadFull(m.rd, dec.buffer[currLen:newLen])
	return err
}
