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

// NewQueue returns a new serial queue.
func NewQueue[N any]() *Queue[N] {
	return &Queue[N]{}
}

// Queue for implementing graph algorithms.
type Queue[N any] struct{ items []N }

// Pop an element to process. This may be in any order, not necessarily FIFO.
func (q *Queue[N]) Pop() N {
	node := q.items[0]
	q.items = q.items[1:]
	return node
}

// Add an element to the work queue.
func (q *Queue[N]) Add(n N) { q.items = append(q.items, n) }

// Empty returns true if the queue is empty.
func (q *Queue[N]) Empty() bool { return len(q.items) == 0 }
