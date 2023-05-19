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

import "time"

// Tracer is a sink for tracing an execution.
type Tracer interface {
	Record(entry *TraceEntry, err error)
	Finish(pending []Action)
}

// TraceEntry represents the execution of an Action.
type TraceEntry struct {
	Action   Action
	Err      error
	Signaled []TraceSignal

	Start time.Time
	End   time.Time
}

// TraceSignal represents the signal of an Event.
type TraceSignal struct {
	Event          Event
	SignaledAction Action
}
