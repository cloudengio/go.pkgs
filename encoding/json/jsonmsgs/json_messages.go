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

// DefaultMaxNativeMessageSize is the default maximum size of a message, in bytes.
const DefaultMaxNativeMessageSize = 1024 * 1024 // 1MB

var (
	ErrMessageTooLarge = errors.New("jsonmsgs: message too large")
)

type options struct {
	// MaxSize specifies the maximum size of a message in bytes.
	// If MaxSize is 0, the default maximum size of 1MB is used.
	maxSize        uint32
	encoderOptions jsontext.Options
	decoderOptions jsontext.Options
}

// Option represents an option for configuring a Messager.
type Option func(*options)

// WithMaxSize sets the maximum size of a message in bytes.
func WithMaxSize(maxSize uint32) Option {
	return func(opts *options) {
		opts.maxSize = maxSize
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
	buffer *bytes.Buffer
}

// Decoder captures the state to decode a single message.
// It is created and returned by Messager.ReadMessage and must released by
// calling Messager.ReleaseDecoder after which it cannot be used again.
type Decoder struct {
	*jsontext.Decoder
	buf    *bytes.Buffer
	buffer []byte
}

type Messager struct {
	rd      io.ReadCloser
	wr      io.Writer
	maxSize uint32
	encPool sync.Pool
	decPool sync.Pool
	wmu     sync.Mutex
	rmu     sync.Mutex
}

// NewMessager creates a new Messager with the given writer and readCloser.
// If maxSize is not specified via WithMaxSize, DefaultMaxNativeMessageSize (1MB) is used.
func NewMessager(wr io.Writer, rd io.ReadCloser, opts ...Option) *Messager {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if o.maxSize == 0 {
		o.maxSize = DefaultMaxNativeMessageSize
	}
	if wr == nil {
		wr = io.Discard
	}
	if rd == nil {
		rd = io.NopCloser(bytes.NewReader(nil))
	}
	nm := &Messager{
		wr:      wr,
		rd:      rd,
		maxSize: o.maxSize,
	}
	nm.encPool = sync.Pool{
		New: func() any {
			buf := bytes.NewBuffer(make([]byte, 0, 1024))
			buf.Write([]byte{0, 0, 0, 0})
			return &Encoder{
				Encoder: jsontext.NewEncoder(buf, o.encoderOptions),
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
	enc.Reset(enc.buffer)
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

// WriteMessage writes a message to the underlying writer with a 4-byte little-endian
// length prefix. The encoder is returned to the pool after use regardless of error.
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
	if uint32(size) > m.maxSize {
		return fmt.Errorf("%w: message size %d exceeds maximum %d", ErrMessageTooLarge, size, m.maxSize)
	}
	data[0] = byte(size)
	data[1] = byte(size >> 8)
	data[2] = byte(size >> 16)
	data[3] = byte(size >> 24)
	for off := 0; off < len(data); {
		n, err := m.wr.Write(data[off:])
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
	/*
			n, err := m.wr.Write(data)
			if err != nil {
				return fmt.Errorf("failed to write complete message: %w", err)
			}
		if n != len(data) {
			return fmt.Errorf("failed to write complete message: wrote %d of %d bytes", n, len(data))
		}*/
	return nil
}

// ReadMessage reads a message from the underlying reader, returning a
// Decoder that can be used to decode the message. The Decoder must be released
// by calling ReleaseDecoder when no longer needed. ReadMessage will block
// until a complete message is read or an error occurs.
func (m *Messager) ReadMessage() (*Decoder, error) {
	m.rmu.Lock()
	defer m.rmu.Unlock()
	// Read the length of the message as a 4-byte little-endian integer.
	var lenBytes [4]byte
	if _, err := io.ReadFull(m.rd, lenBytes[:]); err != nil {
		return nil, err
	}
	length := uint32(lenBytes[0]) |
		uint32(lenBytes[1])<<8 |
		uint32(lenBytes[2])<<16 |
		uint32(lenBytes[3])<<24
	if length > m.maxSize {
		return nil, fmt.Errorf("%w: message size %d exceeds maximum %d", ErrMessageTooLarge, length, m.maxSize)
	}
	dec := m.decPool.Get().(*Decoder)
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
	dec.Reset(dec.buf)
	return dec, nil
}
