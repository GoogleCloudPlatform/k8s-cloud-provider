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

	"github.com/kr/pretty"
)

func TestFieldTraits(t *testing.T) {
	type sti struct {
		I  int
		LS []string
	}
	type st struct {
		I   int
		S   string
		St  sti
		PSt *sti
		PS  *string
		LS  []string
		M   map[string]string
	}

	for _, tc := range []struct {
		name     string
		a        st
		b        st
		ignores  []Path
		wantDiff bool
		wantErr  bool
	}{
		{
			name:    "ignore field",
			a:       st{I: 5},
			b:       st{I: 10},
			ignores: []Path{Path{}.Pointer().Field("I")},
		},
		{
			name:     "ignore field, diff on different field",
			a:        st{I: 5, S: "abc"},
			b:        st{I: 10, S: "def"},
			ignores:  []Path{Path{}.Pointer().Field("I")},
			wantDiff: true,
		},
		{
			name:    "ignore struct",
			a:       st{St: sti{I: 10}},
			b:       st{St: sti{I: 5}},
			ignores: []Path{Path{}.Pointer().Field("St")},
		},
		{
			name:    "ignore pointer struct",
			a:       st{PSt: &sti{I: 10}},
			b:       st{PSt: &sti{I: 5}},
			ignores: []Path{Path{}.Pointer().Field("PSt")},
		},
		{
			name:    "ignore inner struct field",
			a:       st{St: sti{I: 10}},
			b:       st{St: sti{I: 5}},
			ignores: []Path{Path{}.Pointer().Field("St").Field("I")},
		},
		{
			name:     "ignore inner struct field with diff",
			a:        st{St: sti{I: 10, LS: []string{"a"}}},
			b:        st{St: sti{I: 5}},
			ignores:  []Path{Path{}.Pointer().Field("St").Field("I")},
			wantDiff: true,
		},
		{
			name:    "ignore slice",
			a:       st{St: sti{LS: []string{"abc"}}},
			b:       st{St: sti{}},
			ignores: []Path{Path{}.Pointer().Field("St")},
		},
		{
			name:    "ignore map",
			a:       st{M: map[string]string{}},
			b:       st{M: map[string]string{"a": "b"}},
			ignores: []Path{Path{}.Pointer().Field("M")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dt := &FieldTraits{}
			for _, i := range tc.ignores {
				dt.OutputOnly(i)
			}
			r, err := diff(&tc.a, &tc.b, dt)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Fatalf("Diff() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
			if gotErr {
				return
			}
			if r.HasDiff() != tc.wantDiff {
				t.Errorf("HasDiff = %t, want %t. diff = %s", r.HasDiff(), tc.wantDiff, pretty.Sprint(r))
			}
		})
	}
}

func TestFieldTraitsClone(t *testing.T) {
	dt := &FieldTraits{}
	dt.OutputOnly(Path{}.Pointer().Field("A"))

	dtc := dt.Clone()
	if !reflect.DeepEqual(dt, dtc) {
		t.Errorf("Clone() differs: dt = %+v, dt.Clone = %+v", dt, dtc)
	}
}

func TestFieldTraitsCheckSchema(t *testing.T) {
	type st struct {
		A int
		S struct {
			A int
			L []string
		}
		P *string
	}

	for _, tc := range []struct {
		name    string
		ft      *FieldTraits
		ty      reflect.Type
		wantErr bool
	}{
		{
			name: "valid",
			ft: func() *FieldTraits {
				var ret FieldTraits
				ret.OutputOnly(Path{}.Pointer().Field("A"))
				ret.OutputOnly(Path{}.Pointer().Field("S"))
				ret.OutputOnly(Path{}.Pointer().Field("S").Field("A"))
				ret.OutputOnly(Path{}.Pointer().Field("S").Field("L"))
				return &ret
			}(),
			ty: reflect.TypeOf(&st{}),
		},
		{
			name: "path is not a field",
			ft: func() *FieldTraits {
				var ret FieldTraits
				ret.OutputOnly(Path{}.Pointer())
				return &ret
			}(),
			ty:      reflect.TypeOf(&st{}),
			wantErr: true,
		},
		{
			name: "path references fields that don't exist",
			ft: func() *FieldTraits {
				var ret FieldTraits
				ret.OutputOnly(Path{}.Pointer().Field("X"))
				return &ret
			}(),
			ty:      reflect.TypeOf(&st{}),
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.ft.CheckSchema(tc.ty)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Errorf("CheckSchema() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
		})
	}
}
