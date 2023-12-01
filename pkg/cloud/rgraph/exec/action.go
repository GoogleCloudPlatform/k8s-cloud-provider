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

package exec

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
)

// Action is an operation that updates external resources. An Action depends on
// zero or more Events to be Signaled before the Action CanRun.
type Action interface {
	// CanRun returns true if all of the Events this Action is waiting for have
	// been signaled.
	CanRun() bool
	// Signal this Action with the Event that has occurred. Returns true if the
	// Action was waiting on the Event being Signaled.
	Signal(Event) bool
	// Run the Action, performing the operations. Returns a list of Events to
	// signal and/or any errors that have occurred.
	Run(context.Context, cloud.Cloud) (EventList, error)
	// DryRun simulates running the Action. Returns a list of Events to signal.
	DryRun() EventList
	// String returns a human-readable representation of the Action for logging.
	String() string

	// PendingEvents is the list of events that this Action is still waiting on.
	PendingEvents() EventList

	// Metadata returns metadata used for visualizations.
	Metadata() *ActionMetadata
}

type ActionType string

var (
	ActionTypeCreate ActionType = "Create"
	ActionTypeDelete ActionType = "Delete"
	ActionTypeUpdate ActionType = "Update"
	ActionTypeMeta   ActionType = "Meta"
	ActionTypeCustom ActionType = "Custom"
)

// ActionMetadata is used by visualizations.
type ActionMetadata struct {
	// Name of this action. This must be unique to the execution graph.
	Name string
	// Type of this action.
	Type ActionType
	// Summary is a human readable description of this action.
	Summary string
}

// ActionBase is a helper that implements some standard behaviors of common
// Action implementation.
type ActionBase struct {
	// Want are the events this action is still waiting for.
	Want EventList
	// Done tracks the events that have happened. This is for debugging.
	Done EventList
}

func (b *ActionBase) CanRun() bool             { return len(b.Want) == 0 }
func (b *ActionBase) PendingEvents() EventList { return b.Want }

func (b *ActionBase) Signal(ev Event) bool {
	for i, wantEv := range b.Want {
		if wantEv.Equal(ev) {
			b.Want = append(b.Want[0:i], b.Want[i+1:]...)
			b.Done = append(b.Done, wantEv)

			return true
		}
	}
	return false
}

// NewExistsAction returns an Action that  signals the existence of a Resource.
// It has no other side effects.
func NewExistsAction(id *cloud.ResourceID) Action {
	return &eventAction{
		events: EventList{&existsEvent{id: id}},
	}
}

func NewDoesNotExistAction(id *cloud.ResourceID) Action {
	return &eventAction{
		events: EventList{NewNotExistsEvent(id)},
	}
}

// eventAction exist only to signal events. These Actions do not have side
// effects; they are used to model the starting conditions of an execution.
type eventAction struct {
	events EventList
}

// eventAction is an Action.
var _ Action = (*eventAction)(nil)

func (*eventAction) CanRun() bool             { return true }
func (*eventAction) Signal(Event) bool        { return false }
func (a *eventAction) String() string         { return fmt.Sprintf("EventAction(%v)", a.events) }
func (*eventAction) PendingEvents() EventList { return nil }

func (a *eventAction) DryRun() EventList {
	return a.events
}

func (a *eventAction) Run(context.Context, cloud.Cloud) (EventList, error) {
	return a.events, nil
}

func (a *eventAction) Metadata() *ActionMetadata {
	return &ActionMetadata{
		Name:    fmt.Sprintf("EventAction(%v)", a.events),
		Type:    ActionTypeMeta,
		Summary: fmt.Sprintf("Signal events: %v", a.events),
	}
}
