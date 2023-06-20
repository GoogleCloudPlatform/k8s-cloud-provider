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
	"errors"
	"strings"
)

// actionsFromGraphStr parses a graph in the form of "A -> B -> C; B -> D" to a
// set of testActions with the corresponding dependencies.
//
// - "A -> B": B waits for A's event.
// - "!A -> B": B waits for A's event. A will have an error when executed.
// - "A -> B -> C; A -> D": shorthand for a graph of A -> B, B -> C, A -> D.
func actionsFromGraphStr(graphStr string) []Action {
	actionMap := map[string]*testAction{}
	get := func(ev string) *testAction {
		a, ok := actionMap[ev]
		if !ok {
			a = &testAction{name: ev, events: EventList{StringEvent(ev)}}
			actionMap[ev] = a
		}
		return a
	}
	// Build events from the events string.
	for _, chain := range strings.Split(graphStr, ";") {
		chain = strings.TrimSpace(chain)
		var prev string
		for _, ev := range strings.Split(chain, "->") {
			ev = strings.TrimSpace(ev)
			if ev == "" {
				continue
			}
			injectErr := ev[0] == '!'
			if injectErr {
				ev = ev[1:]
			}
			act := get(ev)
			if injectErr {
				act.err = errors.New("injected")
			}
			if prev != "" {
				act.Want = append(act.Want, StringEvent(prev))
			}
			prev = ev
		}
	}
	var actions []Action
	for _, a := range actionMap {
		actions = append(actions, a)
	}

	return actions
}
