// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package jsonpayload

import (
	"sync"

	"cloudeng.io/types"
)

var (
	decodeRegistry sync.Map
)

func RegisterType[T any, PT *T]() {
	tn := types.TypeName[T]()
	fn := func() any {
		return PT(new(T))
	}
	decodeRegistry.Store(tn, fn)
}

func New(typeName string) (any, bool) {
	raw, ok := decodeRegistry.Load(typeName)
	if !ok {
		return nil, false
	}
	fn := raw.(func() any)
	return fn(), true
}

/*

type callbacks struct {
	constructor func() any
	event       EventCallback
	request     RequestCallback
}

type typeRegistry struct {
	types sync.Map
}

func newTypeRegistry() *typeRegistry {
	return &typeRegistry{}
}

func (tr *typeRegistry) registerType[T any, PT extprotocol.JSONUnmarshaler[T]](event EventCallback, request RequestCallback) {
	typeName := extprotocol.TypeName[T]()
	tr.types.Store(typeName, &callbacks{
		constructor: func() any {
			return PT(new(T))
		},
		event:   event,
		request: request,
	})
}

func (tr *typeRegistry) decode(typeName string, value jsontext.Value) (any, EventCallback, RequestCallback, bool, error) {
	raw, exists := tr.types.Load(typeName)
	if !exists {
		return nil, nil, nil, false, nil
	}
	cb := raw.(*callbacks)
	instance := cb.constructor().(json.UnmarshalerFrom)
	dec := jsontext.NewDecoder(bytes.NewBuffer(value))
	if err := instance.UnmarshalJSONFrom(dec); err != nil {
		return nil, nil, nil, false, err
	}
	return instance, cb.event, cb.request, true, nil
}
*/
