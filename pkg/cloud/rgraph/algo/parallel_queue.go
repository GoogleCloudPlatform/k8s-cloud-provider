/*
Copyright 2023 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package algo

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

type config struct {
	workerCount int
	tracer      Tracer
}

type QueueOption func(*config)

func WorkerCount(n int) QueueOption  { return func(c *config) { c.workerCount = n } }
func UseTracer(t Tracer) QueueOption { return func(c *config) { c.tracer = t } }

type Tracer func(RunInfo)

// RunInfo records the details of a task.
type RunInfo struct {
	// ID for the execution.
	ID string
	// Queued is the timestamp of when the task was queued.
	Queued time.Time
	// Start is the timestamp of when the task was started as a goroutine.
	Start time.Time
	// End is when the task was finished.
	End time.Time
	// Err is the result of the task.
	Err error
}

// NewParallelQueue returns a new queue instance.
func NewParallelQueue[T fmt.Stringer](opts ...QueueOption) *ParallelQueue[T] {
	cfg := config{
		workerCount: 2,
		tracer:      func(RunInfo) {},
	}
	for _, o := range opts {
		o(&cfg)
	}
	return &ParallelQueue[T]{
		c:    cfg,
		in:   make(chan struct{}, 100),
		done: make(chan RunInfo, 100),
	}
}

// ParallelQueue executes parallel tasks on separate goroutines.
type ParallelQueue[T fmt.Stringer] struct {
	c config

	lock    sync.Mutex
	state   queueState
	pending []queueElement[T]
	in      chan struct{}
	done    chan RunInfo
	active  int
}

type queueElement[T fmt.Stringer] struct {
	ri   RunInfo
	item T
}

type queueState int

const (
	stateNotStarted queueState = iota
	stateRunning
	stateDone
)

// Add an item to the queue. This method is threadsafe within op() and
// can be called during Run(). It is NOT safe to call Add() from
// a different, unassociated thread.
func (q *ParallelQueue[T]) Add(item T) {
	q.lock.Lock()
	defer q.lock.Unlock()

	qe := queueElement[T]{
		ri: RunInfo{
			ID:     item.String(),
			Queued: time.Now(),
		},
		item: item,
	}
	q.pending = append(q.pending, qe)
	if q.state == stateRunning {
		// Add() will always result in a call to launch() as
		// we enqueue one element for each call to Add().
		// This means the Run() loop will wake up AT LEAST
		// once per `item`.
		//
		// `item` will in q.pending during launch() because
		// the <-q.in will happen AFTER append(q.pending).
		q.in <- struct{}{}
	}
}

// Run the queue using op() to process each task. Different op()s must
// not have interdependencies (e.g. op1 is blocked on op2 completing),
// as this can result in a deadlock.
//
// Queue execution will stop if op() returns an error or the context
// is canceled. When an error occurs, there may be goroutines that are
// continuing to execute. Use q.WaitForOrphans() to wait for them
// remaining goroutines.
func (q *ParallelQueue[T]) Run(ctx context.Context, op func(context.Context, T) error) error {
	q.lock.Lock()
	if q.state != stateNotStarted {
		return fmt.Errorf("Run() can only be called once (state=%d)", q.state)
	}
	q.state = stateRunning
	q.lock.Unlock()

	for {
		q.lock.Lock()

		klog.V(4).Infof("Run loop: pending: %d active: %d", len(q.pending), q.active)
		if len(q.pending) == 0 && q.active == 0 {
			q.state = stateDone
			q.lock.Unlock()
			return nil
		}
		q.launch(ctx, op)

		q.lock.Unlock()

		select {
		case <-ctx.Done():
			q.lock.Lock()
			q.state = stateDone
			klog.V(2).Infof("Context is Done, exiting early (pending: %d active: %d): %v", len(q.pending), q.active, ctx.Err())
			q.lock.Unlock()
			return ctx.Err()
		case ri := <-q.done:
			q.c.tracer(ri)
			q.lock.Lock()
			q.active--

			if ri.Err != nil {
				q.state = stateDone
				klog.V(2).Infof("Task error, exiting early (pending: %d active: %d): %v", len(q.pending), q.active, ri.Err)
				q.lock.Unlock()
				return ri.Err
			}

			q.lock.Unlock()
		case <-q.in:
			klog.V(4).Info("<-q.in")
			// Fall through to launch().
		}
	}
}

// launch task(s) if possible.
//
// Precondition: q.lock must be locked.
func (q *ParallelQueue[T]) launch(ctx context.Context, op func(context.Context, T) error) {
	pop := func() queueElement[T] {
		t := q.pending[0]
		q.pending = q.pending[1:]
		return t
	}

	klog.V(4).Infof("Launch: active: %d/%d pending: %d", q.active, q.c.workerCount, len(q.pending))

	for q.active < q.c.workerCount && len(q.pending) > 0 {
		elt := pop()
		ri := elt.ri
		q.active++
		klog.V(4).Infof("Launch: %q active: %d/%d pending: %d", elt.item, q.active, q.c.workerCount, len(q.pending))
		go func() {
			klog.V(4).Infof("Task %q start", elt.item)
			ri.Start = time.Now()
			ri.Err = op(ctx, elt.item)
			ri.End = time.Now()
			klog.V(4).Infof("Task %q end", elt.item)
			q.done <- ri
		}()
	}
}

// WaitForOrphans will block until remaining op() goroutines
// finish. Call this if Run() returns an error and you need to know
// that all remaining threads of execution are done.
//
// WaitForOrphans will exit early if ctx is cancelled. This `ctx`
// should be different from the `ctx` given to Run().
func (q *ParallelQueue[T]) WaitForOrphans(ctx context.Context) error {
	q.lock.Lock()
	if q.state != stateDone {
		return fmt.Errorf("WaitForOrphans called when not done (state = %d)", q.state)
	}
	q.lock.Unlock()

	for {
		q.lock.Lock()
		klog.V(4).Infof("WaitForOrphans: active: %d", q.active)
		if q.active == 0 {
			q.lock.Unlock()
			klog.V(4).Info("WaitForOrphans: done")
			return nil
		}
		q.lock.Unlock()

		// We do not need to pull from q.in as Add() will not
		// enqueue to the channel.
		select {
		case <-ctx.Done():
			klog.V(4).Infof("WaitForOrphans: early exit, context Done: %v", ctx.Err())
			return ctx.Err()
		case <-q.done:
			q.lock.Lock()
			q.active--
			q.lock.Unlock()
		}
	}
}
