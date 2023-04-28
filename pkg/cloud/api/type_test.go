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
	"testing"
)

func TestTypeIs(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		val  any
		pl   []kindPredicate
		want bool
	}{
		{val: "abc", pl: []kindPredicate{basicT}, want: true},
		{val: true, pl: []kindPredicate{basicT}, want: true},
		{val: int(42), pl: []kindPredicate{basicT}, want: true},
		{val: int8(42), pl: []kindPredicate{basicT}, want: true},
		{val: int16(42), pl: []kindPredicate{basicT}, want: true},
		{val: int32(42), pl: []kindPredicate{basicT}, want: true},
		{val: int64(42), pl: []kindPredicate{basicT}, want: true},
		{val: uint(42), pl: []kindPredicate{basicT}, want: true},
		{val: uint8(42), pl: []kindPredicate{basicT}, want: true},
		{val: uint16(42), pl: []kindPredicate{basicT}, want: true},
		{val: uint32(42), pl: []kindPredicate{basicT}, want: true},
		{val: uint64(42), pl: []kindPredicate{basicT}, want: true},
		{val: float32(42.0), pl: []kindPredicate{basicT}, want: true},
		{val: float64(42.0), pl: []kindPredicate{basicT}, want: true},
		{val: struct{}{}, pl: []kindPredicate{basicT}, want: false},
		{val: struct{}{}, pl: []kindPredicate{structT}, want: true},
		{val: &struct{}{}, pl: []kindPredicate{structT}, want: false},
		{val: []int{}, pl: []kindPredicate{sliceT}, want: true},
		{val: &struct{}{}, pl: []kindPredicate{ptrT, structT}, want: true},
		{val: "abc", pl: []kindPredicate{stringT}, want: true},
	} {
		got := typeIs(reflect.TypeOf(tc.val), tc.pl...)
		if got != tc.want {
			t.Errorf("typeIs(%v %T) = %t; want %t", tc.val, tc.val, got, tc.want)
		}
	}
}

func TestTypeIsShortcuts(t *testing.T) {
	t.Parallel()

	intVal := 42

	for _, tc := range []struct {
		name string
		f    func(any) bool
		val  any
		want bool
	}{
		{name: "isBasic", f: isBasic, val: 1, want: true},
		{name: "isBasic", f: isBasic, val: "abc", want: true},
		{name: "isPtrToStruct", f: isPtrToStruct, val: &struct{}{}, want: true},
		{name: "isPtrToBasic", f: isPtrToBasic, val: &intVal, want: true},
		{name: "isSliceOfString", f: isSliceOfString, val: []string{"abc"}, want: true},
	} {
		got := tc.f(tc.val)
		if got != tc.want {
			t.Errorf("%s(%v %T) = %t; want %t", tc.name, tc.val, tc.val, got, tc.want)
		}
	}
}
