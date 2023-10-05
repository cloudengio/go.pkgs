// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package heap contains various implementations of heap containers.
package heap

import (
	"bytes"
	"compress/flate"
	"container/heap"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"sync"

	"cloudeng.io/errors"
)

type keyInt64 struct {
	Key   string `json:"k"`
	Value int64  `json:"v"`
}

// KeyedInt64 implements a heap whose values include both a key and
// value to allow for updates to existing items in the heap. It also
// keeps a running sum of the all of the values currently in the heap,
// supports both ascending and descending operations and is safe for
// concurrent use.
type KeyedInt64 struct {
	mu sync.Mutex
	h  *keyedInt64
}

// Order determines if the heap is maintained in ascending or descending
// order.
type Order bool

// Values for Order.
const (
	Ascending  Order = false
	Descending Order = true
)

// NewKeyedInt64 returns a new instance of KeyedInt64.
func NewKeyedInt64(order Order) *KeyedInt64 {
	return &KeyedInt64{
		h: &keyedInt64{
			Order:  order,
			lookup: map[string]int{},
		},
	}
}

type keyedInt64 struct {
	Order  Order
	Values []keyInt64
	Total  int64
	lookup map[string]int
}

func (kih *keyedInt64) Len() int {
	return len(kih.Values)
}

func (kih *keyedInt64) Less(i, j int) bool {
	ascending := (kih.Values[i].Value < kih.Values[j].Value)
	if kih.Order == Descending {
		return !ascending
	}
	return ascending
}

func (kih *keyedInt64) Swap(i, j int) {
	kih.lookup[kih.Values[i].Key] = j
	kih.lookup[kih.Values[j].Key] = i
	kih.Values[i], kih.Values[j] = kih.Values[j], kih.Values[i]
}

func (kih *keyedInt64) Push(x interface{}) {
	m := x.(keyInt64)
	kih.Values = append(kih.Values, m)
	kih.lookup[m.Key] = len(kih.Values) - 1
}

func (kih *keyedInt64) Pop() interface{} {
	old := kih.Values
	n := len(old)
	x := old[n-1]
	kih.Values = old[0 : n-1]
	kih.Total -= x.Value
	delete(kih.lookup, x.Key)
	return x
}

func (kih *keyedInt64) update(kv keyInt64) {
	idx, ok := kih.lookup[kv.Key]
	if !ok {
		heap.Push(kih, kv)
		kih.Total += kv.Value
		return
	}
	kih.Total += kv.Value - kih.Values[idx].Value
	kih.Values[idx] = kv
	heap.Fix(kih, idx)
}

func (kih *keyedInt64) remove(key string) {
	idx, ok := kih.lookup[key]
	if !ok {
		return
	}
	heap.Remove(kih, idx)
}

// Update updates the value associated with key or it adds it to
// the heap.
func (ki *KeyedInt64) Update(key string, value int64) {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	ki.h.update(keyInt64{Key: key, Value: value})
}

// Pop removes the top most value (either largest or smallest) from
// the heap.
func (ki *KeyedInt64) Pop() (string, int64) {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	kv := heap.Pop(ki.h).(keyInt64)
	return kv.Key, kv.Value
}

// TopN removes at most the top most n items from the heap.
func (ki *KeyedInt64) TopN(n int) []struct {
	K string
	V int64
} {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	if n >= len(ki.h.Values) {
		n = len(ki.h.Values)
	}
	out := make([]struct {
		K string
		V int64
	}, n)
	for i := 0; i < n; i++ {
		kv := heap.Pop(ki.h).(keyInt64)
		out[i].K = kv.Key
		out[i].V = kv.Value
	}
	return out
}

// Sum returns the current sum of all values in the heap.
func (ki *KeyedInt64) Sum() int64 {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	return ki.h.Total
}

// Len returns the number of items in the heap.
func (ki *KeyedInt64) Len() int {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	return len(ki.h.Values)
}

// Remove removes the specified item from the heap.
func (ki *KeyedInt64) Remove(key string) {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	ki.h.remove(key)
}

// GobEncode implements gob.GobEncode.
func (ki *KeyedInt64) GobEncode() ([]byte, error) {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	errs := errors.M{}

	// Prepare two buffers, one for values, the other for keys.
	// Write values as varint's and keys as compressed strings.
	nVals := len(ki.h.Values)
	keyBuf := bytes.NewBuffer(make([]byte, 0, nVals*16))
	keyWriter, err := flate.NewWriter(keyBuf, flate.BestCompression)
	if err != nil {
		return nil, err
	}
	keyEnc := gob.NewEncoder(keyWriter)
	valBuf := make([]byte, 0, nVals*3)
	valIdx := 0
	for _, v := range ki.h.Values {
		var b [binary.MaxVarintLen64]byte
		n := binary.PutVarint(b[:], v.Value)
		valIdx += n
		valBuf = append(valBuf, b[:n]...)
		errs.Append(keyEnc.Encode(v.Key))
	}
	valBuf = valBuf[:valIdx]
	errs.Append(keyWriter.Flush())
	errs.Append(keyWriter.Close())
	if err := errs.Err(); err != nil {
		return nil, err
	}

	// Write out the gob encodings of the key and value bufs and
	// associated metadata.
	sz := keyBuf.Len() + valIdx + 64
	buf := bytes.NewBuffer(make([]byte, 0, sz))
	enc := gob.NewEncoder(buf)
	errs.Append(enc.Encode(len(ki.h.Values)))
	errs.Append(enc.Encode(ki.h.Order))
	errs.Append(enc.Encode(ki.h.Total))
	errs.Append(enc.Encode(keyBuf.Bytes()))
	errs.Append(enc.Encode(valBuf))
	return buf.Bytes(), errs.Err()
}

// GobDecode implements gob.GobDecoder.
func (ki *KeyedInt64) GobDecode(buf []byte) error {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	dec := gob.NewDecoder(bytes.NewBuffer(buf))
	errs := errors.M{}
	var size int
	errs.Append(dec.Decode(&size))
	ki.h = &keyedInt64{
		Values: make([]keyInt64, size),
		lookup: make(map[string]int, size),
	}
	errs.Append(dec.Decode(&ki.h.Order))
	errs.Append(dec.Decode(&ki.h.Total))
	var keyBuf, valBuf []byte
	errs.Append(dec.Decode(&keyBuf))
	errs.Append(dec.Decode(&valBuf))

	keyReader := flate.NewReader(bytes.NewBuffer(keyBuf))
	keyDec := gob.NewDecoder(keyReader)
	valIdx := 0
	for i := 0; i < size; i++ {
		var n int
		ki.h.Values[i].Value, n = binary.Varint(valBuf[valIdx:])
		valIdx += n
		var key string
		errs.Append(keyDec.Decode(&key))
		ki.h.Values[i].Key = key
		ki.h.lookup[key] = i
	}
	return errs.Err()
}

type jsonEncoding struct {
	Size   int             `json:"size"`
	Order  Order           `json:"order"`
	Total  int64           `json:"total"`
	Values json.RawMessage `json:"values"`
}

// MarshalJSON implements json.Marshaler.
func (ki *KeyedInt64) MarshalJSON() ([]byte, error) {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	errs := errors.M{}
	valbuf := &bytes.Buffer{}
	enc := json.NewEncoder(valbuf)
	errs.Append(enc.Encode(ki.h.Values))
	buf := &bytes.Buffer{}
	enc = json.NewEncoder(buf)
	errs.Append(enc.Encode(jsonEncoding{
		Size:   len(ki.h.Values),
		Order:  ki.h.Order,
		Total:  ki.h.Total,
		Values: valbuf.Bytes(),
	}))
	return buf.Bytes(), errs.Err()
}

// UnmarshalJSON implements json.Unmarshaler.
func (ki *KeyedInt64) UnmarshalJSON(buf []byte) error {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	dec := json.NewDecoder(bytes.NewBuffer(buf))
	hdr := jsonEncoding{}
	errs := errors.M{}
	errs.Append(dec.Decode(&hdr))
	ki.h = &keyedInt64{
		Order:  hdr.Order,
		Total:  hdr.Total,
		Values: make([]keyInt64, hdr.Size),
		lookup: make(map[string]int, hdr.Size),
	}
	dec = json.NewDecoder(bytes.NewBuffer(hdr.Values))
	errs.Append(dec.Decode(&ki.h.Values))
	for i, k := range ki.h.Values {
		ki.h.lookup[k.Key] = i
	}
	return errs.Err()
}
