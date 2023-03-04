// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package content

import (
	"fmt"
	"sync"
)

// Registry provides a means of registering and looking up handlers for
// processing content types and for converting between content types.
type Registry[T any] struct {
	mu         sync.Mutex
	converters map[Type]map[Type]T
	handlers   map[Type][]T
}

// NewRegistry returns a new instance of Registry.
func NewRegistry[T any]() *Registry[T] {
	return &Registry[T]{
		converters: make(map[Type]map[Type]T),
		handlers:   make(map[Type][]T),
	}
}

// LookupConverters returns the converter registered for converting the 'from'
// content type to the 'to' content type. The returned handlers are in the same
// order as that registered via RegisterConverter.
func (c *Registry[T]) LookupConverters(from, to Type) (T, error) {
	from, to = Clean(from), Clean(to)
	c.mu.Lock()
	defer c.mu.Unlock()
	handler, ok := c.converters[from][to]
	if !ok {
		var t T
		return t, fmt.Errorf("no converter for %v to %v", from, to)
	}
	return handler, nil
}

// RegisterConverters registers a lust of handlers for converting from one
// content type to another. The caller of LookupConverter must decide which
// converter to use.
func (c *Registry[T]) RegisterConverters(from, to Type, converter T) error {
	from, to = Clean(from), Clean(to)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.converters[from] == nil {
		c.converters[from] = make(map[Type]T)
	}
	if _, ok := c.converters[from][to]; ok {
		return fmt.Errorf("converter already registered for %v to %v", from, to)
	}
	c.converters[from][to] = converter
	return nil
}

// LookupHandlers returns the handler registered for the given content type.
func (c *Registry[T]) LookupHandlers(ctype Type) ([]T, error) {
	ctype = Clean(ctype)
	c.mu.Lock()
	defer c.mu.Unlock()
	handlers, ok := c.handlers[Clean(ctype)]
	if !ok {
		return nil, fmt.Errorf("no handler for %v", ctype)
	}
	return handlers, nil
}

// RegisterHandlers registers a handler for a given content type. The caller of
// LookupHandlers must decide which converter to use.
func (c *Registry[T]) RegisterHandlers(ctype Type, handlers ...T) error {
	ctype = Clean(ctype)
	_, _, _, err := ParseTypeFull(ctype)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, ok := c.handlers[ctype]; ok {
		return fmt.Errorf("handler already registered for %v", ctype)
	}
	c.handlers[Clean(ctype)] = handlers
	return nil
}
