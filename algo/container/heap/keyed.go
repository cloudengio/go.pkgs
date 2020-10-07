// Copyright 2020 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package heap contains various implementations of heap containers.
package heap

import (
	"bytes"
	"container/heap"
	"encoding/gob"
	"encoding/json"
	"sync"
)

type keyInt64 struct {
	Key   string `json:"k"`
	Value int64  `json:"v"`
}

// KeyedInt64 implements a heap whose values include both a key and
// value to allow for updates to existing items in the heap. It also
// keeps a running sum of the all of the items currently in the heap,
// supports both ascending and desencding operations. It is safe for
// concurrent use.
type KeyedInt64 struct {
	mu sync.Mutex
	h  *keyedInt64
}

// NewKeyedInt64
func NewKeyedInt64(descending bool) *KeyedInt64 {
	return &KeyedInt64{
		h: &keyedInt64{
			Descending: descending,
			Lookup:     map[string]int{},
		},
	}
}

type keyedInt64 struct {
	Descending bool           `json:"d,omitempty"`
	Lookup     map[string]int `json:"l"`
	Values     []keyInt64     `json:"v"`
	Total      int64          `json:"t"`
}

func (kih *keyedInt64) Len() int {
	return len(kih.Values)
}

func (kih *keyedInt64) Less(i, j int) bool {
	ascending := (kih.Values[i].Value < kih.Values[j].Value)
	if kih.Descending {
		return !ascending
	}
	return ascending
}

func (kih *keyedInt64) Swap(i, j int) {
	kih.Lookup[kih.Values[i].Key] = j
	kih.Lookup[kih.Values[j].Key] = i
	kih.Values[i], kih.Values[j] = kih.Values[j], kih.Values[i]
}

func (kih *keyedInt64) Push(x interface{}) {
	m := x.(keyInt64)
	kih.Values = append(kih.Values, m)
	kih.Lookup[m.Key] = len(kih.Values) - 1
}

func (kih *keyedInt64) Pop() interface{} {
	old := kih.Values
	n := len(old)
	x := old[n-1]
	kih.Values = old[0 : n-1]
	kih.Total -= x.Value
	delete(kih.Lookup, x.Key)
	return x
}

func (kih *keyedInt64) update(kv keyInt64) {
	idx, ok := kih.Lookup[kv.Key]
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
	idx, ok := kih.Lookup[key]
	if !ok {
		return
	}
	heap.Remove(kih, idx)
}

func (ki *KeyedInt64) Update(key string, value int64) {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	ki.h.update(keyInt64{Key: key, Value: value})
}

func (ki *KeyedInt64) Pop() (string, int64) {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	kv := heap.Pop(ki.h).(keyInt64)
	return kv.Key, kv.Value
}

func (ki *KeyedInt64) TopN(n int) []struct {
	Key   string
	Value int64
} {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	if n >= len(ki.h.Values) {
		n = len(ki.h.Values)
	}
	out := make([]struct {
		Key   string
		Value int64
	}, n)
	for i := 0; i < n; i++ {
		kv := heap.Pop(ki.h).(keyInt64)
		out[i].Key = kv.Key
		out[i].Value = kv.Value
	}
	return out
}

func (ki *KeyedInt64) Total() int64 {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	return ki.h.Total
}

func (ki *KeyedInt64) Len() int {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	return len(ki.h.Values)
}

func (ki *KeyedInt64) Remove(key string) {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	ki.h.remove(key)
}

func (ki *KeyedInt64) GobDecode(buf []byte) error {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	dec := gob.NewDecoder(bytes.NewBuffer(buf))
	return dec.Decode(&ki.h)
}

func (ki *KeyedInt64) GobEncode() ([]byte, error) {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(ki.h); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (ki *KeyedInt64) UnmarshalJSON(buf []byte) error {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	dec := json.NewDecoder(bytes.NewBuffer(buf))
	return dec.Decode(&ki.h)
}

func (ki *KeyedInt64) MarshalJSON() ([]byte, error) {
	ki.mu.Lock()
	defer ki.mu.Unlock()
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	if err := enc.Encode(ki.h); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
