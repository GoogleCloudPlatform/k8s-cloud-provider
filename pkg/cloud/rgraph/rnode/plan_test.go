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

package rnode

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPlan(t *testing.T) {
	pl := &Plan{}

	for _, step := range []struct {
		name        string
		wantOp      Operation
		wantDetails *PlanDetails
		f           func(pl *Plan)
	}{
		{
			name:   "init",
			wantOp: OpUnknown,
		},
		{
			name:        "update",
			wantOp:      OpUpdate,
			wantDetails: &PlanDetails{Operation: OpUpdate},
			f:           func(pl *Plan) { pl.Set(PlanDetails{Operation: OpUpdate}) },
		},
		{
			name:        "recreate",
			wantOp:      OpRecreate,
			wantDetails: &PlanDetails{Operation: OpRecreate},
			f:           func(pl *Plan) { pl.Set(PlanDetails{Operation: OpRecreate}) },
		},
	} {
		t.Logf("Step: %s", step.name)
		if step.f != nil {
			step.f(pl)
		}
		if gotOp := pl.Op(); gotOp != step.wantOp {
			t.Errorf("pl.Op() = %q, want %q", gotOp, step.wantOp)
		}
		if diff := cmp.Diff(pl.Details(), step.wantDetails); diff != "" {
			t.Errorf("pl.Details(); -got,+want: %s", diff)
		}
	}
}
