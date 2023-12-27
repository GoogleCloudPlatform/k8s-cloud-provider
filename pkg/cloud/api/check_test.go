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

	teststruct "github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api/converter_test_types"
)

func TestCheckFieldsAreSet(t *testing.T) {
	t.Parallel()

	type sti struct {
		A               int
		NullFields      []string
		ForceSendFields []string
	}
	type st struct {
		A               int
		B               int
		S               *sti
		NullFields      []string
		ForceSendFields []string
	}

	ft := NewFieldTraits()

	ftSystemField := ft.Clone()
	ftSystemField.System(Path{}.Pointer().Field("A"))

	ftOutputOnly := ft.Clone()
	ftOutputOnly.OutputOnly(Path{}.Pointer().Field("A"))

	for _, tc := range []struct {
		name    string
		in      *st
		ft      *FieldTraits
		wantErr bool
	}{
		{
			name: "fields are all set",
			in:   &st{A: 1, B: 2, S: &sti{A: 3}},
			ft:   ft,
		},
		{
			name: "ForceSendFields",
			in: &st{
				B:               2,
				ForceSendFields: []string{"A"},
				S: &sti{
					ForceSendFields: []string{"A"},
				},
			},
			ft: ft,
		},
		{
			name: "NullFields",
			in: &st{
				A:          1,
				B:          2,
				NullFields: []string{"S"},
			},
			ft: ft,
		},
		{
			name: "missing fields",
			in: &st{
				B: 2,
			},
			ft:      ft,
			wantErr: true,
		},
		{
			name: "missing fields (substruct)",
			in: &st{
				A: 1,
				B: 2,
				S: &sti{},
			},
			ft:      ft,
			wantErr: true,
		},
		{
			name: "System field should not be set",
			in: &st{
				A: 1,
				B: 2,
				S: &sti{A: 1},
			},
			ft:      ftSystemField,
			wantErr: true,
		},
		{
			name: "OutputOnly field should not be set",
			in: &st{
				A: 1,
				B: 2,
				S: &sti{A: 1},
			},
			ft:      ftOutputOnly,
			wantErr: true,
		},
		{
			name: "Not null and also in NullFields",
			in: &st{
				A:          1,
				B:          2,
				S:          &sti{A: 1},
				NullFields: []string{"S"},
			},
			ft:      ft,
			wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkPostAccess(tc.ft, reflect.ValueOf(tc.in))
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Errorf("checkFieldsAreSet() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
		})
	}
}

// Mutually recursive types need to be declared outside of a func.
type rec2 struct{ R *rec2i }
type rec2i struct{ R *rec2 }

func TestCheckNoCycles(t *testing.T) {
	t.Parallel()

	type innerSt struct{}
	type okSt struct {
		ST  innerSt
		PST *innerSt
	}
	type rec1 struct{ R *rec1 }
	type rec3 struct{ R ****[]rec3 }
	type rec4 struct{ R map[string]rec4 }

	for _, tc := range []struct {
		name    string
		t       reflect.Type
		wantErr bool
	}{
		{name: "ok", t: reflect.TypeOf(okSt{})},
		{name: "pointer to self", t: reflect.TypeOf(rec1{}), wantErr: true},
		{name: "mutually recursive", t: reflect.TypeOf(rec2{}), wantErr: true},
		{name: "multiple indirect", t: reflect.TypeOf(rec3{}), wantErr: true},
		{name: "map", t: reflect.TypeOf(rec4{}), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkNoCycles(Path{}, tc.t, []string{})
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Errorf("cycleCheck() = %v; gotErr = %t, want %T", err, gotErr, tc.wantErr)
			}
		})
	}
}

func TestCheckResourceTypes(t *testing.T) {
	t.Parallel()

	type innerSt struct{}
	type okSt struct {
		I   int
		S   string
		PS  *string
		LS  []string
		LPS []*string
		M   map[string]int
		ST  innerSt
		PST *innerSt
	}
	type invalidSt1 struct {
		M map[innerSt]int
	}
	type invalidSt2 struct {
		C chan int
	}
	type invalidSt3 struct {
		M map[int]*innerSt
	}

	for _, tc := range []struct {
		name    string
		t       reflect.Type
		wantErr bool
	}{
		{
			name: "ok",
			t:    reflect.TypeOf(okSt{}),
		},
		{
			name:    "invalid map type",
			t:       reflect.TypeOf(invalidSt1{}),
			wantErr: true,
		},
		{
			name:    "invalid channel",
			t:       reflect.TypeOf(invalidSt2{}),
			wantErr: true,
		},
		{
			name:    "invalid map type (pointer)",
			t:       reflect.TypeOf(invalidSt3{}),
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkResourceTypes(Path{}, tc.t)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Errorf("typeCheck() = %v; gotErr = %t, want %T", err, gotErr, tc.wantErr)
			}
		})
	}
}

func TestCheckSchema(t *testing.T) {
	t.Parallel()

	type innerSt struct{}
	type okSt struct {
		Name     string
		SelfLink string

		ST  innerSt
		PST *innerSt
	}
	type badSt struct {
		C chan int

		Name     string
		SelfLink string
	}
	type badStFieldsBad struct {
		Name     int
		SelfLink string
	}

	for _, tc := range []struct {
		name    string
		t       reflect.Type
		wantErr bool
	}{
		{name: "ok", t: reflect.TypeOf(&okSt{})},
		{name: "PlaceholderType is ok", t: reflect.TypeOf(&PlaceholderType{})},
		{name: "fails cycle check", t: reflect.TypeOf(&rec2{}), wantErr: true},
		{name: "fails type check", t: reflect.TypeOf(&badSt{}), wantErr: true},
		{name: "fails type check bad fields", t: reflect.TypeOf(&badStFieldsBad{}), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkSchema(tc.t)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Errorf("ChekcSchema() = %v; gotErr = %t, want %T", err, gotErr, tc.wantErr)
			}
		})
	}
}

func TestConvertToT(t *testing.T) {
	t.Parallel()
	intVar := int(1)
	type sti struct {
		I  int
		LS []string
	}
	type stiB struct {
		I  int
		LS string
	}

	type St struct {
		I   int
		St  sti
		PSt *sti
		PS  *string
		LS  []string
		M   map[string]string
	}
	type StA struct {
		I  int
		RT *sti
		PS *string
		LS []string
		M  map[string]string
	}
	type stB struct {
		I  int
		St sti
		RT *sti
	}

	for _, tc := range []struct {
		name    string
		a       any
		b       any
		wantErr bool
	}{
		{
			name: "basic - same",
			a:    int(1),
			b:    int(1),
		},
		{
			name:    "basic - different",
			a:       int(1),
			b:       "1",
			wantErr: true,
		},
		{
			name:    "basic - pointer",
			a:       int(1),
			b:       &intVar,
			wantErr: true,
		},
		{
			name: "array - same",
			a:    [2]int{1, 2},
			b:    [2]int{1, 2},
		},
		{
			name:    "array - different",
			a:       [2]int{1, 2},
			b:       [2]string{"1", "2"},
			wantErr: true,
		},
		{
			name:    "array - slice",
			a:       [2]int{1, 2},
			b:       []int{},
			wantErr: true,
		},
		{
			name: "slice - same",
			a:    []int{},
			b:    []int{},
		},
		{
			name:    "slice - different",
			a:       []int{},
			b:       []string{},
			wantErr: true,
		},
		{
			name:    "slice - literal",
			a:       []int{},
			b:       int(1),
			wantErr: true,
		},
		{
			name:    "slice - struct",
			a:       []St{},
			b:       St{},
			wantErr: true,
		},
		{
			name: "slice with pointer - same",
			a:    []*St{},
			b:    []*St{},
		},
		{
			name:    "slice with pointer - different",
			a:       []*St{},
			b:       []*StA{},
			wantErr: true,
		},
		{
			name:    "slice with pointer - slice with struct",
			a:       []*St{},
			b:       []St{},
			wantErr: true,
		},
		{
			name:    "slice with pointer - pointer to struct",
			a:       []*St{},
			b:       &St{},
			wantErr: true,
		},
		{
			name: "struct - same",
			a:    St{},
			b:    St{},
		},
		{
			name:    "struct - missing field",
			a:       St{},
			b:       StA{},
			wantErr: true,
		},
		{
			name: "struct - subType",
			a:    teststruct.St{},
			b:    St{},
		},
		{
			name:    "struct - inner struct different",
			a:       teststruct.StA{},
			b:       StA{},
			wantErr: true,
		},
		{
			name: "pointer - same",
			a:    &St{},
			b:    &St{},
		},
		{
			name:    "pointer - different",
			a:       &St{},
			b:       &StA{},
			wantErr: true,
		},
		{
			name:    "pointer - inner struct different",
			a:       &teststruct.StA{},
			b:       &StA{},
			wantErr: true,
		},
		{
			name:    "pointer - type",
			a:       &St{},
			b:       St{},
			wantErr: true,
		},
		{
			name: "map - same",
			a:    map[string]St{},
			b:    map[string]St{},
		},
		{
			name:    "map - different key",
			a:       map[string]St{},
			b:       map[int]St{},
			wantErr: true,
		},
		{
			name:    "map - different value",
			a:       map[string]int{},
			b:       map[string]St{},
			wantErr: true,
		},
		{
			name:    "map - value as a pointer",
			a:       map[string]*St{},
			b:       map[string]St{},
			wantErr: true,
		},
		{
			name: "map - key as a pointer",
			a:    map[*St]string{},
			b:    map[*St]string{},
		},
		{
			name:    "map - pointer to map",
			a:       map[int]string{},
			b:       &map[int]string{},
			wantErr: true,
		},
		{
			name:    "map - slice of key type",
			a:       map[int]string{},
			b:       []int{},
			wantErr: true,
		},
		{
			name:    "map - slice of value type",
			a:       map[int]string{},
			b:       []string{},
			wantErr: true,
		},
		{
			name:    "map - basic key type",
			a:       map[int]string{},
			b:       int(0),
			wantErr: true,
		},
		{
			name:    "map - basic value type",
			a:       map[int]string{},
			b:       "string",
			wantErr: true,
		},
		{
			name:    "channel",
			a:       make(chan int),
			b:       make(chan int),
			wantErr: true,
		},
		{
			name:    "func",
			a:       func() {},
			b:       func() {},
			wantErr: true,
		},
		{
			name: "bool",
			a:    true,
			b:    false,
		},
		{
			name:    "bool - int",
			a:       true,
			b:       intVar,
			wantErr: true,
		},
		{
			name:    "bool - pointer",
			a:       true,
			b:       &intVar,
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckStructuralSubset(Path{}, reflect.TypeOf(tc.a), reflect.TypeOf(tc.b))
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Fatalf("CheckStructuralSubset() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
		})
	}
}
