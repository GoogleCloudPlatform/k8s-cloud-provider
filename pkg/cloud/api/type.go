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

// PlaceholderType is used to represent GCE resource type versions that either
// don't exist (e.g. there is not alpha version of the given resource) or we do
// not intend to use (e.g. we omitted the type/version from being included in
// pkg/cloud/meta).
//
// Example
//
//	// MyRes does not have an Alpha type:
//	type MyRes Resource[ga.MyRes, PlaceholderType, beta.MyRes]
type PlaceholderType struct {
	// Standard fields that the system expects to always exist on a valid
	// resource.
	Name, SelfLink              string
	NullFields, ForceSendFields []string
}

// isPlaceholderType returns true if T is of type PlaceHolderType or
// *PlaceHolderType.
func isPlaceholderType(t any) bool {
	vb := reflect.ValueOf(t)
	if vb.Kind() == reflect.Pointer {
		_, ok := vb.Interface().(*PlaceholderType)
		return ok
	}
	_, ok := vb.Interface().(PlaceholderType)
	return ok
}

type kindPredicate func(t reflect.Type) bool

func makeKindPredicate(kl ...reflect.Kind) kindPredicate {
	return func(t reflect.Type) bool {
		for _, k := range kl {
			if t.Kind() == k {
				return true
			}
		}
		return false
	}
}

var (
	ptrT    = makeKindPredicate(reflect.Pointer)
	sliceT  = makeKindPredicate(reflect.Slice)
	structT = makeKindPredicate(reflect.Struct)
	basicT  = makeKindPredicate([]reflect.Kind{
		reflect.Bool,
		reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
	}...)
	stringT = makeKindPredicate(reflect.String)
)

// typeIs is true if `t` is matches the list of types, after account for
// dereferencing for nested types.
func typeIs(t reflect.Type, pl ...kindPredicate) bool {
	for i := range pl {
		if ok := pl[i](t); !ok {
			return false
		}
		if i != len(pl)-1 {
			switch t.Kind() {
			case reflect.Slice, reflect.Pointer, reflect.Array:
				t = t.Elem()
			default:
				return false
			}
		}
	}
	return true
}

func isBasicT(x reflect.Type) bool          { return typeIs(x, basicT) }
func isBasicV(x reflect.Value) bool         { return isBasicT(x.Type()) }
func isBasic(x any) bool                    { return isBasicV(reflect.ValueOf(x)) }
func isPtrToStructT(x reflect.Type) bool    { return typeIs(x, ptrT, structT) }
func isPtrToStructV(x reflect.Value) bool   { return isPtrToStructT(x.Type()) }
func isPtrToStruct(x any) bool              { return isPtrToStructV(reflect.ValueOf(x)) }
func isPtrToBasicT(x reflect.Type) bool     { return typeIs(x, ptrT, basicT) }
func isPtrToBasicV(x reflect.Value) bool    { return isPtrToBasicT(x.Type()) }
func isPtrToBasic(x any) bool               { return isPtrToBasicV(reflect.ValueOf(x)) }
func isSliceOfStringT(x reflect.Type) bool  { return typeIs(x, sliceT, stringT) }
func isSliceOfStringV(x reflect.Value) bool { return isSliceOfStringT(x.Type()) }
func isSliceOfString(x any) bool            { return isSliceOfStringV(reflect.ValueOf(x)) }
