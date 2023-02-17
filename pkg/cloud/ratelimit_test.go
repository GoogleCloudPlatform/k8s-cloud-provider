/*
Copyright 2018 Google LLC

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

package cloud

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	testDelay = 5 * time.Second
)

func verifyError(t *testing.T, desc string, err, expectErr error) {
	if err != expectErr {
		t.Errorf("Expect error for %v to be %v, but got %v", desc, expectErr, err)
	}
}

func verifyDelay(t *testing.T, delay, expectDelay time.Duration) {
	if delay != expectDelay {
		t.Errorf("Expect delay for strategy to be %v, but got %v", expectDelay, delay)
	}
}

func verifyBlocked(t *testing.T, blocked bool) {
	if !blocked {
		t.Errorf("StrategyRateLimiter.Accept() wasn't blocked, but was expected to")
	}
}

type FakeAcceptor struct{ accept func() }

func (f *FakeAcceptor) Accept() {
	f.accept()
}

type delayRequestTracker struct {
	lock           sync.Mutex
	delayRequested bool
}

func (t *delayRequestTracker) isDelayRequested() bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.delayRequested
}

func (t *delayRequestTracker) setDelayRequested(delayRequested bool) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.delayRequested = delayRequested
}

type fakeThrottlingStrategy struct {
	lock         sync.Mutex
	currentDelay time.Duration
	*delayRequestTracker
}

func (strategy *fakeThrottlingStrategy) Delay() time.Duration {
	strategy.lock.Lock()
	defer strategy.lock.Unlock()
	defer strategy.setDelayRequested(true)
	return strategy.currentDelay
}

func (strategy *fakeThrottlingStrategy) Observe(err error) {
	if err != nil {
		strategy.setDelay(testDelay)
	} else {
		strategy.setDelay(0)
	}
}

func (strategy *fakeThrottlingStrategy) setDelay(delay time.Duration) {
	strategy.lock.Lock()
	defer strategy.lock.Unlock()
	strategy.currentDelay = delay
}

func newFakeThrottlingStrategy() *fakeThrottlingStrategy {
	return &fakeThrottlingStrategy{
		currentDelay:        0,
		delayRequestTracker: &delayRequestTracker{delayRequested: false},
	}
}

func TestAcceptRateLimiter(t *testing.T) {
	t.Parallel()

	fa := &FakeAcceptor{accept: func() {}}
	arl := &AcceptRateLimiter{fa}
	err := arl.Accept(context.Background(), nil)
	verifyError(t, "AcceptRateLimiter.Accept()", err, nil)

	// Use context that has been cancelled and expect a context error returned.
	ctxCancelled, cancelled := context.WithCancel(context.Background())
	cancelled()
	// Verify context is cancelled by now.
	<-ctxCancelled.Done()

	fa.accept = func() { time.Sleep(1 * time.Second) }
	err = arl.Accept(ctxCancelled, nil)
	verifyError(t, "AcceptRateLimiter.Accept()", err, ctxCancelled.Err())
}

func TestMinimumRateLimiter(t *testing.T) {
	t.Parallel()

	fa := &FakeAcceptor{accept: func() {}}
	arl := &AcceptRateLimiter{fa}
	var called bool
	fa.accept = func() { called = true }
	m := &MinimumRateLimiter{RateLimiter: arl, Minimum: 10 * time.Millisecond}

	err := m.Accept(context.Background(), nil)
	verifyError(t, "MinimumRateLimiter.Accept()", err, nil)
	if !called {
		t.Errorf("`called` = false, want true")
	}

	// Use context that has been cancelled and expect a context error returned.
	ctxCancelled, cancelled := context.WithCancel(context.Background())
	cancelled()
	// Verify context is cancelled by now.
	<-ctxCancelled.Done()
	called = false
	err = m.Accept(ctxCancelled, nil)
	verifyError(t, "MinimumRateLimiter.Accept()", err, ctxCancelled.Err())
	if called {
		t.Errorf("`called` = true, want false")
	}
}

func TestStrategyRateLimiter(t *testing.T) {
	t.Parallel()

	fakeClock := clock.NewFakeClock(time.Now())
	strategy := newFakeThrottlingStrategy()
	rl := &StrategyRateLimiter{
		strategy: strategy,
		clock:    fakeClock,
	}
	acceptAndVerify := func() {
		go func() {
			time.Sleep(2 * time.Second)
			fakeClock.Step(0)
		}()
		err := rl.Accept(context.Background(), nil)
		verifyError(t, "StrategyRateLimiter.Accept()", err, nil)
	}

	// Use context that has been cancelled and expect a context error returned.
	ctxCancelled, cancelled := context.WithCancel(context.Background())
	cancelled()
	// Verify context is cancelled by now.
	<-ctxCancelled.Done()
	err := rl.Accept(ctxCancelled, nil)
	verifyError(t, "StrategyRateLimiter.Accept()", err, ctxCancelled.Err())

	acceptAndVerify()
	rl.Observe(context.Background(), fmt.Errorf("test error"), nil)
	verifyDelay(t, strategy.Delay(), testDelay)
	strategy.setDelayRequested(false)

	wg := sync.WaitGroup{}
	wg.Add(1)
	blocked := false
	go func() {
		err := wait.PollImmediate(time.Second, 5*time.Second, func() (bool, error) { return strategy.isDelayRequested(), nil })
		verifyError(t, "wait.PollImmediate()", err, nil)
		time.Sleep(4 * time.Second)
		verifyBlocked(t, blocked)
		fakeClock.Step(testDelay)
		wg.Done()
	}()
	blocked = true
	acceptAndVerify()
	blocked = false
	// Needed if the call to Accept() wasn't blocked, otherwise test would
	// finish without the check for blocked request in the goroutine
	wg.Wait()
}

func TestStrategyRateLimiterGroupBlock(t *testing.T) {
	t.Parallel()

	fakeClock := clock.NewFakeClock(time.Now())
	strategy := newFakeThrottlingStrategy()
	rl := &StrategyRateLimiter{
		strategy: strategy,
		clock:    fakeClock,
	}
	acceptAndVerify := func() {
		go func() {
			time.Sleep(2 * time.Second)
			fakeClock.Step(0)
		}()
		err := rl.Accept(context.Background(), nil)
		verifyError(t, "StrategyRateLimiter.Accept()", err, nil)
	}

	// First request, not blocked
	acceptAndVerify()

	strategy.setDelay(testDelay)
	strategy.setDelayRequested(false)
	finished := 0

	// Second request, should block subsequent requests until delay has passed
	go func() {
		err := rl.Accept(context.Background(), nil)
		finished++
		verifyError(t, "StrategyRateLimiter.Accept()", err, nil)
	}()
	err := wait.PollImmediate(time.Second, 5*time.Second, func() (bool, error) { return strategy.isDelayRequested(), nil })
	verifyError(t, "wait.PollImmediate()", err, nil)
	time.Sleep(2 * time.Second)
	verifyBlocked(t, finished == 0)
	strategy.setDelay(0)

	// Third request
	go func() {
		err = rl.Accept(context.Background(), nil)
		finished++
		verifyError(t, "StrategyRateLimiter.Accept()", err, nil)
	}()
	if finished != 0 {
		t.Errorf("Some calls to StrategyRateLimiter.Accept() were not blocked, but were expected to")
	}

	// Unblock remaining requests but going forward in time
	fakeClock.Step(testDelay)
	err = wait.PollImmediate(time.Second, 10*time.Second, func() (bool, error) {
		fakeClock.Step(0)
		return finished == 2, nil
	})
	verifyError(t, "wait.PollImmediate() for finished requests", err, nil)
	if finished != 2 {
		t.Errorf("Expected 2 finished requests, but got %v", finished)
	}
}
