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
	"sort"
)

func staticFiller(t reflect.Type, p Path) any {
	switch t.Kind() {
	case reflect.Bool:
		return true
	case reflect.Int:
		return int(111)
	case reflect.Int8:
		return int8(111)
	case reflect.Int16:
		return int16(111)
	case reflect.Int32:
		return int32(111)
	case reflect.Int64:
		return int64(111)
	case reflect.Uint:
		return uint(111)
	case reflect.Uint8:
		return uint8(111)
	case reflect.Uint16:
		return uint16(111)
	case reflect.Uint32:
		return uint32(111)
	case reflect.Uint64:
		return uint64(111)
	case reflect.Float32:
		return float32(11.1)
	case reflect.Float64:
		return float64(11.1)
	case reflect.String:
		return "ZZZ"
	}
	panic(fmt.Sprintf("invalid type %s", t))
}

// FillOption is an option to Fill().
type FillOption func(*filler)

// BasicFiller configures a different fill function for basic values in the
// struct.
func BasicFiller(fn func(reflect.Type, Path) any) FillOption {
	return func(f *filler) { f.basicValue = fn }
}

// Fill obj with dummy values for testing. Slices and maps will have
// one element in them so the linked objects will have non-zero
// values.
func Fill(obj any, options ...FillOption) error {
	f := filler{basicValue: staticFiller}
	for _, optFn := range options {
		optFn(&f)
	}
	ac := &acceptorFuncs{
		onBasicF:   f.doBasic,
		onPointerF: f.doPointer,
		onStructF:  f.doStruct,
		onSliceF:   f.doSlice,
		onMapF:     f.doMap,
	}
	return visit(reflect.ValueOf(obj), ac)
}

type filler struct {
	basicValue func(reflect.Type, Path) any
}

func (f *filler) doBasic(p Path, v reflect.Value) (bool, error) {
	if isNoFillPath(p) {
		return false, nil
	}

	if isBasicV(v) {
		v.Set(reflect.ValueOf(f.basicValue(v.Type(), p)))
		return true, nil
	}
	return false, fmt.Errorf("invalid type for doBasic: %s", v.Type())
}

func (f *filler) doPointer(p Path, v reflect.Value) (bool, error) {
	if v.IsZero() {
		v.Set(reflect.New(v.Type().Elem()))
	}
	return true, nil
}

// isNoFillPath returns true for paths that should be ignored for
// standard GCP resources.
func isNoFillPath(p Path) bool {
	if p.Equal(Path{}.Field("ServerResponse")) ||
		p.Equal(Path{}.Pointer().Field("ServerResponse")) {
		return true
	}
	if len(p) > 0 && (p[len(p)-1] == ".NullFields" || p[len(p)-1] == ".ForceSendFields") {
		return true
	}
	return false
}

func (f *filler) doStruct(p Path, v reflect.Value) (bool, error) {
	if isNoFillPath(p) {
		return false, nil
	}
	return true, nil
}

func (f *filler) doSlice(p Path, v reflect.Value) (bool, error) {
	if isNoFillPath(p) {
		return false, nil
	}
	// Create a list with a single element in it.
	v.Set(reflect.MakeSlice(v.Type(), 1, 1))
	// Returning true will cause the visitor to descend into the
	// list item and fill it with values.
	return true, nil
}

func (f *filler) doMap(p Path, v reflect.Value) (bool, error) {
	kt := v.Type().Key()
	kv := reflect.New(kt).Elem()
	vt := v.Type().Elem()
	// "x" is a placeholder for the key value.
	f.doBasic(p.MapIndex("x"), kv)
	newMap := reflect.MakeMapWithSize(v.Type(), 0)
	v.Set(newMap)

	if vt.Kind() == reflect.Pointer {
		mv := reflect.New(vt.Elem())
		v.SetMapIndex(kv, mv)
	} else {
		v.SetMapIndex(kv, reflect.New(vt).Elem())
	}
	// Returning true will cause the visitor to descend into the
	// map and fill it with values.
	return true, nil
}

func fillNullAndForceSend(traits *FieldTraits, v reflect.Value) error {
	acc := newAcceptorFuncs()
	acc.onStructF = func(p Path, v reflect.Value) (bool, error) {
		if p.Equal(Path{}.Pointer().Field("ServerResponse")) {
			return false, nil
		}
		acc, err := newMetafieldAccessor(v)
		if err != nil {
			return false, fmt.Errorf("fillNullAndForceSend: %w", err)
		}

		nullFields := acc.null()
		forceSendFields := acc.forceSend()

		for i := 0; i < v.NumField(); i++ {
			ft := v.Type().Field(i)
			if ft.Name == "NullFields" || ft.Name == "ForceSendFields" {
				continue
			}
			fType := traits.FieldType(p.Field(ft.Name))
			fv := v.Field(i)

			if fType == FieldTypeNonZeroValue {
				switch {
				case fv.IsZero() && fv.Type().Kind() == reflect.Pointer:
					nullFields[ft.Name] = true
				case fv.IsZero():
					forceSendFields[ft.Name] = true
				}
			}
		}

		set := func(m map[string]bool, d reflect.Value) {
			var sl []string
			for k := range m {
				sl = append(sl, k)
			}
			sort.Strings(sl)
			d.Set(reflect.ValueOf(sl))
		}

		set(nullFields, v.FieldByName(nullFieldsName))
		set(forceSendFields, v.FieldByName(forceSendFieldsName))

		return true, nil
	}

	return visit(v, acc)
}
