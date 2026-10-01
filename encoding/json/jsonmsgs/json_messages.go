// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package jsonmsgs provides support for efficient encoding and decoding
// arbitrary json messages over a stream, ie. an arbitrary io.Reader or io.Writer
// etc. The message format is simply a 4 byte little endian length followed by
// the encoded json data.
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

// DefaultMaxMessageSize is the default maximum total reassembled size of a message, in bytes (100MB).
const DefaultMaxMessageSize = 100 * 1024 * 1024 // 100MB

const (
	// FlagFragment indicates that the frame is a fragment of a larger message.
	// It occupies bit 31 of the 4-byte frame header.
	FlagFragment uint32 = 1 << 31

	// FlagMore indicates that more fragments follow for the current message.
	// It occupies bit 30 of the 4-byte frame header.
	FlagMore uint32 = 1 << 30

	// LengthMask masks the 30-bit frame payload length (bits 0..29).
	LengthMask uint32 = 0x3fffffff
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

	// MaxMessageSize specifies the maximum total size of a reassembled message in bytes.
	// If MaxMessageSize is 0, DefaultMaxMessageSize (100MB) is used.
	maxMessageSize uint32

	// maxFragmentedMessageSize specifies the maximum size of a message that can
	// be fragmented in bytes. If 0, no limit is enforced beyond maxMessageSize.
	maxFragmentedMessageSize uint32

	// fragmentation enables automatic message fragmentation.
	// Defaults to false.
	fragmentation bool

	encoderOptions jsontext.Options
	decoderOptions jsontext.Options
}

// Option represents an option for configuring a Messager.
type Option func(*options)

// WithMaxSize sets the maximum size of a single frame in bytes. maxSize
// must not exceed LengthMask (~1GiB); NewMessager panics otherwise, since
// bits 30 and 31 of the frame header are reserved for FlagMore/FlagFragment
// and cannot represent a larger single-frame length.
func WithMaxSize(maxSize uint32) Option {
	return func(opts *options) {
		opts.maxSize = maxSize
	}
}

// WithFragmentSize sets the maximum size of a fragment frame payload in bytes.
// It defaults to MaxSize, but can be configured to be smaller. If fragmentSize
// is 0 or exceeds MaxSize, it is set to MaxSize.
func WithFragmentSize(fragmentSize uint32) Option {
	return func(opts *options) {
		opts.fragmentSize = fragmentSize
	}
}

// WithMaxMessageSize sets the maximum total size of a reassembled message in bytes.
// If 0, DefaultMaxMessageSize (100MB) is used.
func WithMaxMessageSize(maxSize uint32) Option {
	return func(opts *options) {
		opts.maxMessageSize = maxSize
	}
}

// WithMaxFragmentedMessageSize sets the maximum total size of a message that can
// be fragmented in bytes. If set (> 0), WriteMessage returns ErrMessageTooLarge
// if a message to be fragmented exceeds this size, and ReadMessage returns an
// error if the total reassembled size exceeds this limit. This prevents deadlocks
// when writing over buffered channels that could fill up before a complete request
// is sent.
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
	maxMessageSize           uint32
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
// NewMessager panics if maxSize exceeds LengthMask (see WithMaxSize).
func NewMessager(rd io.ReadCloser, wr io.Writer, opts ...Option) *Messager {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if o.maxSize == 0 {
		o.maxSize = DefaultMaxNativeMessageSize
	}
	if o.maxSize > LengthMask {
		panic(fmt.Sprintf("jsonmsgs: maxSize %d exceeds LengthMask %d", o.maxSize, LengthMask))
	}
	if o.fragmentSize == 0 || o.fragmentSize > o.maxSize {
		o.fragmentSize = o.maxSize
	}
	if o.maxMessageSize == 0 {
		o.maxMessageSize = DefaultMaxMessageSize
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
		maxMessageSize:           o.maxMessageSize,
		maxFragmentedMessageSize: o.maxFragmentedMessageSize,
		fragmentation:            o.fragmentation,
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

// WriteMessage writes a message to the underlying writer with a 4-byte little-endian
// length prefix. If fragmentation is enabled (via WithFragmentation) and the serialized
// message exceeds the configured fragment size (which defaults to MaxSize), it is
// transparently fragmented into multiple frames of at most fragment size bytes.
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
	if len(data) < 4 {
		return fmt.Errorf("buffer too small to write length prefix")
	}
	size := len(data) - 4
	if uint64(size) > uint64(m.maxMessageSize) {
		return fmt.Errorf("%w: message size %d exceeds maximum message size %d", ErrMessageTooLarge, size, m.maxMessageSize)
	}

	if (!m.fragmentation && uint32(size) <= m.maxSize) || (m.fragmentation && uint32(size) <= m.fragmentSize) {
		data[0] = byte(size)
		data[1] = byte(size >> 8)
		data[2] = byte(size >> 16)
		data[3] = byte(size >> 24)
		return writeFull(m.wr, data)
	}

	if !m.fragmentation {
		return fmt.Errorf("%w: message size %d exceeds maximum %d", ErrMessageTooLarge, size, m.maxSize)
	}

	if err := m.checkFragmentedLimit(uint64(size)); err != nil {
		return err
	}

	return m.writeFragmentedMessage(data, data[4:])
}

func (m *Messager) writeFragmentedMessage(data []byte, payload []byte) error {
	maxChunk := int(m.fragmentSize)
	for off := 0; off < len(payload); off += maxChunk {
		chunkLen := min(maxChunk, len(payload)-off)
		isLast := (off+chunkLen == len(payload))
		header := FlagFragment | uint32(chunkLen)
		if !isLast {
			header |= FlagMore
		}
		if off == 0 {
			data[0] = byte(header)
			data[1] = byte(header >> 8)
			data[2] = byte(header >> 16)
			data[3] = byte(header >> 24)
			if err := writeFull(m.wr, data[:4+chunkLen]); err != nil {
				return err
			}
			continue
		}
		// Combine the header and payload into a single contiguous write so
		// that unbuffered writers (e.g. a net.Conn) don't incur two Write
		// syscalls per fragment.
		need := 4 + chunkLen
		if cap(m.fragScratch) < need {
			m.fragScratch = make([]byte, need)
		} else {
			m.fragScratch = m.fragScratch[:need]
		}
		m.fragScratch[0] = byte(header)
		m.fragScratch[1] = byte(header >> 8)
		m.fragScratch[2] = byte(header >> 16)
		m.fragScratch[3] = byte(header >> 24)
		copy(m.fragScratch[4:], payload[off:off+chunkLen])
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
	// Read the length of the message as a 4-byte little-endian integer.
	var lenBytes [4]byte
	if _, err := io.ReadFull(m.rd, lenBytes[:]); err != nil {
		return nil, err
	}
	header := uint32(lenBytes[0]) |
		uint32(lenBytes[1])<<8 |
		uint32(lenBytes[2])<<16 |
		uint32(lenBytes[3])<<24

	isFragment := (header & FlagFragment) != 0
	hasMore := (header & FlagMore) != 0
	length := header & LengthMask

	if length > m.maxSize {
		return nil, fmt.Errorf("%w: message size %d exceeds maximum %d", ErrMessageTooLarge, length, m.maxSize)
	}

	dec := m.decPool.Get().(*Decoder)

	if !isFragment {
		if cap(dec.buffer) < int(length) {
			dec.buffer = make([]byte, length)
		} else {
			dec.buffer = dec.buffer[:length]
		}
		if _, err := io.ReadFull(m.rd, dec.buffer); err != nil {
			m.decPool.Put(dec)
			return nil, err
		}
		*dec.buf = *bytes.NewBuffer(dec.buffer)
		dec.Reset(dec.buf, dec.opts)
		return dec, nil
	}

	if err := m.readFragmentedMessage(dec, length, hasMore); err != nil {
		m.decPool.Put(dec)
		return nil, err
	}

	*dec.buf = *bytes.NewBuffer(dec.buffer)
	dec.Reset(dec.buf, dec.opts)
	return dec, nil
}

func (m *Messager) checkFragmentedLimit(totalLen uint64) error {
	if m.maxFragmentedMessageSize > 0 && totalLen > uint64(m.maxFragmentedMessageSize) {
		return fmt.Errorf("%w: message size %d exceeds maximum fragmented message size %d", ErrMessageTooLarge, totalLen, m.maxFragmentedMessageSize)
	}
	if totalLen > uint64(m.maxMessageSize) {
		return fmt.Errorf("%w: message size %d exceeds maximum message size %d", ErrMessageTooLarge, totalLen, m.maxMessageSize)
	}
	return nil
}

func (m *Messager) readFragmentedMessage(dec *Decoder, firstLen uint32, hasMore bool) error {
	if !m.fragmentation {
		return fmt.Errorf("%w: received fragmented message when fragmentation is disabled", ErrMessageTooLarge)
	}

	totalLen := uint64(firstLen)
	if err := m.checkFragmentedLimit(totalLen); err != nil {
		return err
	}

	if cap(dec.buffer) < int(firstLen) {
		dec.buffer = make([]byte, firstLen)
	} else {
		dec.buffer = dec.buffer[:firstLen]
	}
	if _, err := io.ReadFull(m.rd, dec.buffer); err != nil {
		return err
	}

	for hasMore {
		fragLen, more, err := m.readFragmentHeader()
		if err != nil {
			return err
		}
		hasMore = more
		totalLen += uint64(fragLen)
		if err := m.checkFragmentedLimit(totalLen); err != nil {
			return err
		}
		if err := m.appendChunk(dec, fragLen); err != nil {
			return err
		}
	}
	return nil
}

func (m *Messager) readFragmentHeader() (fragLen uint32, hasMore bool, err error) {
	var lenBytes [4]byte
	if _, err := io.ReadFull(m.rd, lenBytes[:]); err != nil {
		return 0, false, err
	}
	hdr := uint32(lenBytes[0]) |
		uint32(lenBytes[1])<<8 |
		uint32(lenBytes[2])<<16 |
		uint32(lenBytes[3])<<24

	if (hdr & FlagFragment) == 0 {
		return 0, false, fmt.Errorf("jsonmsgs: expected fragment frame, got unfragmented frame")
	}
	fragLen = hdr & LengthMask
	if fragLen == 0 {
		return 0, false, fmt.Errorf("jsonmsgs: zero-length fragment")
	}
	if fragLen > m.maxSize {
		return 0, false, fmt.Errorf("%w: fragment size %d exceeds maximum %d", ErrMessageTooLarge, fragLen, m.maxSize)
	}
	return fragLen, (hdr & FlagMore) != 0, nil
}

func (m *Messager) appendChunk(dec *Decoder, fragLen uint32) error {
	currLen := len(dec.buffer)
	newLen := currLen + int(fragLen)
	if cap(dec.buffer) < newLen {
		limit := uint64(m.maxMessageSize)
		if m.maxFragmentedMessageSize > 0 && uint64(m.maxFragmentedMessageSize) < limit {
			limit = uint64(m.maxFragmentedMessageSize)
		}
		growCap := min(uint64(newLen)*2, limit)
		newBuf := make([]byte, newLen, growCap)
		copy(newBuf, dec.buffer)
		dec.buffer = newBuf
	} else {
		dec.buffer = dec.buffer[:newLen]
	}
	_, err := io.ReadFull(m.rd, dec.buffer[currLen:newLen])
	return err
}
