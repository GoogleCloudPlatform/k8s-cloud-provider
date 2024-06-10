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
	"testing"
	"time"
)

type FakeAcceptor struct{ accept func() }

func (f *FakeAcceptor) Accept() {
	f.accept()
}

func TestAcceptRateLimiter(t *testing.T) {
	t.Parallel()

	fa := &FakeAcceptor{accept: func() {}}
	arl := &AcceptRateLimiter{fa}
	err := arl.Accept(context.Background(), nil)
	if err != nil {
		t.Errorf("AcceptRateLimiter.Accept() = %v, want nil", err)
	}

	// Use context that has been cancelled and expect a context error returned.
	ctxCancelled, cancelled := context.WithCancel(context.Background())
	cancelled()
	// Verify context is cancelled by now.
	<-ctxCancelled.Done()

	fa.accept = func() { time.Sleep(1 * time.Second) }
	err = arl.Accept(ctxCancelled, nil)
	if err != ctxCancelled.Err() {
		t.Errorf("AcceptRateLimiter.Accept() = %v, want %v", err, ctxCancelled.Err())
	}
}

func TestMinimumRateLimiter(t *testing.T) {
	t.Parallel()

	fa := &FakeAcceptor{accept: func() {}}
	arl := &AcceptRateLimiter{fa}
	var called bool
	fa.accept = func() { called = true }
	m := &MinimumRateLimiter{RateLimiter: arl, Minimum: 10 * time.Millisecond}

	err := m.Accept(context.Background(), nil)
	if err != nil {
		t.Errorf("MinimumRateLimiter.Accept = %v, want nil", err)
	}
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
	if err != ctxCancelled.Err() {
		t.Errorf("AcceptRateLimiter.Accept() = %v, want %v", err, ctxCancelled.Err())
	}
	if called {
		t.Errorf("`called` = true, want false")
	}
}

func TestTickerRateLimiter(t *testing.T) {
	t.Parallel()

	trl := NewTickerRateLimiter(100, time.Second)
	err := trl.Accept(context.Background(), nil)
	if err != nil {
		t.Errorf("TickerRateLimiter.Accept = %v, want nil", err)
	}

	// Use context that has been cancelled and expect a context error returned.
	ctxCancelled, cancelled := context.WithCancel(context.Background())
	cancelled()
	// Verify context is cancelled by now.
	<-ctxCancelled.Done()
	err = trl.Accept(ctxCancelled, nil)
	if err != ctxCancelled.Err() {
		t.Errorf("TickerRateLimiter.Accept() = %v, want %v", err, ctxCancelled.Err())
	}
}

func TestCompositeRateLimiter(t *testing.T) {
	t.Parallel()

	var calledA bool
	fa := &FakeAcceptor{accept: func() { calledA = true }}
	arl := &AcceptRateLimiter{fa}
	rl := NewCompositeRateLimiter(arl)

	// Call default.
	err := rl.Accept(context.Background(), nil)
	if err != nil {
		t.Errorf("CompositeRateLimiter.Accept = %v, want nil", err)
	}
	if !calledA {
		t.Errorf("`calledA` = false, want true")
	}

	calledA = false
	calledB := false
	fb := &FakeAcceptor{accept: func() { calledB = true }}
	brl := &AcceptRateLimiter{fb}
	rl.Register("Meshes", "", brl)

	// Call registered rate limiter.
	err = rl.Accept(context.Background(), &CallContextKey{Service: "Meshes"})
	if err != nil {
		t.Errorf("CompositeRateLimiter.Accept = %v, want nil", err)
	}
	if !calledB {
		t.Errorf("`calledB` = false, want true")
	}
	if calledA {
		t.Errorf("`calledA` = true, want false")
	}

	calledB = false
	// Call default rate limiter when registered is not found
	err = rl.Accept(context.Background(), &CallContextKey{Service: "service-does-not-exist"})
	if err != nil {
		t.Errorf("CompositeRateLimiter.Accept = %v, want nil", err)
	}
	if !calledA {
		t.Errorf("`calledA` = false, want true")
	}
	if calledB {
		t.Errorf("`calledB` = true, want false")
	}

	calledA = false
	calledC := false
	fc := &FakeAcceptor{accept: func() { calledC = true }}
	crl := &AcceptRateLimiter{fc}
	rl.Register("", "Get", crl)

	// Call rate limiter for network service when no project was specified
	err = rl.Accept(context.Background(), &CallContextKey{ProjectID: "project-does-not-exist", Service: "Networks", Operation: "Get"})
	if err != nil {
		t.Errorf("CompositeRateLimiter.Accept = %v, want nil", err)
	}
	if !calledC {
		t.Errorf("`calledC` = false, want true")
	}
	if calledA {
		t.Errorf("`calledA` = true, want false")
	}
	if calledB {
		t.Errorf("`calledB` = true, want false")
	}
}

type CountingRateLimiter int

func (crl *CountingRateLimiter) Accept(_ context.Context, key *CallContextKey) error {
	*crl++
	return nil
}

func (*CountingRateLimiter) Observe(context.Context, error, *RateLimitKey) {}

func TestCompositeRateLimiter_Table(t *testing.T) {
	t.Parallel()

	def := new(CountingRateLimiter)
	rl := NewCompositeRateLimiter(def)
	defNetRL := new(CountingRateLimiter)
	rl.Register("networks", "", defNetRL)
	getNetRL := new(CountingRateLimiter)
	rl.Register("networks", "get", getNetRL)

	for _, project := range []string{"", "projectB", "project-does-not-exist"} {
		for _, service := range []string{"", "networks", "service-does-not-exist"} {
			for _, operation := range []string{"", "get", "operation-does-not-exist"} {
				key := &CallContextKey{
					ProjectID: project,
					Service:   service,
					Operation: operation,
				}
				err := rl.Accept(context.Background(), key)
				if err != nil {
					t.Errorf("CompositeRateLimiter.Accept = %v, want nil", err)
				}
			}
		}
	}

	if *def != 18 {
		t.Errorf("def served %d calls, want = 18", *def)
	}
	if *defNetRL != 6 {
		t.Errorf("defNetRL served %d calls, want = 6", *defNetRL)
	}
	if *getNetRL != 3 {
		t.Errorf("getNetRL served %d calls, want = 3", *getNetRL)
	}
}
