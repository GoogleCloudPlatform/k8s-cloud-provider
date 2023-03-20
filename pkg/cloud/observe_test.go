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

package cloud

import (
	"context"
	"errors"
	"testing"
)

type fakeCO struct {
	startCalled bool
	endCalled   bool
	err         error
}

func (f *fakeCO) Start(ctx context.Context, _ *CallContextKey) { f.startCalled = true }
func (f *fakeCO) End(ctx context.Context, _ *CallContextKey, err error) {
	f.endCalled = true
	f.err = err
}

func TestCallObserver(t *testing.T) {
	errInjected := errors.New("injected")

	for _, tc := range []struct {
		name    string
		wantErr error
	}{
		{name: "no error", wantErr: nil},
		{name: "err", wantErr: errInjected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			obs := &fakeCO{}
			ctx = WithCallObserver(ctx, obs)

			callObserverStart(ctx, nil)
			callObserverEnd(ctx, nil, tc.wantErr)

			if !obs.startCalled || !obs.endCalled {
				t.Errorf("startCalled = %t, endCalled = %t; want true, true", obs.startCalled, obs.endCalled)
			}
			if obs.err != tc.wantErr {
				t.Errorf("obs.err = %v, want %v", obs.err, tc.wantErr)
			}
		})
	}
}
