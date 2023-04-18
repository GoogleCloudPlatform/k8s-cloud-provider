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

package api

import (
	"fmt"
	"strings"
)

// Path specifies a field in nested object. The type of the reference
// is given by the first character:
//
// - "." is a field reference
// - "!" is a slice index
// - ":" is a map key
// - "*" is a pointer deref.
type Path []string

// Field returns the path extended with a struct field reference.
func (p Path) Field(name string) Path {
	return append(p, "."+name)
}

// Field returns the path extended with a slice dereference.
func (p Path) Index(i int) Path {
	return append(p, fmt.Sprintf("!%d", i))
}

// MapIndex returns the path extended with a map index.
func (p Path) MapIndex(k any) Path {
	return append(p, fmt.Sprintf(":%v", k))
}

// Pointer returns the path extended with a pointer dereference.
func (p Path) Pointer() Path {
	return append(p, "*")
}

// Equal returns true if other is the same path.
func (p Path) Equal(other Path) bool {
	if len(p) != len(other) {
		return false
	}
	for i := range p {
		if p[i] != other[i] {
			return false
		}
	}
	return true
}

// String implements Stringer.
func (p Path) String() string {
	return strings.Join(p, "")
}
