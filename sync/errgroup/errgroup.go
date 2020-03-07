// Copyright 2020 cloudeng LLC. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

// Package errgroup simplifies common patterns of goroutine use, in particular
// making it straightforward to reliably wait on parallel or pipelined
// goroutines, exiting either when the first error is encountered or waiting
// for all goroutines to finish regardless of error outcome. Contexts are
// used to control cancelation. It is modeled on golang.org/x/sync/errgroup and
// other similar packages. It makes use of cloudeng.io/errors to simplify
// collecting multiple errors.
package errgroup

import (
	"context"
	"sync"

	"cloudeng.io/errors"
)

// T represents a set of goroutines working on some common
// coordinated sets of tasks.
//
// T may be instantiated directly, in which case, all go routines will run
// to completion and all errors will be collected and made available vie the
// Errors field and the return value of Wait. Alternatively WithContext can be
// used to create Group with an embedded cancel function that will be called
// once either when the first error occurs or when Wait is called. WithCancel
// behaves like WithContext but allows both the context and cancel function
// to be supplied which is required for working with context.WithDeadline
// and context.WithTimeout.
//
type T struct {
	wg         sync.WaitGroup
	cancelFunc func()
	cancelOnce sync.Once
	errors     errors.M
}

// WithContext returns a new Group that will call the cancel function
// derived from the supplied context once on either a first non-nil error
// being returned by a goroutine or when Wait is called.
func WithContext(ctx context.Context) (*T, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return &T{cancelFunc: cancel}, ctx
}

// WithCancel returns a new T that will call the supplied cancel function once
// on either a first non-nil error being returned or when Wait is called.
func WithCancel(cancel func()) *T {
	return &T{cancelFunc: cancel}
}

func (g *T) possiblyCancel() {
	g.cancelOnce.Do(func() {
		if g.cancelFunc != nil {
			g.cancelFunc()
		}
	})
}

// Go runs the supplied function from a goroutine.
func (g *T) Go(f func() error) {
	g.wg.Add(1)

	go func() {
		defer g.wg.Done()
		if err := f(); err != nil {
			g.errors.Append(err)
			g.possiblyCancel()
		}
	}()
}

// Wait waits for all goroutines to
func (g *T) Wait() error {
	g.wg.Wait()
	g.possiblyCancel()
	return g.errors.Err()
}
