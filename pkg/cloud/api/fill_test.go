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

	"github.com/google/go-cmp/cmp"
)

func TestFill(t *testing.T) {
	t.Parallel()

	type inner struct {
		I int
	}

	type st struct {
		I   int
		I8  int8
		I16 int16
		I32 int32
		I64 int64
		U   uint
		U8  uint8
		U16 uint16
		U32 uint32
		U64 uint64
		F32 float32
		F64 float64
		B   bool
		S   string

		PS   *string
		IS   inner
		PIS  *inner
		LS   []string
		LSt  []inner
		LPS  []*string
		LPSt []*inner
		M    map[string]inner
		MP   map[string]*inner
		MLS  map[string][]string

		NullFields      []string
		ForceSendFields []string
		ServerResponse  int // This isn't the accurate type, but suffices for our testing.
	}

	var s st
	err := Fill(&s)
	if err != nil {
		t.Fatalf("Fill() = %v, want nil", err)
	}

	zzzStr := "ZZZ"

	want := st{
		I:    111,
		I8:   111,
		I16:  111,
		I32:  111,
		I64:  111,
		U:    111,
		U8:   111,
		U16:  111,
		U32:  111,
		U64:  111,
		F32:  11.1,
		F64:  11.1,
		B:    true,
		S:    "ZZZ",
		PS:   &zzzStr,
		IS:   inner{I: 111},
		PIS:  &inner{I: 111},
		LS:   []string{"ZZZ"},
		LSt:  []inner{{I: 111}},
		LPS:  []*string{&zzzStr},
		LPSt: []*inner{{I: 111}},
		M:    map[string]inner{"ZZZ": {I: 111}},
		MP:   map[string]*inner{"ZZZ": {I: 111}},
		MLS:  map[string][]string{"ZZZ": {"ZZZ"}},
	}
	if diff := cmp.Diff(s, want); diff != "" {
		t.Errorf("Fill(); -got,+want = %s", diff)
	}
}

func TestFillNullAndForceSend(t *testing.T) {
	t.Parallel()

	type sti struct {
		A               int
		NullFields      []string
		ForceSendFields []string
	}
	type st struct {
		A               int
		B               *string
		C               string
		D               *sti
		NullFields      []string
		ForceSendFields []string
	}

	for _, tc := range []struct {
		name    string
		ft      *FieldTraits
		in      any
		want    any
		wantErr bool
	}{
		{
			name: "fill with zero values",
			ft:   NewFieldTraits(),
			in:   &st{},
			want: &st{
				NullFields:      []string{"B", "D"},
				ForceSendFields: []string{"A", "C"},
			},
		},
		{
			name: "fill no zeros",
			ft:   NewFieldTraits(),
			in:   &st{A: 5, B: new(string), C: "x", D: &sti{A: 2}},
			want: &st{A: 5, B: new(string), C: "x", D: &sti{A: 2}},
		},
		{
			name: "fill substruct",
			ft:   NewFieldTraits(),
			in:   &st{A: 5, D: &sti{}},
			want: &st{
				A:               5,
				NullFields:      []string{"B"},
				ForceSendFields: []string{"C"},
				D: &sti{
					ForceSendFields: []string{"A"},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := fillNullAndForceSend(tc.ft, reflect.ValueOf(tc.in))
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("fillNullAndForceSend() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
			if diff := cmp.Diff(tc.in, tc.want); diff != "" {
				t.Error(diff)
			}
		})
	}
}
