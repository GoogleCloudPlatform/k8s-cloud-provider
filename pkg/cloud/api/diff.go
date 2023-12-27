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
)

// TODO: how to diff force send fields? null fields? and zero values?

// diff returns a diff between A and B.
//
// TODO: the behavior of this is not symmetric -- diff(A,B) != diff(B,A).
func diff[T any](a, b *T, trait *FieldTraits) (*DiffResult, error) {
	if trait == nil {
		trait = &FieldTraits{}
	}
	d := &differ[T]{
		traits: trait,
		result: &DiffResult{},
	}
	err := d.do(Path{}, reflect.ValueOf(a), reflect.ValueOf(b))
	if err != nil {
		return nil, err
	}
	return d.result, nil
}

func diffStructs[A any, B any](a *A, b *B) (*DiffResult, error) {
	d := &differ[A]{
		traits: &FieldTraits{},
		result: &DiffResult{},
	}
	err := d.do(Path{}, reflect.ValueOf(a), reflect.ValueOf(b))
	if err != nil {
		return nil, err
	}
	return d.result, nil
}

// DiffResult gives a list of elements that differ.
type DiffResult struct {
	Items []DiffItem
}

// HasDiff is true if the result is has a diff.
func (r *DiffResult) HasDiff() bool { return len(r.Items) > 0 }

func (r *DiffResult) add(state DiffItemState, p Path, a, b reflect.Value) {
	di := DiffItem{
		State: state,
		Path:  p,
	}
	if a.IsValid() {
		// Interface() will panic if is called on unexported types in this case
		// the best we can do is to pass its name to the result.
		if a.CanInterface() {
			di.A = a.Interface()
		} else {
			di.A = a.String()
		}
	}
	if b.IsValid() {
		// Interface() will panic if is called on unexported types in this case
		// the best we can do is to pass its name to the result.
		if b.CanInterface() {
			di.B = b.Interface()
		} else {
			di.B = b.String()
		}
	}
	r.Items = append(r.Items, di)
}

// DiffItemState gives details on the diff.
type DiffItemState string

var (
	//  DiffItemDifferent means the element at the Path differs between A and B.
	DiffItemDifferent DiffItemState = "Different"
	//  DiffItemOnlyInA means the element at the Path only exists in A, the
	//  value in B is nil.
	DiffItemOnlyInA DiffItemState = "OnlyInA"
	//  DiffItemOnlyInB means the element at the Path only exists in B, the
	//  value in B is nil.
	DiffItemOnlyInB DiffItemState = "OnlyInB"
)

// DiffItem is an element that is different.
type DiffItem struct {
	State DiffItemState
	Path  Path
	A     any
	B     any
}

type differ[T any] struct {
	traits *FieldTraits
	result *DiffResult
}

func (d *differ[T]) do(p Path, av, bv reflect.Value) error {
	// cmpZero applies to pointer, slice and map values. Returns true if no
	// further diff'ing is required for the values.
	cmpZero := func() bool {
		switch {
		case av.IsZero() && bv.IsZero():
			return true
		case !av.IsZero() && bv.IsZero():
			d.result.add(DiffItemOnlyInA, p, av, bv)
			return true
		case av.IsZero() && !bv.IsZero():
			d.result.add(DiffItemOnlyInB, p, av, bv)
			return true
		}
		return false
	}

	switch {
	case isBasicV(av):
		if !av.Equal(bv) {
			d.result.add(DiffItemDifferent, p, av, bv)
		}
		return nil

	case av.Type().Kind() == reflect.Pointer:
		if cmpZero() {
			return nil
		}
		return d.do(p.Pointer(), av.Elem(), bv.Elem())

	case av.Type().Kind() == reflect.Struct:
		for i := 0; i < av.NumField(); i++ {
			afv := av.Field(i)
			aft := av.Type().Field(i)

			if aft.Name == "NullFields" || aft.Name == "ForceSendFields" {
				continue
			}

			fp := p.Field(aft.Name)
			switch d.traits.fieldType(fp) {
			case FieldTypeOutputOnly, FieldTypeSystem:
				continue
			}

			bfv := bv.FieldByName(aft.Name)
			if !bfv.IsValid() {
				d.result.add(DiffItemOnlyInA, p, av, bv)
				continue
			}
			if err := d.do(fp, afv, bfv); err != nil {
				return fmt.Errorf("differ struct %p: %w", fp, err)
			}
		}
		return nil

	case av.Type().Kind() == reflect.Slice:
		if cmpZero() {
			return nil
		}
		// If we find the list lengths are difference, don't recurse into a list
		// to compare item by item. There isn't a use case for a more fine grain
		// diff within a slice at the moment.
		if av.Len() != bv.Len() {
			d.result.add(DiffItemDifferent, p, av, bv)
			return nil
		}
		for i := 0; i < av.Len(); i++ {
			asv := av.Index(i)
			bsv := bv.Index(i)
			sp := p.Index(i)
			if err := d.do(sp, asv, bsv); err != nil {
				return fmt.Errorf("differ slice %p: %w", sp, err)
			}
		}
		return nil

	case av.Type().Kind() == reflect.Map:
		if cmpZero() {
			return nil
		}
		if av.Len() != bv.Len() {
			d.result.add(DiffItemDifferent, p, av, bv)
			return nil
		}
		// For maps of the same size, for the maps to be equal, all keys in A
		// must be present in B for these to be equal. This means we don't have
		// to check  in the opposite direction from B to A. However, this makes
		// the Diff function non-symmetric.
		for _, amk := range av.MapKeys() {
			amv := av.MapIndex(amk)
			bmv := bv.MapIndex(amk)
			mp := p.MapIndex(amk)

			if !bmv.IsValid() {
				d.result.add(DiffItemDifferent, mp, amv, bmv)
			}
			if err := d.do(mp, amv, bmv); err != nil {
				return fmt.Errorf("differ map %p: %w", mp, err)
			}
		}
		return nil
	}

	return fmt.Errorf("differ: invalid type: %s", av.Type())
}
