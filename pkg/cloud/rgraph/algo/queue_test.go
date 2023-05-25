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

package algo

import "testing"

func TestQueue(t *testing.T) {
	type action func(*Queue[int])

	addValue := func(v int) action {
		return func(q *Queue[int]) { q.Add(v) }
	}
	pop := func(want int) action {
		return func(q *Queue[int]) {
			got := q.Pop()
			if got != want {
				t.Fatalf("q.Pop() = %d, want %d", got, want)
			}
		}
	}
	checkEmpty := func(want bool) action {
		return func(q *Queue[int]) {
			if got := q.Empty(); got != want {
				t.Fatalf("q.Empty() = %t, want %t", got, want)
			}
		}
	}

	for _, tc := range []struct {
		name    string
		actions []action
	}{
		{
			name: "empty",
			actions: []action{
				checkEmpty(true),
			},
		},
		{
			name: "add pop",
			actions: []action{
				addValue(10),
				checkEmpty(false),
				pop(10),
				checkEmpty(true),
			},
		},
		{
			name: "add N pop N",
			actions: []action{
				addValue(10),
				checkEmpty(false),
				addValue(20),
				checkEmpty(false),
				addValue(30),
				checkEmpty(false),
				pop(10),
				checkEmpty(false),
				pop(20),
				checkEmpty(false),
				pop(30),
				checkEmpty(true),
			},
		},
		{
			name: "add N pop N interleaved",
			actions: []action{
				addValue(10),
				checkEmpty(false),
				addValue(20),
				checkEmpty(false),
				pop(10),
				checkEmpty(false),
				pop(20),
				checkEmpty(true),
				addValue(30),
				checkEmpty(false),
				pop(30),
				checkEmpty(true),
			},
		},
	} {
		t.Run(tc.name, func(*testing.T) {
			q := NewQueue[int]()
			for _, f := range tc.actions {
				f(q)
			}
		})
	}
}
