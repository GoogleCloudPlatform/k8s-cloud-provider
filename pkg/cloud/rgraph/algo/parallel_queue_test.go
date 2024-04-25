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
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"k8s.io/klog/v2"
)

func newTaskControl(q *ParallelQueue[*task]) *taskControl {
	return &taskControl{
		q:       q,
		tasks:   map[string]*task{},
		signals: map[string]chan struct{}{},
	}
}

type taskControl struct {
	q *ParallelQueue[*task]

	lock    sync.Mutex
	tasks   map[string]*task
	signals map[string]chan struct{}
}

func (c *taskControl) newTask(id string, steps []step) *task {
	t := &task{c: c, id: id, steps: steps}
	c.tasks[id] = t
	for _, s := range steps {
		switch {
		case s.signal != "" && c.signals[s.signal] == nil:
			c.signals[s.signal] = make(chan struct{})
		case s.wait != "" && c.signals[s.wait] == nil:
			c.signals[s.wait] = make(chan struct{})
		}
	}
	return t
}

type verifyOptions struct {
	orphans map[string]bool
}

func (c *taskControl) verify(t *testing.T, tracer *taskTracer, opts *verifyOptions) {
	t.Helper()

	tracer.lock.Lock()
	defer tracer.lock.Unlock()

	// All tasks should have been executed.
	tasksGot := map[string]struct{}{}
	tasksWant := map[string]struct{}{}
	for _, ri := range tracer.got {
		tasksGot[ri.ID] = struct{}{}
	}
	for id := range c.tasks {
		if opts != nil && opts.orphans != nil && opts.orphans[id] {
			// task was expected to be leaked and never complete.
			continue
		}
		tasksWant[id] = struct{}{}
	}
	if diff := cmp.Diff(tasksGot, tasksWant); diff != "" {
		t.Fatalf("Diff(tasksGot, tasksWant) -got+want: %s", diff)
	}
}

func (c *taskControl) stepAdd(id string) {
	c.lock.Lock()
	t, ok := c.tasks[id]
	if !ok {
		// Implicitly create tasks that are referenced but not have newTask to
		// reduce test verbosity.
		t = c.newTask(id, nil)
	}
	c.lock.Unlock()
	klog.Infof("Add %q", id)
	c.q.Add(t)
}

func (c *taskControl) stepWait(id string) {
	c.lock.Lock()
	ch := c.signals[id]
	c.lock.Unlock()
	klog.Infof("Wait %q", id)
	<-ch
}

func (c *taskControl) stepSignal(id string) {
	c.lock.Lock()
	ch := c.signals[id]
	c.lock.Unlock()
	klog.Infof("Signal %q", id)
	ch <- struct{}{}
}

func (c *taskControl) queueOp(_ context.Context, t *task) error {
	return t.run()
}

type task struct {
	c     *taskControl
	id    string
	steps []step
}

type step struct {
	// add if non-empty will launch a new task with the given id.
	add string
	// wait if non-empty will wait for signal with the given id.
	wait string
	// signal if non-empty will signal the given id.
	signal string
	// sleep if non-zero will time.Sleep for the duration.
	sleep time.Duration
	// f will be executed if non-nil.
	f func()
	// err will cause the task to return an error if non-nil.
	err error
}

func (t *task) String() string { return t.id }

func (t *task) run() error {
	for _, step := range t.steps {
		switch {
		case step.add != "":
			t.c.stepAdd(step.add)
		case step.wait != "":
			t.c.stepWait(step.wait)
		case step.signal != "":
			t.c.stepSignal(step.signal)
		case step.sleep != 0:
			time.Sleep(step.sleep)
		case step.f != nil:
			step.f()
		default:
			return step.err
		}
	}
	return nil
}

type taskTracer struct {
	lock sync.Mutex
	got  []RunInfo
}

func (c *taskTracer) do(ri RunInfo) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.got = append(c.got, ri)
}

func TestParallelQueue(t *testing.T) {
	for _, tc := range []struct {
		name          string
		setup         func(context.Context, *ParallelQueue[*task], *taskControl) context.Context
		verifyOptions *verifyOptions
		wantErr       bool

		waitForOrphans        bool
		wantWaitForOrphansErr bool
	}{
		{
			name:           "empty queue",
			setup:          func(ctx context.Context, _ *ParallelQueue[*task], _ *taskControl) context.Context { return ctx },
			waitForOrphans: true,
		},
		{
			name: "one task",
			setup: func(ctx context.Context, q *ParallelQueue[*task], c *taskControl) context.Context {
				q.Add(c.newTask("a", nil))
				return ctx
			},
			waitForOrphans: true,
		},
		{
			name: "ten tasks",
			setup: func(ctx context.Context, q *ParallelQueue[*task], c *taskControl) context.Context {
				for i := 0; i < 10; i++ {
					q.Add(c.newTask(fmt.Sprintf("task_%d", i), nil))
				}
				return ctx
			},
			waitForOrphans: true,
		},
		{
			name: "Add called from task",
			setup: func(ctx context.Context, q *ParallelQueue[*task], c *taskControl) context.Context {
				q.Add(c.newTask("a", []step{
					{add: "b"},
				}))
				return ctx
			},
			waitForOrphans: true,
		},
		{
			name: "Add with slow task",
			setup: func(ctx context.Context, q *ParallelQueue[*task], c *taskControl) context.Context {
				q.Add(c.newTask("a", []step{
					{sleep: 100 * time.Millisecond},
					{add: "b"},
				}))
				return ctx
			},
			waitForOrphans: true,
		},
		{
			name: "multiple Add called from tasks",
			setup: func(ctx context.Context, q *ParallelQueue[*task], c *taskControl) context.Context {
				q.Add(c.newTask("a", []step{
					{add: "b0"},
					{add: "c0"},
					{add: "d0"},
				}))
				c.newTask("b0", []step{
					{add: "b1"},
					{add: "b2"},
					{add: "b3"},
				})
				c.newTask("c0", []step{
					{add: "c1"},
					{add: "c2"},
				})
				c.newTask("d0", []step{
					{add: "d1"},
					{add: "d2"},
					{add: "d3"},
				})
				return ctx
			},
			waitForOrphans: true,
		},
		{
			name: "a spawns b; b must execute before a completes",
			setup: func(ctx context.Context, q *ParallelQueue[*task], c *taskControl) context.Context {
				q.Add(c.newTask("a", []step{
					{add: "b"},
					{wait: "done"},
				}))
				c.newTask("b", []step{
					{sleep: 10 * time.Millisecond},
					{signal: "done"},
				})
				return ctx
			},
			waitForOrphans: true,
		},
		{
			name: "run exits with context cancel",
			setup: func(ctx context.Context, q *ParallelQueue[*task], c *taskControl) context.Context {
				ctx, cancel := context.WithCancel(ctx)
				q.Add(c.newTask("a", []step{
					{f: func() { cancel() }},
					// This will orphan a goroutine as the "forever" will never be signaled.
					{wait: "forever"},
				}))
				return ctx
			},
			verifyOptions: &verifyOptions{
				orphans: map[string]bool{"a": true},
			},
			wantErr: true,
		},
		{
			name: "run exits with task error",
			setup: func(ctx context.Context, q *ParallelQueue[*task], c *taskControl) context.Context {
				q.Add(c.newTask("a", []step{
					{err: errors.New("injected")},
				}))
				return ctx
			},
			wantErr:        true,
			waitForOrphans: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tracer := &taskTracer{}
			q := NewParallelQueue[*task](
				WorkerCount(2),
				UseTracer(tracer.do),
			)

			taskc := newTaskControl(q)
			ctx := tc.setup(context.Background(), q, taskc)

			err := q.Run(ctx, taskc.queueOp)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("q.Run() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}

			taskc.verify(t, tracer, tc.verifyOptions)

			if tc.waitForOrphans {
				err := q.WaitForOrphans(context.Background())
				if gotErr := err != nil; gotErr != tc.wantWaitForOrphansErr {
					t.Fatalf("q.WaitForOrphans() = %v; gotErr = %t, want %t", err, gotErr, tc.wantWaitForOrphansErr)
				}
			}
		})
	}
}

func TestParallelQueueErr(t *testing.T) {
	tracer := &taskTracer{}
	q := NewParallelQueue[*task](
		WorkerCount(2),
		UseTracer(tracer.do),
	)
	taskc := newTaskControl(q)
	err := q.Run(context.Background(), taskc.queueOp)
	if err != nil {
		t.Fatalf("q.Run() = %v; want nil", err)
	}
	// Cannot be run twice.
	err = q.Run(context.Background(), taskc.queueOp)
	if err == nil {
		t.Fatalf("q.Run() = %v; want not nil", err)
	}
}

func TestParallelQueueWaitForOrphans(t *testing.T) {
	tracer := &taskTracer{}
	q := NewParallelQueue[*task](
		WorkerCount(2),
		UseTracer(tracer.do),
	)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	taskc := newTaskControl(q)
	ok := q.Add(taskc.newTask("a", []step{
		{f: func() { cancel() }},
		{wait: "done"},
		{sleep: 10 * time.Millisecond},
	}))
	if !ok {
		t.Fatalf("q.Add(_) = %v, want true", ok)
	}
	q.Run(ctx, taskc.queueOp) // ignore err
	// Unblock the task.
	taskc.stepSignal("done")
	// WaitForOrphans should return when the task is done.
	err := q.WaitForOrphans(context.Background())
	if err != nil {
		t.Fatalf("q.Run() = %v; want nil", err)
	}
	// Check that queue will return false when queue state is done.
	ok = q.Add(taskc.newTask("a", nil))
	if ok {
		t.Fatalf("q.Add(_) = %v, want false", ok)
	}
}
