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
	"testing"

	"github.com/kr/pretty"
)

func TestDiff(t *testing.T) {
	t.Parallel()

	type sti struct {
		I  int
		LS []string
	}
	type st struct {
		I   int
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
		wantDiff bool
		wantErr  bool
	}{
		{
			name: "empty",
			a:    st{},
			b:    st{},
		},
		{
			name: "basic eq",
			a:    st{I: 3},
			b:    st{I: 3},
		},
		{
			name:     "basic diff",
			a:        st{I: 5},
			b:        st{I: 10},
			wantDiff: true,
		},
		{
			name:     "struct eq",
			a:        st{St: sti{I: 5}},
			b:        st{St: sti{I: 5}},
			wantDiff: false,
		},
		{
			name:     "struct diff",
			a:        st{St: sti{I: 5}},
			b:        st{St: sti{I: 3}},
			wantDiff: true,
		},
		{
			name:     "pointer st eq",
			a:        st{PSt: &sti{I: 5}},
			b:        st{PSt: &sti{I: 5}},
			wantDiff: false,
		},
		{
			name:     "pointer st a nil",
			a:        st{},
			b:        st{PSt: &sti{}},
			wantDiff: true,
		},
		{
			name:     "pointer st b nil",
			a:        st{PSt: &sti{}},
			b:        st{},
			wantDiff: true,
		},
		{
			name:     "pointer st diff",
			a:        st{PSt: &sti{I: 5}},
			b:        st{PSt: &sti{I: 7}},
			wantDiff: true,
		},
		{
			name:     "slice eq",
			a:        st{LS: []string{"abc"}},
			b:        st{LS: []string{"abc"}},
			wantDiff: false,
		},
		{
			name:     "slice a nil",
			a:        st{},
			b:        st{LS: []string{"bbb"}},
			wantDiff: true,
		},
		{
			name:     "slice b nil",
			a:        st{LS: []string{"aaa"}},
			b:        st{},
			wantDiff: true,
		},
		{
			name:     "slice diff same len",
			a:        st{LS: []string{"aaa"}},
			b:        st{LS: []string{"bbb"}},
			wantDiff: true,
		},
		{
			name:     "slice diff len",
			a:        st{LS: []string{"aaa1", "aaa2"}},
			b:        st{LS: []string{"bbb"}},
			wantDiff: true,
		},
		{
			name:     "map eq",
			a:        st{M: map[string]string{"a": "b"}},
			b:        st{M: map[string]string{"a": "b"}},
			wantDiff: false,
		},
		{
			name:     "map a nil",
			a:        st{},
			b:        st{M: map[string]string{"a": "b"}},
			wantDiff: true,
		},
		{
			name:     "map b nil",
			a:        st{M: map[string]string{"a": "b"}},
			b:        st{},
			wantDiff: true,
		},
		{
			name:     "map diff val",
			a:        st{M: map[string]string{"a": "b"}},
			b:        st{M: map[string]string{"a": "c"}},
			wantDiff: true,
		},
		{
			name:     "map diff key",
			a:        st{M: map[string]string{"a": "b"}},
			b:        st{M: map[string]string{"b": "c"}},
			wantDiff: true,
		},
		{
			name:     "map diff len",
			a:        st{M: map[string]string{"a": "b", "b": "c"}},
			b:        st{M: map[string]string{"b": "c"}},
			wantDiff: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := diff(&tc.a, &tc.b, nil)
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

		type invalidSt struct {
			C chan int
		}

		t.Run("invalid type", func(t *testing.T) {
			_, err := diff(&invalidSt{}, &invalidSt{}, nil)
			if err == nil {
				t.Error("Diff = nil, want err")
			}
		})
	}
}
