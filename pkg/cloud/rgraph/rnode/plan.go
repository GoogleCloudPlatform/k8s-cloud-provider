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
	"bytes"
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
)

// Plan for what will be done to the Node.
type Plan struct {
	// details is a history of Actions that were planned,
	// including previous values Set(). The current plan is at the
	// end of this list. We keep the previous values for debug
	// output.
	details []PlanDetails
}

// Operation to perform on the Node.
type Operation string

var (
	// OpUnknown means no planning has been done.
	OpUnknown Operation = "Unknown"
	// OpNothing means nothing will happen.
	OpNothing Operation = "Nothing"
	// OpCreate will create the resource.
	OpCreate Operation = "Create"
	// OpRecreate will update the resource by first deleting, then creating the
	// resource. This is necessary as some resources cannot be updated directly
	// in place.
	OpRecreate Operation = "Recreate"
	// OpUpdate will call one or more updates ethods. The specific RPCs called
	// to do the update will be specific to the resource type itself.
	OpUpdate Operation = "Update"
	// OpDelete will delete the resource.
	OpDelete Operation = "Delete"
)

// PlanDetails is a human-readable reasons describing the Sync operation that
// has been planned.
type PlanDetails struct {
	// Operation associated with this explanation.
	Operation Operation
	// Why is a human readable string describing why this operation was
	// selected.
	Why string
	// Diff is an optional description of the diff between the current and
	// wanted resources.
	Diff *api.DiffResult
}

// Op to perform.
func (p *Plan) Op() Operation {
	details := p.Details()
	if details == nil {
		return OpUnknown
	}
	return details.Operation
}

// Details returns details on the current plan.
func (p *Plan) Details() *PlanDetails {
	if len(p.details) == 0 {
		return nil
	}
	// The latest plan is at the end of the list.
	return &p.details[len(p.details)-1]
}

// Set the plan to the specified action.
func (p *Plan) Set(a PlanDetails) {
	p.details = append(p.details, a)
}

func (p *Plan) String() string {
	if p == nil || len(p.details) == 0 {
		return "no plan"
	}
	return fmt.Sprintf("%+v", p.Details())
}

// GraphvizString returns a Graphviz-formatted summary of the plan.
func (p *Plan) GraphvizString() string {
	if p == nil || len(p.details) == 0 {
		return "no plan"
	}
	curAction := p.Details()
	var s string
	s += fmt.Sprintf("%s: %s", curAction.Operation, curAction.Why)
	if curAction.Diff != nil {
		if len(curAction.Diff.Items) > 0 {
			s += "<br/>"
			for _, item := range curAction.Diff.Items {
				s += fmt.Sprintf("[DIFF] %s: %s<br/>", item.State, item.Path)
			}
		}
	}
	return s
}

// Explain returns a human-readable string that is suitable for analysis. It
// will be rather verbose.
func (p *Plan) Explain() string {
	buf := &bytes.Buffer{}

	details := p.Details()
	fmt.Fprintf(buf, "%s: %s", details.Operation, details.Why)
	if details.Diff != nil && len(details.Diff.Items) > 0 {
		fmt.Fprintln(buf)
		for _, item := range details.Diff.Items {
			fmt.Fprintf(buf, "  [DIFF] %s: %s\n", item.State, item.Path)
		}
	}
	return buf.String()
}
