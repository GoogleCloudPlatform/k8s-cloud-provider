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

// checkPostAccess validates the fields for consistency. See the error messages
// below for the properties being checked.
func checkPostAccess(traits *FieldTraits, v reflect.Value) error {
	acc := newAcceptorFuncs()
	acc.onStructF = func(p Path, v reflect.Value) (bool, error) {
		if p.Equal(Path{}.Pointer().Field("ServerResponse")) {
			return false, nil
		}

		acc, err := newMetafieldAccessor(v)
		if err != nil {
			return false, fmt.Errorf("checkPostAccess %v: %w", p, err)
		}
		for i := 0; i < v.NumField(); i++ {
			ft := v.Type().Field(i)
			if ft.Name == "NullFields" || ft.Name == "ForceSendFields" {
				continue
			}
			fType := traits.FieldType(p.Field(ft.Name))
			fv := v.Field(i)
			fp := p.Field(ft.Name)

			switch fType {
			case FieldTypeSystem:
				if !fv.IsZero() {
					return false, fmt.Errorf("%s has a non-zero value (%v) but is a System field", fv.Interface(), fp)
				}
			case FieldTypeOutputOnly:
				if !fv.IsZero() {
					return false, fmt.Errorf("%s has a non-zero value (%v) but is an OutputOnly field", fv.Interface(), fp)
				}
			case FieldTypeNonZeroValue:
				switch {
				case fv.IsZero() && !acc.inNull(ft.Name) && !acc.inForceSend(ft.Name):
					return false, fmt.Errorf("%s is zero value but not in a NullFields or ForceSendFields %v %t", fp, fv.Interface(), fv.IsZero())
				case !fv.IsZero() && acc.inNull(ft.Name):
					return false, fmt.Errorf("%s is non-nil and also in NullFields", fp)
				}
			case FieldTypeOrdinary, FieldTypeAllowZeroValue:
				continue
			default:
				return false, fmt.Errorf("invalid FieldType: %q", fType)
			}
		}
		return true, nil
	}
	return visit(v, acc)
}

// checkNoCycles there are no cycles where a struct type appears 2+ times on the
// same path. Our algorithms requires special handling for recursive structures.
func checkNoCycles(p Path, t reflect.Type, seen []string) error {
	switch t.Kind() {
	case reflect.Slice:
		return checkNoCycles(p.Index(0), t.Elem(), seen)
	case reflect.Pointer:
		return checkNoCycles(p.Pointer(), t.Elem(), seen)
	case reflect.Map:
		// Use "x" as the placeholder for the map key in the Path for debugging
		// output purposes.
		return checkNoCycles(p.MapIndex("x"), t.Elem(), seen)
	case reflect.Struct:
		typeName := fmt.Sprintf("%s/%s", t.PkgPath(), t.Name())
		for _, seenTypeName := range seen {
			if typeName == seenTypeName {
				return fmt.Errorf("recursive type found at %s: %s", p, typeName)
			}
		}
		// Add this struct type to the list of types seen on this path.
		seen = append(seen, fmt.Sprintf("%s/%s", t.PkgPath(), t.Name()))
		for i := 0; i < t.NumField(); i++ {
			if err := checkNoCycles(p.Field(t.Field(i).Name), t.Field(i).Type, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkResourceTypes the type is something we can handle. Assumes
// checkNoCycles() passed.
func checkResourceTypes(p Path, t reflect.Type) error {
	// valid_type => basic | ...
	if isBasicT(t) {
		return nil
	}
	switch t.Kind() {
	case reflect.Pointer:
		if err := checkResourceTypes(p, t.Elem()); err != nil {
			return err
		}
	case reflect.Struct:
		// struct => {all fields are valid_type}
		for i := 0; i < t.NumField(); i++ {
			tf := t.Field(i)
			if err := checkResourceTypes(p.Field(tf.Name), tf.Type); err != nil {
				return err
			}
		}
	case reflect.Slice:
		// slice => {elements => valid_type}
		if err := checkResourceTypes(p.Index(0), t.Elem()); err != nil {
			return err
		}
	case reflect.Map:
		// map => key is basic type; value is valid_type
		if !isBasicT(t.Key()) {
			return fmt.Errorf("map key must be basic type %s: %v", p.Pointer(), t)
		}
		// Supported value types.
		if !isBasicT(t.Elem()) {
			switch t.Elem().Kind() {
			case reflect.Slice, reflect.Struct:
			default:
				return fmt.Errorf("unsupported value type %s: %v", p, t)
			}
			// Use "x" as the placeholder for the map key in the Path for debugging
			// output purposes.
			if err := checkResourceTypes(p.MapIndex("x"), t.Elem()); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported type %s: %v", p, t)
	}
	return nil
}

func checkSchema(t reflect.Type) error {
	// Run cycleCheck first, other checks will blow up if there are cycles.
	if err := checkNoCycles(Path{}, t, []string{}); err != nil {
		return err
	}
	if err := checkResourceTypes(Path{}, t); err != nil {
		return err
	}
	// Check that common fields are present.
	if t.Kind() != reflect.Pointer {
		return fmt.Errorf("object is not a pointer (%s)", t)
	}
	st := t.Elem()
	for _, fn := range []string{"Name", "SelfLink"} {
		f, ok := st.FieldByName(fn)
		if !ok || f.Type.Kind() != reflect.String {
			return fmt.Errorf("object has missing or invalid type for the %s field", fn)
		}
	}

	return nil
}

// CheckStructuralSubset checks if type From is the subset of To type. Type is a
// subset of another if it contains fields with the same type and name. This
// function is not symmetric.
func CheckStructuralSubset(from, to reflect.Type) error {
	return checkStructuralSubsetImpl(Path{}, from, to)
}

// checkStructuralSubsetImpl this is recursive function to check if type From is
// a subset of To type. Path parameter is used for better error reporting.
func checkStructuralSubsetImpl(p Path, from, to reflect.Type) error {
	if from.Kind() != to.Kind() {
		return fmt.Errorf("%s has different type: %v != %v", p, from.Kind(), to.Kind())
	}
	if isBasicT(from) {
		return nil
	}
	switch from.Kind() {
	case reflect.Pointer:
		return checkStructuralSubsetImpl(p.Pointer(), from.Elem(), to.Elem())

	case reflect.Struct:
		for i := 0; i < from.NumField(); i++ {
			af := from.Field(i)
			bf, exist := to.FieldByName(af.Name)
			if !exist {
				return fmt.Errorf("%s: type %T does not have field %v", p.String(), to, af.Name)
			}
			if err := checkStructuralSubsetImpl(p.Field(af.Name), af.Type, bf.Type); err != nil {
				return err
			}
		}
		return nil

	case reflect.Slice, reflect.Array:
		return checkStructuralSubsetImpl(p.AnySliceIndex(), from.Elem(), to.Elem())

	case reflect.Map:
		path := p.AnyMapIndex()
		err := checkStructuralSubsetImpl(path, from.Key(), to.Key())
		if err != nil {
			return err
		}
		return checkStructuralSubsetImpl(path, from.Elem(), to.Elem())
	}
	return fmt.Errorf("%s Unsupported type %v", p.String(), from.Kind())
}
