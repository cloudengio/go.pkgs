// Copyright 2023 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package content provides support for working with different content types.
// In particular it defines a mean of specifiying content types and a registry
// for matching content types against handlers
package content

import (
	"fmt"
	"sync"
)

// Registry provides a means of registering and looking up handlers for
// converting between content types.
type Registry struct {
	mu sync.Mutex
	db map[string]map[string]interface{}
}

// NewRegistry returns a new instance of Registry.
func NewRegistry() *Registry {
	return &Registry{
		db: make(map[string]map[string]interface{}),
	}
}

func fromTo(from, to Type) (string, string, error) {
	ft, err := ParseType(from)
	if err != nil {
		return "", "", err
	}
	tt, err := ParseType(to)
	if err != nil {
		return "", "", err
	}
	return ft, tt, nil
}

// Lookup returns the handler registered for converting from one content type
// to another, and the parameter and value associated with the to type. The
// parameter to the from type is ignored.
func (c *Registry) Lookup(from, to Type) (parameter, value string, handler interface{}, err error) {
	ft, err := ParseType(from)
	if err != nil {
		return "", "", nil, err
	}
	tt, parameter, value, err := ParseTypeFull(to)
	if err != nil {
		return "", "", nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	handler, ok := c.db[ft][tt]
	if !ok {
		return "", "", nil, fmt.Errorf("no handler for %v to %v", from, to)
	}
	return parameter, value, handler, nil
}

// Register registers a handler for converting from one content type to another.
func (c *Registry) Register(from, to Type, handler interface{}) error {
	ft, tt, err := fromTo(from, to)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.db[ft] == nil {
		c.db[ft] = make(map[string]interface{})
	}
	if _, ok := c.db[ft][tt]; ok {
		return fmt.Errorf("handler already registered for %v to %v", from, to)
	}
	c.db[ft][tt] = handler
	return nil
}
