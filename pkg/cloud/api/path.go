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
	"reflect"
	"strconv"
	"strings"
)

// Path specifies a field in nested object. The type of the reference
// is given by the first character:
//
// - "." is a field reference
// - "!" is a slice index
// - ":" is a map key
// - "*" is a pointer deref.
// - "#" is all array elements reference
type Path []string

const (
	pathField      = '.'
	pathSliceIndex = '!'
	pathMapIndex   = ':'
	pathPointer    = '*'

	// anySliceIndex is a slice path with wildcard index to match any index number.
	anySliceIndex = string(pathSliceIndex) + "#"
	// anyMapIndex is a map path with wildcard index to match any string key.
	anyMapIndex = string(pathMapIndex) + "#"
)

// Field returns the path extended with a struct field reference.
func (p Path) Field(name string) Path {
	return append(p, string(pathField)+name)
}

// AnySliceIndex returns a path extended to match any slice index.
func (p Path) AnySliceIndex() Path {
	return append(p, anySliceIndex)
}

// AnyMapIndex returns a path extended to match any map index.
func (p Path) AnyMapIndex() Path {
	return append(p, anyMapIndex)
}

// Index returns the path extended with a slice dereference.
func (p Path) Index(i int) Path {
	return append(p, fmt.Sprintf("%c%d", pathSliceIndex, i))
}

// MapIndex returns the path extended with a map index.
func (p Path) MapIndex(k any) Path {
	return append(p, fmt.Sprintf("%c%v", pathMapIndex, k))
}

// Pointer returns the path extended with a pointer dereference.
func (p Path) Pointer() Path {
	return append(p, string(pathPointer))
}

// Equal returns true if other is the same path. Note that this equality
// comparison does NOT interpret wildcard matches, e.g. .AnyIndex is only
// Equal() to .AnyIndex, not Equal to .Index(x).
//
// Use Match() instead to match interpreting wildcards.
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

// Match the given path, interpreting wildcard matching for the comparison. This
// function is symmetrical. Use Equal() instead to compare Paths for absolute
// equality (no wildcard interpretation).
func (p Path) Match(other Path) bool {
	if len(p) != len(other) {
		return false
	}
	for i := range p {
		if !isMatch(p[i], other[i]) {
			return false
		}
	}
	return true
}

// isMatch compares elements from the path, interpreting wildcard matching for
// the comparison.
func isMatch(a, b string) bool {
	if a == anySliceIndex && isSliceIndex(b) {
		return true
	}
	if b == anySliceIndex && isSliceIndex(a) {
		return true
	}
	if a == anyMapIndex && isMapIndex(b) {
		return true
	}
	if b == anyMapIndex && isMapIndex(a) {
		return true
	}
	return a == b
}

// HasPrefix returns true if prefix is the prefix of this path. Prefix uses
// Match() semantics for wildcards.
func (p Path) HasPrefix(prefix Path) bool {
	if len(prefix) == 0 {
		return true
	}
	if len(prefix) > len(p) {
		return false
	}

	var i int
	for i = range prefix {
		if !isMatch(prefix[i], p[i]) {
			return false
		}
	}
	return true
}

// isSliceIndex returns true if the element is a SliceIndex
func isSliceIndex(element string) bool {
	if len(element) < 2 || element[0] != pathSliceIndex {
		return false
	}
	_, err := strconv.Atoi(element[1:])
	return err == nil
}

// isMapIndex returns true if the element is a MapIndex
func isMapIndex(element string) bool {
	return len(element) > 1 && element[0] == pathMapIndex
}

func isArrayIndex(path string) bool {
	if len(path) < 2 {
		return false
	}
	return path[0] == pathSliceIndex
}

// String implements Stringer.
func (p Path) String() string {
	return strings.Join(p, "")
}

// ResolveType will attempt to traverse the type with the Path and return the
// type of the field.
func (p Path) ResolveType(t reflect.Type) (reflect.Type, error) {
	for i, x := range p {
		switch x[0] {
		case pathField:
			if t.Kind() != reflect.Struct {
				return nil, fmt.Errorf("at %s element %d, expected struct, got %s", p, i, t)
			}
			fieldName := x[1:]
			sf, ok := t.FieldByName(fieldName)
			if !ok {
				return nil, fmt.Errorf("at %s element %d, no field named %q", p, i, fieldName)
			}
			t = sf.Type
		case pathSliceIndex:
			if t.Kind() != reflect.Slice {
				return nil, fmt.Errorf("at %s element %d, expected slice, got %s", p, i, t)
			}
			t = t.Elem()
		case pathMapIndex:
			if t.Kind() != reflect.Map {
				return nil, fmt.Errorf("at %s element %d, expected map, got %s", p, i, t)
			}
			t = t.Elem()
		case pathPointer:
			if t.Kind() != reflect.Pointer {
				return nil, fmt.Errorf("at %s element %d, expected pointer, got %s", p, i, t)
			}
			t = t.Elem()
		default:
			return nil, fmt.Errorf("at %s element %d, invalid path type %q", p, i, x[0])
		}
	}
	return t, nil
}
