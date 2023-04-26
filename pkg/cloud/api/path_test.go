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

import "testing"

func TestPath(t *testing.T) {
	var p Path

	// Order of test cases matters as these build on each other.
	for _, tc := range []struct {
		op   func()
		want string
	}{
		{op: func() { p = p.Field("abc") }, want: ".abc"},
		{op: func() { p = p.Index(5) }, want: ".abc!5"},
		{op: func() { p = p.Pointer() }, want: ".abc!5*"},
		{op: func() { p = p.Field("def") }, want: ".abc!5*.def"},
		{op: func() { p = p.MapIndex("key1") }, want: ".abc!5*.def:key1"},
	} {
		tc.op()
		got := p.String()
		if got != tc.want {
			t.Errorf("p = %v, got %q, want %q", p, got, tc.want)
		}
	}
}

func TestPathEqual(t *testing.T) {
	for _, tc := range []struct {
		a, b Path
		want bool
	}{
		{a: Path{}, b: Path{}, want: true},
		{a: Path{}.Field("abc"), b: Path{}.Field("abc"), want: true},
		{a: Path{}.Index(1), b: Path{}.Index(1), want: true},
		{a: Path{}.MapIndex("abc"), b: Path{}.MapIndex("abc"), want: true},
		{a: Path{}.Pointer(), b: Path{}.Pointer(), want: true},
		{a: Path{}, b: Path{}.Pointer(), want: false},
		{a: Path{}.Index(0), b: Path{}.MapIndex(0), want: false},
	} {
		got := tc.a.Equal(tc.b)
		if got != tc.want {
			t.Errorf("Equal(%s, %s) = %t, want %t", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestPathHasPrefix(t *testing.T) {
	for _, tc := range []struct {
		a, b Path
		want bool
	}{
		{a: Path{}, b: Path{}, want: true},
		{a: Path{}.Pointer(), b: Path{}, want: true},
		{a: Path{}.Pointer(), b: Path{}.Pointer(), want: true},
		{a: Path{}.Field("x"), b: Path{}.Field("y"), want: false},
		{a: Path{}.Field("x").Field("y"), b: Path{}.Field("x").Field("y"), want: true},
		{a: Path{}.Field("x").Field("y"), b: Path{}.Field("x").Field("z"), want: false},
		{a: Path{}.Pointer(), b: Path{}.Pointer().Field("x"), want: false},
		{a: Path{}.Pointer(), b: Path{}.Pointer().MapIndex("x"), want: false},
		{a: Path{}.Pointer().MapIndex("x").Field("z"), b: Path{}.Pointer().MapIndex("x"), want: true},
		{a: Path{}.Pointer(), b: Path{}.Pointer().Field("x"), want: false},
		{a: Path{}.Pointer(), b: Path{}.Field("x"), want: false},
		{a: Path{}.Pointer(), b: Path{}.Field("x").Field("x"), want: false},
		{a: Path{}.Field("x").Field("x"), b: Path{}.Pointer(), want: false},
		{a: Path{}.Field("x").Field("x"), b: Path{}.Field("x"), want: true},
	} {
		got := tc.a.HasPrefix(tc.b)
		if got != tc.want {
			t.Errorf("%q.HasPrefix(%q) = %t, want %t", tc.a, tc.b, got, tc.want)
		}
	}
}
