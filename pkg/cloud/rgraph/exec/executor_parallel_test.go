/*
Copyright 2024 Google LLC

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

package exec

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/google/go-cmp/cmp"
)

func TestParallelExecutor(t *testing.T) {
	for _, tc := range []struct {
		name  string
		graph string
		// pending should be sorted alphabetically for comparison.
		pending []string
		wantErr bool
	}{
		{
			name:  "empty graph",
			graph: "",
		},
		{
			name:  "one action",
			graph: "A",
		},
		{
			name:  "action and dependency",
			graph: "A -> B",
		},
		{
			name:  "chain of 3 actions",
			graph: "A -> B -> C",
		},
		{
			name:  "two chains with common root",
			graph: "A -> B -> C; A -> C",
		},
		{
			name:    "two node cycle",
			graph:   "A -> B -> A",
			pending: []string{"A", "B"},
			wantErr: true,
		},
		{
			name:  "lot of children",
			graph: "A -> B; A -> C; A -> D -> B; A -> E -> F; A -> G",
		},
		{
			name:  "complex fan in",
			graph: "A -> Z; B -> Z; C -> D -> B",
		},
		{
			name:    "cycle in larger graph",
			graph:   "A -> B -> C -> D -> C; X -> Y",
			pending: []string{"C", "D"},
			wantErr: true,
		},
		{
			name:    "error in action",
			graph:   "A -> B -> !C -> D",
			pending: []string{"D"},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockCloud := cloud.NewMockGCE(&cloud.SingleProjectRouter{ID: "proj1"})
			actions := actionsFromGraphStr(tc.graph)

			ex, err := NewParallelExecutor(mockCloud,
				actions,
				TimeoutOption(1*time.Minute),
				ErrorStrategyOption(StopOnError))
			if err != nil {
				t.Fatalf("NewParallelExecutor(_, _) = %v, want nil", err)
			}
			result, err := ex.Run(context.Background())
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("ex.Run(_, _) = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
			got := sortedStrings(result.Pending, func(a Action) string { return a.(*testAction).name })
			if diff := cmp.Diff(got, tc.pending); diff != "" {
				t.Errorf("pending: diff -got,+want: %s", diff)
			}
		})
	}
}

func TestParallelExecutorErrorStrategy(t *testing.T) {
	for _, tc := range []struct {
		name  string
		graph string
		// pending should be sorted alphabetically for comparison.
		pending []string
		errs    []string
	}{
		{
			name:    "linear graph",
			graph:   "A -> !B -> C -> D -> E",
			pending: []string{"C", "D", "E"},
			errs:    []string{"B"},
		},
		{
			name:    "branched graph",
			graph:   "A -> !B -> C; A -> D; A -> E; A -> F",
			pending: []string{"C"},
			errs:    []string{"B"},
		},
	} {
		mockCloud := cloud.NewMockGCE(&cloud.SingleProjectRouter{ID: "proj1"})
		actions := actionsFromGraphStr(tc.graph)

		for _, strategy := range []ErrorStrategy{StopOnError, ContinueOnError} {
			name := tc.name + " " + string(strategy)
			t.Run(name, func(t *testing.T) {
				ex, err := NewParallelExecutor(mockCloud,
					actions,
					ErrorStrategyOption(strategy),
				)
				if err != nil {
					t.Fatalf("NewParallelExecutor() = %v, want nil", err)
				}
				result, err := ex.Run(context.Background())
				if err == nil {
					t.Fatalf("Run() = %v; expected error", err)
				}
				gotErrs := sortedStrings(result.Errors, func(a ActionWithErr) string { return a.Action.(*testAction).name })

				if diff := cmp.Diff(gotErrs, tc.errs); diff != "" {
					t.Errorf("errors: diff -got,+want: %s", diff)
				}
				got := sortedStrings(result.Pending, func(a Action) string { return a.(*testAction).name })
				if diff := cmp.Diff(got, tc.pending); diff != "" {
					t.Errorf("pending: diff -got,+want: %s", diff)
				}
			})
		}
	}
}

func TestParallelExecutorTimeoutOptions(t *testing.T) {
	for _, tc := range []struct {
		name string

		timeout               time.Duration
		waitForOrphansTimeout time.Duration
		injectError           bool
		wantErr               bool
		// actions should be sorted alphabetically for comparison.
		completed []string
		pending   []string
		errors    []string
	}{
		{
			name:                  "All actions should finish within timeout",
			timeout:               10 * time.Second,
			waitForOrphansTimeout: 1 * time.Second,
			completed:             []string{"A", "B", "C", "D"},
		},
		{
			name:                  "Active actions should finish in waitForOrphans",
			timeout:               2 * time.Second,
			waitForOrphansTimeout: 5 * time.Second,
			completed:             []string{"A", "B", "C"},
			pending:               []string{"D"},
			wantErr:               true,
		},
		{
			name:                  "Actions did not finish in waitForOrphans",
			timeout:               1 * time.Second,
			waitForOrphansTimeout: 2 * time.Second,
			completed:             []string{"A", "B"},
			pending:               []string{"D"},
			wantErr:               true,
		},
		{
			name:      "Actions should finish when no timeout is set",
			completed: []string{"A", "B", "C", "D"},
		},
		{
			name:      "Orphaned actions should finish when no waitForOrphansTimeout is set",
			timeout:   1 * time.Second,
			completed: []string{"A", "B", "C"},
			pending:   []string{"D"},
			wantErr:   true,
		},
		{
			name:        "Active actions should finish after error without timeout",
			injectError: true,
			completed:   []string{"A", "C"},
			errors:      []string{"B"},
			pending:     []string{"D"},
			wantErr:     true,
		},
		{
			name:                  "Active actions should finish after error occurs with waitForOrphansTimeout",
			waitForOrphansTimeout: 20 * time.Second,
			injectError:           true,
			completed:             []string{"A", "C"},
			errors:                []string{"B"},
			pending:               []string{"D"},
			wantErr:               true,
		},
		{
			name:                  "Actions did not finish after error occurs with waitForOrphansTimeout",
			waitForOrphansTimeout: 1 * time.Second,
			injectError:           true,
			completed:             []string{"A"},
			errors:                []string{"B"},
			pending:               []string{"D"},
			wantErr:               true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {

			// Prepare actions A -> B; A -> C; C->D where C is long lasting operation.
			a := &testAction{name: "A", events: EventList{StringEvent("A")}}
			b := &testAction{name: "B"}
			if tc.injectError {
				b.err = fmt.Errorf("B action in error")
			}
			b.Want = EventList{StringEvent("A")}
			c := &testAction{
				name:   "C",
				events: EventList{StringEvent("C")},
				runHook: func(ctx context.Context) error {
					t.Log("Action c run hook, wait 5sec")
					time.Sleep(5 * time.Second)
					t.Log("Action c run hook, finish wait")
					return nil
				},
			}
			c.Want = EventList{StringEvent("A")}
			d := &testAction{name: "D"}
			d.Want = EventList{StringEvent("C")}
			actions := []Action{a, b, c, d}

			mockCloud := cloud.NewMockGCE(&cloud.SingleProjectRouter{ID: "proj1"})
			ex, err := NewParallelExecutor(mockCloud,
				actions,
				ErrorStrategyOption(StopOnError),
				TimeoutOption(tc.timeout),
				WaitForOrphansTimeoutOption(tc.waitForOrphansTimeout),
			)
			if err != nil {
				t.Fatalf("NewParallelExecutor(_, _, %v, %v) = %v; want nil", tc.timeout, tc.waitForOrphansTimeout, err)
			}
			result, err := ex.Run(context.Background())

			t.Logf("result.Completed: %v", result.Completed)
			t.Logf("result.Error: %v", result.Errors)
			t.Logf("result.Pending: %v", result.Pending)

			gotErr := err != nil
			if tc.wantErr != gotErr {
				t.Fatalf("NewParallelExecutor() = %v, got error: %v want error: %v", err, gotErr, tc.wantErr)
			}
			got := sortedStrings(result.Completed, func(a Action) string { return a.(*testAction).name })
			if diff := cmp.Diff(got, tc.completed); diff != "" {
				t.Errorf("completed: diff -got,+want: %s", diff)
			}

			got = sortedStrings(result.Pending, func(a Action) string { return a.(*testAction).name })
			if diff := cmp.Diff(got, tc.pending); diff != "" {
				t.Errorf("pending: diff -got,+want: %s", diff)
			}

			got = sortedStrings(result.Errors, func(a ActionWithErr) string { return a.Action.(*testAction).name })
			if diff := cmp.Diff(got, tc.errors); diff != "" {
				t.Errorf("errors: diff -got,+want: %s", diff)
			}
		})
	}
}
