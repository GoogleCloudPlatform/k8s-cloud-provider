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
	"sync"
	"time"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"k8s.io/apimachinery/pkg/util/clock"
)

// RateLimitKey is a key identifying the operation to be rate limited. The rate limit
// queue will be determined based on the contents of RateKey.
type RateLimitKey struct {
	// ProjectID is the non-numeric ID of the project.
	ProjectID string
	// Operation is the specific method being invoked (e.g. "Get", "List").
	Operation string
	// Version is the API version of the call.
	Version meta.Version
	// Service is the service being invoked (e.g. "Firewalls", "BackendServices")
	Service string
}

// RateLimiter is the interface for a rate limiting policy.
type RateLimiter interface {
	// Accept uses the RateLimitKey to derive a sleep time for the calling
	// goroutine. This call will block until the operation is ready for
	// execution.
	//
	// Accept returns an error if the given context ctx was canceled
	// while waiting for acceptance into the queue.
	Accept(ctx context.Context, key *RateLimitKey) error
	// Observe uses the RateLimitKey to handle response results, which may affect
	// the sleep time for the Accept function.
	Observe(ctx context.Context, err error, key *RateLimitKey)
}

// acceptor is an object which blocks within Accept until a call is allowed to run.
// Accept is a behavior of the flowcontrol.RateLimiter interface.
type acceptor interface {
	// Accept blocks until a call is allowed to run.
	Accept()
}

// AcceptRateLimiter wraps an Acceptor with RateLimiter parameters.
type AcceptRateLimiter struct {
	// Acceptor is the underlying rate limiter.
	Acceptor acceptor
}

// Accept wraps an Acceptor and blocks on Accept or context.Done(). Key is ignored.
func (rl *AcceptRateLimiter) Accept(ctx context.Context, _ *RateLimitKey) error {
	ch := make(chan struct{})
	go func() {
		rl.Acceptor.Accept()
		close(ch)
	}()
	select {
	case <-ch:
		break
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// Observe does nothing.
func (rl *AcceptRateLimiter) Observe(context.Context, error, *RateLimitKey) {
}

// ThrottlingStrategy handles delays based on feedbacks provided.
type ThrottlingStrategy interface {
	// Delay returns the delay for next request.
	Delay() time.Duration
	// Observe recalculates the next delay based on the feedback.
	Observe(err error)
}

// StrategyRateLimiter wraps a ThrottlingStrategy with RateLimiter parameters.
type StrategyRateLimiter struct {
	lock sync.Mutex
	// strategy is the underlying throttling strategy.
	strategy ThrottlingStrategy
	
	clock clock.Clock
}

// NewStrategyRateLimiter returns a StrategyRateLimiter backed by the provided ThrottlingStrategy.
func NewStrategyRateLimiter(strategy ThrottlingStrategy) *StrategyRateLimiter {
	return &StrategyRateLimiter{
		strategy: strategy,
		clock:    clock.RealClock{},
	}
}

// Accept block for the delay provided by the strategy or until context.Done(). Key is ignored.
func (rl *StrategyRateLimiter) Accept(ctx context.Context, _ *RateLimitKey) error {
	rl.lock.Lock()
	defer rl.lock.Unlock()
	select {
	case <-rl.clock.After(rl.strategy.Delay()):
		break
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

// Observe pushes error further to the strategy. Key is ignored.
func (rl *StrategyRateLimiter) Observe(_ context.Context, err error, _ *RateLimitKey) {
	rl.strategy.Observe(err)
}

// NopRateLimiter is a rate limiter that performs no rate limiting.
type NopRateLimiter struct {
}

// Accept everything immediately.
func (*NopRateLimiter) Accept(context.Context, *RateLimitKey) error {
	return nil
}

// Observe does nothing.
func (*NopRateLimiter) Observe(context.Context, error, *RateLimitKey) {
}

// MinimumRateLimiter wraps a RateLimiter and will only call its Accept until the minimum
// duration has been met or the context is cancelled.
type MinimumRateLimiter struct {
	// RateLimiter is the underlying ratelimiter which is called after the mininum time is reacehd.
	RateLimiter RateLimiter
	// Minimum is the minimum wait time before the underlying ratelimiter is called.
	Minimum time.Duration
}

// Accept blocks on the minimum duration and context. Once the minimum duration is met,
// the func is blocked on the underlying ratelimiter.
func (m *MinimumRateLimiter) Accept(ctx context.Context, key *RateLimitKey) error {
	select {
	case <-time.After(m.Minimum):
		return m.RateLimiter.Accept(ctx, key)
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Observe just passes error to the underlying ratelimiter.
func (m *MinimumRateLimiter) Observe(ctx context.Context, err error, key *RateLimitKey) {
	m.RateLimiter.Observe(ctx, err, key)
}
