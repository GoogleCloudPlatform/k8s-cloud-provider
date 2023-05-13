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
	"reflect"
)

// acceptor is a set of callbacks for visit() that will be invoked when
// the given type of field is traversed in the obj.
//
// Each func returns true as to whether or not to have the visit()
// descend into the element when it is a container type (slice, map,
// struct). Returning an error will abort the visit() call immediately
// and return the error.
type acceptor interface {
	onBasic(Path, reflect.Value) (bool, error)
	onPointer(Path, reflect.Value) (bool, error)
	onStruct(Path, reflect.Value) (bool, error)
	onSlice(Path, reflect.Value) (bool, error)
	onMap(Path, reflect.Value) (bool, error)
}

// acceptorFunc returns true as to whether or not to have the visit()
// descend into the element when it is a container type (slice, map,
// struct). Returning an error will abort the visit() call immediately
// and return the error.
type acceptorFunc func(Path, reflect.Value) (bool, error)

func newAcceptorFuncs() *acceptorFuncs {
	return &acceptorFuncs{
		onBasicF:   func(Path, reflect.Value) (bool, error) { return true, nil },
		onPointerF: func(Path, reflect.Value) (bool, error) { return true, nil },
		onStructF:  func(p Path, v reflect.Value) (bool, error) { return true, nil },
		onSliceF:   func(Path, reflect.Value) (bool, error) { return true, nil },
		onMapF:     func(Path, reflect.Value) (bool, error) { return true, nil },
	}
}

type acceptorFuncs struct {
	onBasicF   acceptorFunc
	onPointerF acceptorFunc
	onStructF  acceptorFunc
	onSliceF   acceptorFunc
	onMapF     acceptorFunc
}

func (f *acceptorFuncs) onBasic(p Path, v reflect.Value) (bool, error)   { return f.onBasicF(p, v) }
func (f *acceptorFuncs) onPointer(p Path, v reflect.Value) (bool, error) { return f.onPointerF(p, v) }
func (f *acceptorFuncs) onStruct(p Path, v reflect.Value) (bool, error)  { return f.onStructF(p, v) }
func (f *acceptorFuncs) onSlice(p Path, v reflect.Value) (bool, error)   { return f.onSliceF(p, v) }
func (f *acceptorFuncs) onMap(p Path, v reflect.Value) (bool, error)     { return f.onMapF(p, v) }

// acceptorFromFunc creates an acceptor struct with the same func for
// all types. This is a convenience method for simple uses of visit().
func acceptorFromFunc(f acceptorFunc) acceptor {
	return &acceptorFuncs{
		onBasicF:   f,
		onPointerF: f,
		onStructF:  f,
		onSliceF:   f,
		onMapF:     f,
	}
}

// visit the given value with the acceptor. Each type will invoke the
// corresponding on* function. For nested elements, the callbacks will
// be invoked on the outer object (e.g. slice) and the inner elements
// (each element of the slice).
func visit(v reflect.Value, a acceptor) error { return visitImpl(Path{}, v, a) }

func visitImpl(p Path, v reflect.Value, a acceptor) error {
	switch {
	case isBasicV(v):
		if _, err := a.onBasic(p, v); err != nil {
			return err
		}
	case v.Type().Kind() == reflect.Pointer:
		descend, err := a.onPointer(p, v)
		if err != nil {
			return err
		}
		if !v.IsZero() && descend {
			err := visitImpl(p.Pointer(), v.Elem(), a)
			if err != nil {
				return err
			}
		}
	case v.Type().Kind() == reflect.Struct:
		descend, err := a.onStruct(p, v)
		if err != nil {
			return err
		}
		if descend {
			for i := 0; i < v.NumField(); i++ {
				fv := v.Field(i)
				ft := v.Type().Field(i)
				if err := visitImpl(p.Field(ft.Name), fv, a); err != nil {
					return err
				}
			}
		}
	case v.Type().Kind() == reflect.Slice:
		descend, err := a.onSlice(p, v)
		if err != nil {
			return err
		}
		if descend {
			for i := 0; i < v.Len(); i++ {
				sv := v.Index(i)
				if err := visitImpl(p.Index(i), sv, a); err != nil {
					return err
				}
			}
		}
	case v.Type().Kind() == reflect.Map:
		descend, err := a.onMap(p, v)
		if err != nil {
			return err
		}
		if descend {
			for _, mk := range v.MapKeys() {
				mv := v.MapIndex(mk)
				// Create a temporary setable map
				// value for cases where visitImpl
				// wants to mutate the value in the
				// map. This slightly inefficient for
				// read-only operations.
				setableMV := reflect.New(mv.Type()).Elem()
				setableMV.Set(mv)
				if err := visitImpl(p.MapIndex(mk), setableMV, a); err != nil {
					return err
				}
				v.SetMapIndex(mk, setableMV)
			}
		}
	}
	return nil
}
