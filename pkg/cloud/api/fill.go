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
	"hash/fnv"
	"reflect"
)

// Fill obj with dummy values for testing. Slices and maps will have
// one element in them so the linked objects will have non-zero
// values.
func Fill(obj any) error {
	f := filler{}
	ac := &acceptorFuncs{
		onBasicF:   f.doBasic,
		onPointerF: f.doPointer,
		onStructF:  f.doStruct,
		onSliceF:   f.doSlice,
		onMapF:     f.doMap,
	}
	return visit(reflect.ValueOf(obj), ac)
}

type filler struct{}

func fillHash(p Path) uint64 {
	h := fnv.New64()
	h.Write([]byte(p.String()))
	return h.Sum64()
}

func fillString(p Path) string {
	// TODO
	return p.String()
}

func (f *filler) doBasic(p Path, v reflect.Value) (bool, error) {
	h := fillHash(p)

	switch v.Kind() {
	case reflect.Bool:
		v.Set(reflect.ValueOf(true))
	case reflect.String:
		v.Set(reflect.ValueOf(fillString(p)))
	case reflect.Int:
		v.Set(reflect.ValueOf(int(h)))
	case reflect.Int8:
		v.Set(reflect.ValueOf(int8(h)))
	case reflect.Int16:
		v.Set(reflect.ValueOf(int16(h)))
	case reflect.Int32:
		v.Set(reflect.ValueOf(int32(h)))
	case reflect.Int64:
		v.Set(reflect.ValueOf(int64(h)))
	case reflect.Uint:
		v.Set(reflect.ValueOf(uint(h)))
	case reflect.Uint8:
		v.Set(reflect.ValueOf(uint8(h)))
	case reflect.Uint16:
		v.Set(reflect.ValueOf(uint16(h)))
	case reflect.Uint32:
		v.Set(reflect.ValueOf(uint32(h)))
	case reflect.Uint64:
		v.Set(reflect.ValueOf(uint64(h)))
	case reflect.Float32:
		v.Set(reflect.ValueOf(float32(h) / 10))
	case reflect.Float64:
		v.Set(reflect.ValueOf(float64(h) / 10))
	default:
		return false, fmt.Errorf("invalid type for doBasic: %s", v.Type())
	}

	return true, nil
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
	v.Set(reflect.MakeSlice(v.Type(), 1, 1))
	return true, nil
}

func (f *filler) doMap(p Path, v reflect.Value) (bool, error) {
	// TODO

	return false, nil
}
