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

func TestMetafieldAccessor(t *testing.T) {
	type st struct {
		NullFields      []string
		ForceSendFields []string
	}

	for _, tc := range []struct {
		name          string
		in            any
		wantNull      map[string]bool
		wantForceSend map[string]bool
		wantErr       bool
	}{
		{
			name:          "zero value",
			in:            &st{},
			wantNull:      map[string]bool{},
			wantForceSend: map[string]bool{},
		},
		{
			name:    "invalid",
			in:      new(int),
			wantErr: true,
		},
		{
			name:          "values",
			in:            &st{NullFields: []string{"A"}, ForceSendFields: []string{"B"}},
			wantNull:      map[string]bool{"A": true},
			wantForceSend: map[string]bool{"B": true},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, err := newMetafieldAccessor(reflect.ValueOf(tc.in).Elem())
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("newMetafieldAccessor() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
			if err != nil {
				return
			}
			if diff := cmp.Diff(a.null(), tc.wantNull); diff != "" {
				t.Errorf("a.null() = -got,+want: %s", diff)
			}
			if diff := cmp.Diff(a.forceSend(), tc.wantForceSend); diff != "" {
				t.Errorf("a.forceSend() = -got,+want: %s", diff)
			}
		})
	}
}
