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
	"testing"

	"github.com/google/go-cmp/cmp"
)

func testCopier(t *testing.T) *copier {
	lfn := func(msg string, kv ...any) {
		for i := 0; i < len(kv)/2; i++ {
			msg += fmt.Sprintf(" %v: %v", kv[i*2], kv[i*2+1])
		}
		t.Log(msg)
	}
	return newCopier(copierLogS(lfn))
}

func TestCopyBasic(t *testing.T) {
	v := reflect.ValueOf

	var (
		v1 bool
		v2 string
		v3 int
		v4 int8
		v5 int16
		v6 int32
		v7 int64
		v8 uint
		v9 uint8
		va uint16
		vb uint32
		vc uint64
		vd float32
		ve float64
	)

	for _, tc := range []struct {
		name      string
		dest, src reflect.Value
		want      any
	}{
		{name: "bool", dest: v(&v1).Elem(), src: v(true), want: true},
		{name: "string", dest: v(&v2).Elem(), src: v("abc"), want: "abc"},
		{name: "int", dest: v(&v3).Elem(), src: v(int(13)), want: int(13)},
		{name: "int8", dest: v(&v4).Elem(), src: v(int8(13)), want: int8(13)},
		{name: "int16", dest: v(&v5).Elem(), src: v(int16(13)), want: int16(13)},
		{name: "int32", dest: v(&v6).Elem(), src: v(int32(13)), want: int32(13)},
		{name: "int64", dest: v(&v7).Elem(), src: v(int64(13)), want: int64(13)},
		{name: "uint", dest: v(&v8).Elem(), src: v(uint(13)), want: uint(13)},
		{name: "uint8", dest: v(&v9).Elem(), src: v(uint8(13)), want: uint8(13)},
		{name: "uint16", dest: v(&va).Elem(), src: v(uint16(13)), want: uint16(13)},
		{name: "uint32", dest: v(&vb).Elem(), src: v(uint32(13)), want: uint32(13)},
		{name: "uint64", dest: v(&vc).Elem(), src: v(uint64(13)), want: uint64(13)},
		{name: "float32", dest: v(&vd).Elem(), src: v(float32(13)), want: float32(13)},
		{name: "float64", dest: v(&ve).Elem(), src: v(float64(13)), want: float64(13)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := testCopier(t).doBasic(Path{}, tc.dest, tc.src)
			if err != nil {
				t.Fatalf("copyBasic() = %v, want nil", err)
			}
			if !reflect.DeepEqual(tc.dest.Interface(), tc.want) {
				t.Fatalf("copyBasic(); got %v, want %v", tc.dest.Interface(), tc.want)
			}
		})
	}
}

func TestCopyPointer(t *testing.T) {
	v := reflect.ValueOf

	type stt struct {
		A int
	}
	type ptrTypes struct {
		I  *int
		S  *string
		ST *stt
	}

	src := ptrTypes{I: new(int), S: new(string), ST: new(stt)}
	*src.I = 13
	*src.S = "hello"
	src.ST.A = 42
	dest := ptrTypes{I: new(int), S: new(string), ST: new(stt)}
	var nilDest ptrTypes

	for _, tc := range []struct {
		name      string
		src, dest reflect.Value
		wantErr   bool
		want      any
	}{
		{
			name: "*int",
			src:  v(src.I),
			dest: v(dest.I),
			want: 13,
		},
		{
			name: "*string",
			src:  v(src.S),
			dest: v(dest.S),
			want: "hello",
		},
		{
			name: "*struct",
			src:  v(src.ST),
			dest: v(dest.ST),
			want: stt{A: 42},
		},
		{
			name: "*int nilDest",
			src:  v(src.I),
			dest: v(&nilDest.I).Elem(),
			want: 13,
		},
		{
			name: "*string nilDest",
			src:  v(src.S),
			dest: v(&nilDest.S).Elem(),
			want: "hello",
		},
		{
			name: "*struct nilDest",
			src:  v(src.ST),
			dest: v(&nilDest.ST).Elem(),
			want: stt{A: 42},
		},
		{
			name:    "invalid types",
			src:     v(1),
			dest:    v(1),
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := testCopier(t).doPointer(Path{}, tc.dest, tc.src)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Fatalf("copy() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
			if !gotErr && !reflect.DeepEqual(tc.dest.Elem().Interface(), tc.want) {
				t.Errorf("dest = %v, want %v", tc.dest.Elem().Interface(), tc.want)
			}
		})
	}
}

func TestCopySlice(t *testing.T) {
	newI := func(x int) *int { return &x }
	v := reflect.ValueOf

	type st1 struct{ I int }
	type st2 struct{ I int }

	for _, tc := range []struct {
		name      string
		src, dest reflect.Value
		wantErr   bool
		want      any
	}{
		{
			name: "nil slice",
			src:  v([]string{}),
			dest: v(&[]string{}).Elem(),
			want: []string{},
		},
		{
			name: "slice len=1",
			src:  v([]string{"abc"}),
			dest: v(&[]string{}).Elem(),
			want: []string{"abc"},
		},
		{
			name: "slice len=2",
			src:  v([]string{"abc", "zzz"}),
			dest: v(&[]string{}).Elem(),
			want: []string{"abc", "zzz"},
		},
		{
			name: "slice len=2 ovewrite",
			src:  v([]string{"abc", "zzz"}),
			dest: v(&[]string{"xxx"}).Elem(),
			want: []string{"abc", "zzz"},
		},
		{
			name: "slice of pointers",
			src:  v([]*int{newI(42), newI(99)}),
			dest: v(&[]*int{}).Elem(),
			want: []*int{newI(42), newI(99)},
		},
		{
			name: "translate struct types",
			src:  v([]st1{{I: 1}, {I: 2}}),
			dest: v(&[]st2{}).Elem(),
			want: []st2{{I: 1}, {I: 2}},
		},
		{
			name: "translate struct *types",
			src:  v([]*st1{{I: 1}, {I: 2}}),
			dest: v(&[]*st2{}).Elem(),
			want: []*st2{{I: 1}, {I: 2}},
		},
		{
			name:    "invalid types",
			src:     v(1),
			dest:    v(&[]string{}).Elem(),
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := testCopier(t).doSlice(Path{}, tc.dest, tc.src)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Fatalf("copy() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
			if gotErr {
				return
			}

			if diff := cmp.Diff(tc.dest.Interface(), tc.want); diff != "" {
				t.Errorf("-got,+want = %s", diff)
			}
		})
	}
}

func TestCopyStruct(t *testing.T) {
	newI := func(x int) *int { return &x }
	v := reflect.ValueOf

	type S1 struct {
		A int
		B string
		D []int
		P *int

		ServerResponse string
	}
	type S2 struct {
		A int
		C string
		D []int
		P *int

		ServerResponse string
	}
	type S3 struct {
		A S1
		B *S2
	}
	type S4 struct {
		A S2
		B *S1
	}

	for _, tc := range []struct {
		name        string
		src, dest   reflect.Value
		wantErr     bool
		want        any
		wantMissing []string
	}{
		{
			name: "zero value",
			src:  v(S1{}),
			dest: v(&S2{A: 13, P: newI(10)}).Elem(),
			want: S2{},
		},
		{
			name:        "missing field B",
			src:         v(S1{B: "abc"}),
			dest:        v(&S2{}).Elem(),
			want:        S2{},
			wantMissing: []string{".B"},
		},
		{
			name:        "copy fields that exist (S1 to S2)",
			src:         v(S1{A: 13, B: "abc", D: []int{10, 11, 12}, P: newI(7)}),
			dest:        v(&S2{C: "xyz"}).Elem(),
			want:        S2{A: 13, C: "xyz", D: []int{10, 11, 12}, P: newI(7)},
			wantMissing: []string{".B"},
		},
		{
			name:        "copy fields that exist (S2 to S1)",
			src:         v(S2{A: 13, C: "xyz"}),
			dest:        v(&S1{B: "abc", P: newI(7)}).Elem(),
			want:        S1{A: 13, B: "abc"},
			wantMissing: []string{".C"},
		},
		{
			name: "zero src does not clobber dest",
			src:  v(S1{}),
			dest: v(&S2{C: "xyz"}).Elem(),
			want: S2{C: "xyz"},
		},
		{
			name: "nested struct",
			src: v(S3{
				A: S1{A: 12, B: "abc"},
				B: &S2{D: []int{7}},
			}),
			dest:        v(&S4{}).Elem(),
			want:        S4{A: S2{A: 12}, B: &S1{D: []int{7}}},
			wantMissing: []string{".A.B"},
		},
		{
			name: "ServerResponse is not copied",
			src:  v(S1{ServerResponse: "abc"}),
			dest: v(&S2{}).Elem(),
			want: S2{},
		},
		{
			name:    "invalid type",
			src:     v(1),
			dest:    v(&S1{}).Elem(),
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cc := testCopier(t)
			err := cc.doStruct(Path{}, tc.dest, tc.src)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Fatalf("copy() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
			if gotErr {
				return
			}
			if !reflect.DeepEqual(tc.dest.Interface(), tc.want) {
				t.Errorf("dest = %+v, want %+v", tc.dest.Interface(), tc.want)
			}
			if len(cc.missing) != len(tc.wantMissing) {
				t.Fatalf("cc.Missing = %v; want %v", cc.missing, tc.wantMissing)
			}
			for i := range tc.wantMissing {
				if cc.missing[i].Path.String() != tc.wantMissing[i] {
					t.Fatalf("cc.Missing[i] = %q, want %q", cc.missing[i].Path.String(), tc.wantMissing[i])
				}
			}
		})
	}
}

func TestCopyMap(t *testing.T) {
	type st struct{ I int }

	v := reflect.ValueOf
	var nilmap map[string]int

	for _, tc := range []struct {
		name      string
		dest, src reflect.Value
		want      any
		wantErr   bool
	}{
		{
			name: "empty map",
			dest: v(&map[string]int{}).Elem(),
			src:  v(map[string]int{}),
			want: map[string]int{},
		},
		{
			name: "map[string]int",
			dest: v(&map[string]int{}).Elem(),
			src:  v(map[string]int{"abc": 12}),
			want: map[string]int{"abc": 12},
		},
		{
			name: "map[string][]string",
			dest: v(&map[string][]string{}).Elem(),
			src:  v(map[string][]string{"abc": {"x"}}),
			want: map[string][]string{"abc": {"x"}},
		},
		{
			name: "map[string]*struct",
			dest: v(&map[string]*st{}).Elem(),
			src:  v(map[string]*st{}),
			want: map[string]*st{},
		},
		{
			name: "copy struct",
			dest: v(&map[int]st{}).Elem(),
			src:  v(map[int]st{1: {I: 20}}),
			want: map[int]st{1: {I: 20}},
		},
		{
			name: "nil map",
			dest: v(&map[string]int{}).Elem(),
			src:  v(nilmap),
			want: nilmap,
		},
		{
			name:    "non-basic key",
			dest:    v(&map[string]int{}).Elem(),
			src:     v(map[struct{}]int{}),
			wantErr: true,
		},
		{
			name:    "mismatched key types",
			dest:    v(&map[string]int{}).Elem(),
			src:     v(map[int]int{}),
			wantErr: true,
		},
		{
			name:    "mismatched value types",
			dest:    v(&map[int]*int{}).Elem(),
			src:     v(map[int][]int{}),
			wantErr: true,
		},
		{
			name:    "pointer values currently not supported",
			dest:    v(&map[int]*int{}).Elem(),
			src:     v(map[int]*int{1: new(int)}),
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cc := testCopier(t)
			err := cc.doMap(Path{}, tc.dest, tc.src)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Fatalf("copyMap() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
			if gotErr {
				return
			}
			if diff := cmp.Diff(tc.dest.Interface(), tc.want); diff != "" {
				t.Fatalf("copyMap: -got,+want: %s (dest=%T)", diff, tc.dest.Interface())
			}
		})
	}
}

func TestCopyMetaFields(t *testing.T) {
	type st1 struct {
		SomeField    string
		St1OnlyField string

		NullFields      []string
		ForceSendFields []string
	}
	type st2 struct {
		SomeField    string
		St2OnlyField string

		NullFields      []string
		ForceSendFields []string
	}

	for _, tc := range []struct {
		name    string
		fn      string
		src     st1
		dest    st2
		want    st2
		wantErr bool
	}{
		{
			name: "empty ForceSendFields",
			fn:   "ForceSendFields",
		},
		{
			name: "propagate fields",
			fn:   "ForceSendFields",
			src:  st1{ForceSendFields: []string{"SomeField"}},
			want: st2{ForceSendFields: []string{"SomeField"}},
		},
		{
			name: "leave unknown fields alone",
			fn:   "ForceSendFields",
			src:  st1{ForceSendFields: []string{"SomeField"}},
			dest: st2{ForceSendFields: []string{"St2OnlyField"}},
			want: st2{ForceSendFields: []string{"SomeField", "St2OnlyField"}},
		},
		{
			name: "NullFields",
			fn:   "NullFields",
			src:  st1{NullFields: []string{"SomeField"}},
			dest: st2{NullFields: []string{"St2OnlyField"}},
			want: st2{NullFields: []string{"SomeField", "St2OnlyField"}},
		},
		{
			name:    "field does not exist in src",
			fn:      "ForceSendFields",
			src:     st1{ForceSendFields: []string{"InvalidField"}},
			wantErr: true,
		},
		{
			name: "field does not exist in dest, should not appear in metafield",
			fn:   "ForceSendFields",
			src:  st1{ForceSendFields: []string{"St1OnlyField"}},
			dest: st2{ForceSendFields: []string{}},
			want: st2{ForceSendFields: []string{}},
		},
		{
			name: "mix of all fields",
			fn:   "ForceSendFields",
			src:  st1{ForceSendFields: []string{"SomeField", "St1OnlyField"}},
			dest: st2{ForceSendFields: []string{"St2OnlyField"}},
			want: st2{ForceSendFields: []string{"SomeField", "St2OnlyField"}},
		},
	} {
		t.Run(tc.name, func(*testing.T) {
			srcV := reflect.ValueOf(&tc.src).Elem()
			destV := reflect.ValueOf(&tc.dest).Elem()
			srcFieldV := srcV.FieldByName(tc.fn)
			destFieldV := destV.FieldByName(tc.fn)

			cc := testCopier(t)
			err := cc.doMetaFields(Path{}, destFieldV, srcFieldV, destV, srcV)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Fatalf("copyMetaFields() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
			if gotErr {
				return
			}
			if diff := cmp.Diff(tc.dest, tc.want); diff != "" {
				t.Fatalf("copyMap: -got,+want: %s", diff)
			}
		})
	}
}
