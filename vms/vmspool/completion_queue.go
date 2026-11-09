// Copyright 2026 cloudeng llc. All rights reserved.
// Use of this source code is governed by the Apache-2.0
// license that can be found in the LICENSE file.

package vmspool

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"sync"
	"time"

	"cloudeng.io/errors"
	"cloudeng.io/logging/ctxlog"
	"cloudeng.io/sync/ctxsync"
	"cloudeng.io/sync/patterns"
)

// CompletedVM represents the operations available on a completed VM.
type CompletedVM interface {
	// GetVM returns the VM the work ran on, or nil if it had none, in which
	// case there is nothing for the queue to release.
	GetVM() *VM
	// GetLogger returns the logger to report this event with. The logger
	// is likely preconfigured with per-vm, per-job fields.
	GetLogger(*slog.Logger) *slog.Logger
}

// CompletionEvent represents a single occurrence of a completed VM any error
// that occurred related to the execution of the VM, as opposed to the outcome
// of the work the VM was performing which is captured in the Outcome field.
type CompletionEvent struct {
	expirationTime time.Time
	VM             CompletedVM
	Err            error  // The error that occurred during the execution of the VM, if any.
	Outcome        string // The outcome of the work performed by the VM, used to categorize the completion event.
}

// CompletionQueue parks finished work, and the VM that ran it, for a period of
// time before the VM is deleted, so that it can be inspected after the fact.
// Work is queued by outcome, each outcome configured with its own capacity and
// retention by WithCompletionOutcome, so that a failure can be kept for longer
// than a success, which is usually what is wanted.
//
// A single completion queue may be shared across multiple VM pools; it reports
// the VMs it is holding both in total and per pool, so that each pool can
// account for the VMs it created that the queue has yet to release.
type CompletionQueue struct {
	// ctx is the context the FIFOs were created with, and hence governs their
	// goroutines. Close consults it to tell a live queue, which will flush and
	// close its output channel, from one whose goroutines have already exited.
	ctx    context.Context
	logger *slog.Logger
	pubsub *patterns.PubSub[struct{}]

	closeOnce sync.Once
	closeErr  error

	// pushMu guards the sends into the outcome queues against Close closing
	// their input channels: a send racing that close panics. Push holds it for
	// reading for the whole of its send, and Close takes it for writing, which
	// both shuts out later pushes and waits for those already under way.
	pushMu sync.RWMutex
	closed bool // GUARDED by pushMu

	mu            sync.Mutex
	outcomeQueues map[string]*outcomeQueue
	vmIDs         map[string]bool
	vmIDsByPool   map[string][]string

	// wg tracks in-flight asynchronous VM deletions triggered on expiration.
	wg ctxsync.WaitGroup
}

type outcomeQueue struct {
	cq         *patterns.FIFO[CompletionEvent]
	capacity   int
	expiration time.Duration
}

// CompletionOutcomeOption represents an option for configuring a completion
// outcome in the queue.
type CompletionOutcomeOption func(*CompletionQueue)

// WithCompletionOutcome configures a completion outcome in the queue with the
// given name, capacity, and expiration duration.
func WithCompletionOutcome(name string, capacity int, expiration time.Duration) CompletionOutcomeOption {
	return func(q *CompletionQueue) {
		q.outcomeQueues[name] = &outcomeQueue{
			capacity:   capacity,
			expiration: expiration,
		}
	}
}

// NewCompletionQueue returns a queue with one sub-queue per outcome configured
// by WithCompletionOutcome, each holding at most that outcome's capacity of
// events and retaining each for that outcome's expiration before deleting the
// VM it was holding. An event whose Outcome names no configured sub-queue is
// discarded by Push, so a queue created with no options holds nothing.
//
// The queue's goroutines are governed by ctx, so cancelling it ends them;
// Close copes with a queue whose context has already been cancelled, which is
// what happens on an interrupt.
func NewCompletionQueue(ctx context.Context, opts ...CompletionOutcomeOption) *CompletionQueue {
	q := &CompletionQueue{
		ctx:           ctx,
		logger:        ctxlog.Logger(ctx),
		pubsub:        patterns.New[struct{}](),
		vmIDs:         make(map[string]bool),
		vmIDsByPool:   map[string][]string{},
		outcomeQueues: make(map[string]*outcomeQueue),
	}
	for _, opt := range opts {
		opt(q)
	}
	for _, cq := range q.outcomeQueues {
		cq.cq = patterns.NewFIFO(q.ctx, cq.capacity,
			patterns.WithPeriodicScan(time.Second, q.expiration))
	}
	return q
}

// eventVM returns the VM an event is holding, or nil if it is holding none,
// including when the event carries no CompletedVM at all.
func eventVM(e CompletionEvent) *VM {
	if e.VM == nil {
		return nil
	}
	vm := e.VM.GetVM()
	if vm == nil || vm.ID() == "" {
		return nil
	}
	return vm
}

func (q *CompletionQueue) recordVM(e CompletionEvent) {
	vm := eventVM(e)
	if vm == nil {
		return
	}
	id, pn := vm.ID(), vm.PoolName()
	q.mu.Lock()
	q.vmIDs[id] = true
	// A VM is recorded once however many events name it, so that the per-pool
	// list stays a set and its length keeps agreeing with the count of VMs
	// being held.
	if !slices.Contains(q.vmIDsByPool[pn], id) {
		q.vmIDsByPool[pn] = append(q.vmIDsByPool[pn], id)
	}
	q.mu.Unlock()
	q.pubsub.Publish(struct{}{})
}

func (q *CompletionQueue) unrecordVM(e CompletionEvent) {
	vm := eventVM(e)
	if vm == nil {
		return
	}
	q.Remove(vm.PoolName(), vm.ID())
}

// expiration reports whether an event has been held for long enough, and
// deletes the VM it was holding if it has. It runs on the FIFO's own goroutine;
// deletion runs asynchronously in a tracked goroutine so that a slow deletion
// does not block delivery of other events or expiration of other VMs.
func (q *CompletionQueue) expiration(e CompletionEvent) bool {
	if time.Now().After(e.expirationTime) {
		logger := q.logger
		if e.VM != nil {
			logger = e.VM.GetLogger(q.logger)
		}
		if e.Err != nil {
			logger.Error("completion event expired for failed work", "error", e.Err)
		} else {
			logger.Info("completion event expired for successful work")
		}
		q.unrecordVM(e)
		if vm := eventVM(e); vm != nil {
			q.wg.Go(func() {
				ctx, cancel := context.WithTimeout(context.Background(), DefaultStopTimeout)
				defer cancel()
				if err := vm.Delete(ctx); err != nil {
					logger.Error("failed to delete VM", "deletion_err", err)
				}
			})
		}
		return true
	}
	return false
}

// Push adds a completion event to the queue for its specified outcome. An
// event whose Outcome names no configured outcome is discarded, as is one
// pushed to a closed queue or to one whose context has been cancelled: in each
// case nothing would ever release the VM, which is left to the pool that
// created it to delete.
func (q *CompletionQueue) Push(event CompletionEvent) {
	q.pushMu.RLock()
	defer q.pushMu.RUnlock()
	if q.closed {
		return
	}
	cq, ok := q.outcomeQueues[event.Outcome]
	if !ok {
		return
	}
	event.expirationTime = time.Now().Add(cq.expiration)
	q.recordVM(event)
	select {
	case cq.cq.In() <- event:
	case <-q.ctx.Done():
		// The FIFO's goroutine has already returned, so the send would never
		// be received and this call would never return.
		q.unrecordVM(event)
	}
}

// Out returns the channel of events for the requested outcome, or nil if no
// such outcome was configured or the queue is closed.
func (q *CompletionQueue) Out(outcome string) <-chan CompletionEvent {
	q.pushMu.RLock()
	defer q.pushMu.RUnlock()
	if q.closed {
		return nil
	}
	cq, ok := q.outcomeQueues[outcome]
	if !ok {
		return nil
	}
	return cq.cq.Out()
}

// VMIDs returns a copy of the set of VM IDs currently parked in the queue, in
// no particular order.
func (q *CompletionQueue) VMIDs() []string {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]string, 0, len(q.vmIDs))
	for k := range q.vmIDs {
		out = append(out, k)
	}
	return out
}

// VMIDsForPool returns a copy of the set of VM IDs currently parked in the
// queue for the specified pool.
func (q *CompletionQueue) VMIDsForPool(pool string) []string {
	q.mu.Lock()
	defer q.mu.Unlock()
	if list, ok := q.vmIDsByPool[pool]; ok {
		return slices.Clone(list)
	}
	return nil
}

// NumEvents returns the number of VMs the queue is holding across all of its
// outcomes. An event that was pushed without a VM is not counted: there is
// nothing for the queue to release and nothing for a caller to report.
func (q *CompletionQueue) NumEvents() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.vmIDs)
}

// NumEventsPerPool returns the number of VMs the queue is holding that came
// from the named pool, counted as NumEvents counts them.
func (q *CompletionQueue) NumEventsPerPool(poolName string) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.vmIDsByPool[poolName])
}

// Remove stops tracking the VM, for a caller that has taken responsibility
// for it, such as one draining the queue. Removing a VM the queue is not
// holding does nothing.
func (q *CompletionQueue) Remove(poolName, vmID string) {
	q.mu.Lock()
	_, held := q.vmIDs[vmID]
	delete(q.vmIDs, vmID)
	if idx := slices.Index(q.vmIDsByPool[poolName], vmID); idx >= 0 {
		q.vmIDsByPool[poolName] = slices.Delete(q.vmIDsByPool[poolName], idx, idx+1)
	}
	q.mu.Unlock()
	if held {
		q.pubsub.Publish(struct{}{})
	}
}

// Subscribe returns a channel that is signalled whenever the set of VMs in
// the queue changes, and a function that must be called to release the
// subscription.
func (q *CompletionQueue) Subscribe(ctx context.Context) (<-chan struct{}, func()) {
	sub := q.pubsub.Subscribe(ctx, 1)
	return sub.C(), func() { q.pubsub.Unsubscribe(sub) }
}

// Close drains every outcome's queue and deletes the VMs they were holding. It
// is idempotent, since a queue shared by several pools is closed by each of
// them: a later call returns the result of the first.
func (q *CompletionQueue) Close(ctx context.Context) error {
	q.closeOnce.Do(func() {
		// Shut out any further pushes, and wait for those under way, before
		// closing the queues' input channels: a send racing that close panics.
		q.pushMu.Lock()
		q.closed = true
		q.pushMu.Unlock()

		// The queues are taken under the lock but drained without it: draining
		// releases each VM, which unrecords it, which needs the lock itself.
		q.mu.Lock()
		names := slices.Sorted(maps.Keys(q.outcomeQueues))
		queues := make([]*outcomeQueue, 0, len(names))
		for _, n := range names {
			queues = append(queues, q.outcomeQueues[n])
		}
		q.mu.Unlock()
		var errs errors.M
		for i, cq := range queues {
			errs.Append(q.closeCQ(ctx, fmt.Sprintf("releasing vm from outcome queue %q", names[i]), cq.cq))
		}
		q.wg.Wait(ctx)
		q.closeErr = errs.Err()
	})
	return q.closeErr
}

// drainTimeout bounds how long closeCQ waits for a live queue to deliver its
// next event before giving up on it.
const drainTimeout = 5 * time.Second

// releaseVM deletes the VM an event was holding, if any.
func (q *CompletionQueue) releaseVM(ctx context.Context, msg string, e CompletionEvent, errs *errors.M) {
	q.unrecordVM(e)
	vm := eventVM(e)
	if vm == nil {
		return
	}
	ctxlog.Info(ctx, msg, "vm", vm.ID())
	errs.Append(vm.Delete(ctx))
}

// drainNonBlocking drains whatever events are already buffered in the FIFO's
// output channel without waiting for further delivery.
func (q *CompletionQueue) drainNonBlocking(ctx context.Context, msg string, cq *patterns.FIFO[CompletionEvent], errs *errors.M) error {
	for {
		select {
		case e, ok := <-cq.Out():
			if !ok {
				cq.Stop(ctx)
				return errs.Err()
			}
			q.releaseVM(ctx, msg, e, errs)
		default:
			cq.Stop(ctx)
			return errs.Err()
		}
	}
}

func (q *CompletionQueue) closeCQ(ctx context.Context, msg string, cq *patterns.FIFO[CompletionEvent]) error {
	var errs errors.M
	close(cq.In())
	if q.ctx.Err() != nil {
		// The queue's context has been cancelled, as it is on interrupt, so the
		// FIFO's goroutine has already returned without closing Out. Only what
		// it delivered before exiting can be drained; a blocking receive would
		// never return, which is what used to hang shutdown.
		return q.drainNonBlocking(ctx, msg, cq, &errs)
	}
	// The queue is live, so closing In makes the FIFO flush its buffer and then
	// close Out. Wait for that, bounded so that a wedged FIFO cannot stall
	// shutdown either.
	timer := time.NewTimer(drainTimeout)
	defer timer.Stop()
	for {
		select {
		case e, ok := <-cq.Out():
			if !ok {
				cq.Stop(ctx)
				return errs.Err()
			}
			q.releaseVM(ctx, msg, e, &errs)
			// Deleting a VM takes time, so the timeout bounds the wait for each
			// event rather than the drain as a whole.
			// Safely drain timer channel if it already fired to avoid blocking on reset.
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(drainTimeout)
		case <-q.ctx.Done():
			// The queue's context was cancelled mid-drain; drain whatever was
			// delivered before the FIFO exited and return.
			return q.drainNonBlocking(ctx, msg, cq, &errs)
		case <-ctx.Done():
			cq.Stop(ctx)
			return errs.Err()
		case <-timer.C:
			ctxlog.Warn(ctx, "gave up draining the completion queue", "queue", msg, "timeout", drainTimeout)
			cq.Stop(ctx)
			return errs.Err()
		}
	}
}
