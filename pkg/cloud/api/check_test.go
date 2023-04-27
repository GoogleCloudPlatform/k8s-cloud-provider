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

func TestCheckFieldsAreSet(t *testing.T) {
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
