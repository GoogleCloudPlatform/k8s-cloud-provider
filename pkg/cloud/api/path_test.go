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

func TestPath(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
		{a: Path{}.Index(0), b: Path{}.AnySliceIndex(), want: false},
		{a: Path{}.MapIndex(0), b: Path{}.AnySliceIndex(), want: false},
		{a: Path{}.MapIndex(0), b: Path{}.AnyMapIndex(), want: false},
		{a: Path{}.Index(0), b: Path{}.AnyMapIndex(), want: false},
		{a: Path{}.AnySliceIndex(), b: Path{}.AnyMapIndex(), want: false},
	} {
		got := tc.a.Equal(tc.b)
		if got != tc.want {
			t.Errorf("Equal(%s, %s) = %t, want %t", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestPathMatch(t *testing.T) {
	t.Parallel()

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
		{a: Path{}.Index(0), b: Path{}.AnySliceIndex(), want: true},
		{a: Path{}.Index(0).Field("abcd").Field("cde"), b: Path{}.AnySliceIndex().Field("abcd"), want: false},
		{a: Path{}.MapIndex(0), b: Path{}.AnySliceIndex(), want: false},
		{a: Path{}.MapIndex(10), b: Path{}.AnyMapIndex(), want: true},
		{a: Path{}.Index(0), b: Path{}.AnyMapIndex(), want: false},
		{a: Path{}.AnySliceIndex(), b: Path{}.Index(8), want: true},
		{a: Path{}.AnySliceIndex().Field("abcd").Field("cde"), b: Path{}.Index(2).Field("abcd"), want: false},
		{a: Path{}.AnySliceIndex(), b: Path{}.MapIndex(0), want: false},
		{a: Path{}.AnyMapIndex(), b: Path{}.MapIndex(1), want: true},
		{a: Path{}.AnyMapIndex(), b: Path{}.Index(9), want: false},
		{a: Path{}.AnyMapIndex(), b: Path{}.AnySliceIndex(), want: false},
	} {
		if got := tc.a.Match(tc.b); got != tc.want {
			t.Errorf("%s.Match(%s) = %t, want %t", tc.a, tc.b, got, tc.want)
		}
		// Match is symmetrical. Check reversed order.
		if got := tc.b.Match(tc.a); got != tc.want {
			t.Errorf("%s.Match(%s) = %t, want %t", tc.b, tc.a, got, tc.want)
		}
	}
}

func TestPathHasPrefix(t *testing.T) {
	t.Parallel()

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
		{a: Path{}.Field("x").Index(10), b: Path{}.Field("x").AnySliceIndex(), want: true},
		{a: Path{}.Field("x").AnySliceIndex(), b: Path{}.Field("x").Index(10), want: true},
		{a: Path{}.Field("x").Field("x").Index(10), b: Path{}.Field("x").AnySliceIndex(), want: false},
		{a: Path{}.Field("x").Index(10), b: Path{}.Field("x").AnyMapIndex(), want: false},
		{a: Path{}.Field("x").MapIndex(10), b: Path{}.Field("x").AnyMapIndex(), want: true},
		{a: Path{}.Field("x").AnyMapIndex(), b: Path{}.Field("x").MapIndex(10), want: true},
	} {
		if got := tc.a.HasPrefix(tc.b); got != tc.want {
			t.Errorf("%q.HasPrefix(%q) = %t, want %t", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestResolveType(t *testing.T) {
	t.Parallel()

	type st struct {
		A int
		B []string
		C map[string]int
		D *float32

		S struct {
			A int
			S struct {
				A int
			}
		}
		P *struct {
			A int
		}
	}

	for _, tc := range []struct {
		name     string
		in       any
		p        Path
		wantType reflect.Type
		wantErr  bool
	}{
		{
			name:     "st.A",
			in:       &st{},
			p:        Path{}.Pointer().Field("A"),
			wantType: reflect.TypeOf(int(1)),
		},
		{
			name:     "st.B",
			in:       &st{},
			p:        Path{}.Pointer().Field("B").Index(0),
			wantType: reflect.TypeOf("x"),
		},
		{
			name:     "st.C",
			in:       &st{},
			p:        Path{}.Pointer().Field("C").MapIndex("x"),
			wantType: reflect.TypeOf(int(1)),
		},
		{
			name:     "st.D",
			in:       &st{},
			p:        Path{}.Pointer().Field("D").Pointer(),
			wantType: reflect.TypeOf(float32(1)),
		},
		{
			name:     "st.S.S.A",
			in:       &st{},
			p:        Path{}.Pointer().Field("S").Field("S").Field("A"),
			wantType: reflect.TypeOf(int(1)),
		},
		{
			name:     "st.P.A",
			in:       &st{},
			p:        Path{}.Pointer().Field("P").Pointer().Field("A"),
			wantType: reflect.TypeOf(int(1)),
		},
		{
			name:    "no such field",
			in:      &st{},
			p:       Path{}.Pointer().Field("X"),
			wantErr: true,
		},
		{
			name:    "type mismatch: struct",
			in:      &st{},
			p:       Path{}.Field("X"),
			wantErr: true,
		},
		{
			name:    "type mismatch: slice",
			in:      &st{},
			p:       Path{}.Pointer().Index(0),
			wantErr: true,
		},
		{
			name:    "type mismatch: map",
			in:      &st{},
			p:       Path{}.Pointer().MapIndex("x"),
			wantErr: true,
		},
		{
			name:    "type mismatch: pointer",
			in:      &st{},
			p:       Path{}.Pointer().Pointer(),
			wantErr: true,
		},
		{
			name:    "type mismatch: path too long",
			in:      &st{},
			p:       Path{}.Pointer().Field("A").Field("X"),
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ty, err := tc.p.ResolveType(reflect.TypeOf(tc.in))
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("ResolveType() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			} else if gotErr {
				return
			}
			if ty != tc.wantType {
				t.Fatalf("ResolveType() = %v, want %v", ty, tc.wantType)
			}
		})
	}
}

func TestIsSliceIndex(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		desc    string
		element string
		want    bool
	}{
		{
			desc:    "slice index short number",
			element: "!0",
			want:    true,
		},
		{
			desc:    "slice index long number",
			element: "!5679",
			want:    true,
		},
		{
			desc:    "slice any index",
			element: "!#",
		},
		{
			desc:    "slice key with string",
			element: "!abcd",
		},
		{
			desc:    "slice key",
			element: "!",
		},
		{
			desc:    "map key",
			element: ":0",
		},
		{
			desc:    "pointer key",
			element: "*0",
		},
		{
			desc:    "single number",
			element: "10",
		},
		{
			desc:    "field reference with number",
			element: ".10",
		},
		{
			desc:    "string",
			element: "abcd",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			if got := isSliceIndex(tc.element); got != tc.want {
				t.Errorf("isSliceIndex(%s) = %v, want = %v", tc.element, got, tc.want)
			}
		})
	}
}
func TestIsMatchElement(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		desc string
		a    string
		b    string
		want bool
	}{
		{
			desc: "slice index short number",
			a:    "!0",
			b:    "!0",
			want: true,
		},
		{
			desc: "slice index long number",
			a:    "!100",
			b:    "!100",
			want: true,
		},
		{
			desc: "slice index wildcard matching",
			a:    "!#",
			b:    "!0",
			want: true,
		},
		{
			desc: "slice index long wildcard matching",
			a:    "!#",
			b:    "!500",
			want: true,
		},
		{
			desc: "map key ",
			a:    ":abc",
			b:    ":abc",
			want: true,
		},
		{
			desc: "map key wildcard matching",
			a:    ":abc",
			b:    ":#",
			want: true,
		},
		{
			desc: "struct field",
			a:    ".a",
			b:    ".a",
			want: true,
		},
		{
			desc: "pointer field",
			a:    "*a",
			b:    "*a",
			want: true,
		},
		{
			desc: "slice index does not match",
			a:    "!10",
			b:    "!0",
		},
		{
			desc: "slice index does not match map key",
			a:    ":0",
			b:    "!0",
		},
		{
			desc: "slice any index does not match map key",
			a:    ":0",
			b:    "!#",
		},
		{
			desc: "slice index does not match map any key",
			a:    ":0",
			b:    "!#",
		},
		{
			desc: "slice index does not match map index wildcard matching",
			a:    ":#",
			b:    "!#",
		},
		{
			desc: "slice index does not match struct",
			a:    "!0",
			b:    ".a",
		},
		{
			desc: "map key does not match",
			a:    ":a",
			b:    ":abc",
		},
		{
			desc: "map key does not match struct",
			a:    ":a",
			b:    ".a",
		},
		{
			desc: "map key does not match pointer",
			a:    ":abc",
			b:    "*abc",
		},
		{
			desc: "struct does not match pointer",
			a:    ".abc",
			b:    "*abc",
		},
		{
			desc: "struct does not match",
			a:    ".abc",
			b:    ".a",
		},
		{
			desc: "pointer does not match",
			a:    "*abc",
			b:    "*a",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			got := isMatch(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("isMatch(%s, %s) = %v, want = %v", tc.a, tc.b, got, tc.want)
			}
			// isMatch is symmetrical, check that reversed order returned the
			// same value

			if revGot := isMatch(tc.b, tc.a); got != revGot {
				t.Errorf("isMatch(%s, %s) = %v, want = %v", tc.b, tc.a, revGot, got)
			}
		})
	}
}
