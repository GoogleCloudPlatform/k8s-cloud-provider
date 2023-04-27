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

const (
	pathField      = '.'
	pathSliceIndex = '!'
	pathMapIndex   = ':'
	pathPointer    = '*'
)

// Field returns the path extended with a struct field reference.
func (p Path) Field(name string) Path {
	return append(p, string(pathField)+name)
}

// Field returns the path extended with a slice dereference.
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

// HasPrefix returns true if prefix is the prefix of this path.
func (p Path) HasPrefix(prefix Path) bool {
	if len(prefix) == 0 {
		return true
	}
	if len(prefix) > len(p) {
		return false
	}

	var i int
	for i = range prefix {
		if p[i] != prefix[i] {
			return false
		}
	}
	if i != len(prefix)-1 {
		return false
	}
	return true
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
