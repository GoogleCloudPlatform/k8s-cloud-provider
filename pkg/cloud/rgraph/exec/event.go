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
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
)

// Event occurs when an Action completes. Events can signal dependent Actions to
// be available for execution.
type Event interface {
	// Equal returns true if this event is == to other.
	Equal(other Event) bool
	// String implements Stringer.
	String() string
}

// NewExistsEvent returns and event that signals that the resource ID exists.
func NewExistsEvent(id *cloud.ResourceID) Event {
	return &existsEvent{id: id}
}

type existsEvent struct{ id *cloud.ResourceID }

func (e *existsEvent) Equal(other Event) bool {
	switch other := other.(type) {
	case *existsEvent:
		return e.id.Equal(other.id)
	}
	return false
}

func (e *existsEvent) String() string {
	return fmt.Sprintf("Exists(%v)", e.id)
}

// NewNotExistsEvent returns and event that signals that the resource ID no
// longer exists.
func NewNotExistsEvent(id *cloud.ResourceID) Event {
	return &notExistsEvent{id: id}
}

type notExistsEvent struct{ id *cloud.ResourceID }

func (e *notExistsEvent) Equal(other Event) bool {
	switch other := other.(type) {
	case *notExistsEvent:
		return e.id.Equal(other.id)
	}
	return false
}

func (e *notExistsEvent) String() string {
	return fmt.Sprintf("NotExists(%v)", e.id)
}

// NewDropRefEvent returns an event that signals that a resource reference has
// changed (From no longer refers to To).
func NewDropRefEvent(from, to *cloud.ResourceID) Event {
	return &dropRefEvent{
		from: from,
		to:   to,
	}
}

type dropRefEvent struct {
	from, to *cloud.ResourceID
}

func (e *dropRefEvent) Equal(other Event) bool {
	switch other := other.(type) {
	case *dropRefEvent:
		return e.from.Equal(other.from) && e.to.Equal(other.to)
	}
	return false
}

func (e *dropRefEvent) String() string {
	return fmt.Sprintf("DropRef(%v => %v)", e.from, e.to)
}

// StringEvent is an Event identified by a string. This is an easy way to create
// custom events.
type StringEvent string

func (e StringEvent) Equal(other Event) bool {
	switch other := other.(type) {
	case StringEvent:
		return e == other
	}
	return false
}

func (e StringEvent) String() string { return string(e) }
