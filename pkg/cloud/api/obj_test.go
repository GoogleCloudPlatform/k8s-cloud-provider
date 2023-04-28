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

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/google/go-cmp/cmp"
	"github.com/kr/pretty"
)

func newTestResource[G any, A any, B any](tt TypeTrait[G, A, B]) *resource[G, A, B] {
	return NewResource(&cloud.ResourceID{
		ProjectID: "proj-1",
		Resource:  "st",
		Key:       meta.GlobalKey("obj-1"),
	}, tt)
}

type testTrait[G any, A any, B any] struct {
	BaseTypeTrait[G, A, B]
}

func (testTrait[G, A, B]) FieldTraits(meta.Version) *FieldTraits {
	ret := &FieldTraits{}

	ret.AllowZeroValue(Path{}.Pointer().Field("I"))
	ret.AllowZeroValue(Path{}.Pointer().Field("S"))
	ret.AllowZeroValue(Path{}.Pointer().Field("F"))
	ret.AllowZeroValue(Path{}.Pointer().Field("St"))
	ret.AllowZeroValue(Path{}.Pointer().Field("St").Field("I"))
	ret.AllowZeroValue(Path{}.Pointer().Field("StP"))
	ret.AllowZeroValue(Path{}.Pointer().Field("StP").Pointer().Field("I"))
	ret.AllowZeroValue(Path{}.Pointer().Field("LStr"))
	ret.AllowZeroValue(Path{}.Pointer().Field("LPStr"))
	ret.AllowZeroValue(Path{}.Pointer().Field("M"))
	ret.AllowZeroValue(Path{}.Pointer().Field("Name"))
	ret.AllowZeroValue(Path{}.Pointer().Field("SelfLink"))

	ret.AllowZeroValue(Path{}.Pointer().Field("AI"))
	ret.AllowZeroValue(Path{}.Pointer().Field("St").Field("A"))
	ret.AllowZeroValue(Path{}.Pointer().Field("StP").Pointer().Field("A"))

	ret.AllowZeroValue(Path{}.Pointer().Field("BI"))
	ret.AllowZeroValue(Path{}.Pointer().Field("St").Field("B"))
	ret.AllowZeroValue(Path{}.Pointer().Field("StP").Pointer().Field("B"))

	ret.AllowZeroValue(Path{}.Pointer().Field("ABS"))

	return ret
}

func TestResourceToX(t *testing.T) {
	t.Parallel()

	type inner struct{ I int }
	type innerA struct {
		I int
		A string
	}

	type innerB struct {
		I int
		B string
	}

	type st struct {
		I int
		S string
		F float64

		St  inner
		StP *inner

		LStr  []string
		LPStr []*string

		M map[string]int

		Name            string
		SelfLink        string
		NullFields      []string
		ForceSendFields []string
	}

	type stA struct {
		I   int
		S   string
		F   float64
		AI  int
		ABS string

		St  innerA
		StP *innerA

		LStr  []string
		LPStr []*string

		M map[string]int

		Name            string
		SelfLink        string
		NullFields      []string
		ForceSendFields []string
	}

	type stB struct {
		I   int
		S   string
		F   float64
		BI  int
		ABS string

		St  innerB
		StP *innerB

		LStr  []string
		LPStr []*string

		M map[string]int

		Name            string
		SelfLink        string
		NullFields      []string
		ForceSendFields []string
	}

	type stObj = resource[st, stA, stB]

	type testCase struct {
		name      string
		edit      func(x *st)
		editAlpha func(x *stA)
		editBeta  func(x *stB)

		want         any
		wantErr      bool
		wantAlpha    any
		wantAlphaErr bool
		wantBeta     any
		wantBetaErr  bool

		wantEditErr      bool
		wantEditAlphaErr bool
		wantEditBetaErr  bool
	}

	testCases := []testCase{
		{
			name: "basic types",
			edit: func(x *st) {
				x.I = 13
				x.S = "abc"
				x.F = 4.2
			},
			want:      &st{I: 13, S: "abc", F: 4.2},
			wantAlpha: &stA{I: 13, S: "abc", F: 4.2},
			wantBeta:  &stB{I: 13, S: "abc", F: 4.2},
		},
		{
			name: "alpha only fields",
			editAlpha: func(x *stA) {
				x.I = 12
				x.AI = 13
			},
			want:        &st{I: 12},
			wantErr:     true,
			wantAlpha:   &stA{I: 12, AI: 13},
			wantBeta:    &stB{I: 12},
			wantBetaErr: true,
		},
		{
			name: "beta only fields",
			editBeta: func(x *stB) {
				x.I = 12
				x.BI = 13
			},
			want:         &st{I: 12},
			wantErr:      true,
			wantAlpha:    &stA{I: 12},
			wantAlphaErr: true,
			wantBeta:     &stB{I: 12, BI: 13},
		},
		{
			name: "alpha beta fields",
			editBeta: func(x *stB) {
				x.I = 12
				x.ABS = "abc"
			},
			want:      &st{I: 12},
			wantErr:   true,
			wantAlpha: &stA{I: 12, ABS: "abc"},
			wantBeta:  &stB{I: 12, ABS: "abc"},
		},
		{
			name: "inner struct",
			edit: func(x *st) {
				x.St.I = 13
			},
			want:      &st{St: inner{I: 13}},
			wantAlpha: &stA{St: innerA{I: 13}},
			wantBeta:  &stB{St: innerB{I: 13}},
		},
		{
			name: "inner struct alpha only",
			editAlpha: func(x *stA) {
				x.St.I = 13
				x.St.A = "abc"
			},
			want:        &st{St: inner{I: 13}},
			wantErr:     true,
			wantAlpha:   &stA{St: innerA{I: 13, A: "abc"}},
			wantBeta:    &stB{St: innerB{I: 13}},
			wantBetaErr: true,
		},
		{
			name: "inner struct beta only",
			editBeta: func(x *stB) {
				x.St.I = 13
				x.St.B = "abc"
			},
			want:         &st{St: inner{I: 13}},
			wantErr:      true,
			wantAlpha:    &stA{St: innerA{I: 13}},
			wantAlphaErr: true,
			wantBeta:     &stB{St: innerB{I: 13, B: "abc"}},
		},
		{
			name: "inner pointer struct",
			edit: func(x *st) {
				x.StP = &inner{I: 13}
			},
			want:      &st{StP: &inner{I: 13}},
			wantAlpha: &stA{StP: &innerA{I: 13}},
			wantBeta:  &stB{StP: &innerB{I: 13}},
		},
		{
			name: "inner pointer struct alpha",
			editAlpha: func(x *stA) {
				x.StP = &innerA{I: 13}
			},
			want:      &st{StP: &inner{I: 13}},
			wantAlpha: &stA{StP: &innerA{I: 13}},
			wantBeta:  &stB{StP: &innerB{I: 13}},
		},
		{
			name: "inner pointer struct beta",
			editBeta: func(x *stB) {
				x.StP = &innerB{I: 13}
			},
			want:      &st{StP: &inner{I: 13}},
			wantAlpha: &stA{StP: &innerA{I: 13}},
			wantBeta:  &stB{StP: &innerB{I: 13}},
		},
		{
			name:      "string list",
			edit:      func(x *st) { x.LStr = []string{"a", "b"} },
			want:      &st{LStr: []string{"a", "b"}},
			wantAlpha: &stA{LStr: []string{"a", "b"}},
			wantBeta:  &stB{LStr: []string{"a", "b"}},
		},
		{
			name:      "map",
			edit:      func(x *st) { x.M = map[string]int{"a": 1} },
			want:      &st{M: map[string]int{"a": 1}},
			wantAlpha: &stA{M: map[string]int{"a": 1}},
			wantBeta:  &stB{M: map[string]int{"a": 1}},
		},
		{
			name:      "edit ga then alpha",
			edit:      func(x *st) { x.I = 11 },
			editAlpha: func(x *stA) { x.I = 42 },
			want:      &st{I: 42},
			wantAlpha: &stA{I: 42},
			wantBeta:  &stB{I: 42},
		},
		{
			name:      "edit ga then beta",
			edit:      func(x *st) { x.I = 11 },
			editBeta:  func(x *stB) { x.I = 42 },
			want:      &st{I: 42},
			wantAlpha: &stA{I: 42},
			wantBeta:  &stB{I: 42},
		},
		{
			name: "edit ga then alpha with inner struct",
			edit: func(x *st) { x.I = 11 },
			editAlpha: func(x *stA) {
				x.St.I = 42
			},
			want:      &st{I: 11, St: inner{I: 42}},
			wantAlpha: &stA{I: 11, St: innerA{I: 42}},
			wantBeta:  &stB{I: 11, St: innerB{I: 42}},
		},
		{
			name:      "ForceSendFields",
			edit:      func(x *st) { x.ForceSendFields = []string{"I"} },
			want:      &st{ForceSendFields: []string{"I"}},
			wantAlpha: &stA{ForceSendFields: []string{"I"}},
			wantBeta:  &stB{ForceSendFields: []string{"I"}},
		},
		{
			name:        "ForceSendFields alpha only",
			editAlpha:   func(x *stA) { x.ForceSendFields = []string{"AI"} },
			want:        &st{},
			wantErr:     true,
			wantAlpha:   &stA{ForceSendFields: []string{"AI"}},
			wantBeta:    &stB{},
			wantBetaErr: true,
		},
		{
			name:      "ForceSendFields alpha beta only",
			editAlpha: func(x *stA) { x.ForceSendFields = []string{"ABS"} },
			want:      &st{},
			wantErr:   true,
			wantAlpha: &stA{ForceSendFields: []string{"ABS"}},
			wantBeta:  &stB{ForceSendFields: []string{"ABS"}},
		},
		{
			name:        "ForceSendFields invalid field",
			edit:        func(x *st) { x.ForceSendFields = []string{"InvalidField"} },
			wantEditErr: true,
		},
		{
			name:      "NullFields",
			edit:      func(x *st) { x.NullFields = []string{"I"} },
			want:      &st{NullFields: []string{"I"}},
			wantAlpha: &stA{NullFields: []string{"I"}},
			wantBeta:  &stB{NullFields: []string{"I"}},
		},
		{
			name:        "NullFields alpha only",
			editAlpha:   func(x *stA) { x.NullFields = []string{"AI"} },
			want:        &st{},
			wantErr:     true,
			wantAlpha:   &stA{NullFields: []string{"AI"}},
			wantBeta:    &stB{},
			wantBetaErr: true,
		},
		{
			name:      "NullFields alpha beta only",
			editAlpha: func(x *stA) { x.NullFields = []string{"ABS"} },
			want:      &st{},
			wantErr:   true,
			wantAlpha: &stA{NullFields: []string{"ABS"}},
			wantBeta:  &stB{NullFields: []string{"ABS"}},
		},
		{
			name:        "NullFields invalid field",
			edit:        func(x *st) { x.NullFields = []string{"InvalidField"} },
			wantEditErr: true,
		},
	}

	check := func(o *stObj, tc *testCase) {
		got, err := o.ToGA()
		if gotErr := err != nil; gotErr != tc.wantErr {
			t.Fatalf("o.ToGA() = %v; gotErr = %t, wantErr = %t", err, gotErr, tc.wantErr)
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("o.ToGA() = %s, want %s", pretty.Sprint(got), pretty.Sprint(tc.want))
		}

		gotAlpha, err := o.ToAlpha()
		if gotErr := err != nil; gotErr != tc.wantAlphaErr {
			t.Fatalf("o.ToAlpha() = %v; gotErr = %t, wantErr = %t", err, gotErr, tc.wantAlphaErr)
		}
		if !reflect.DeepEqual(gotAlpha, tc.wantAlpha) {
			t.Errorf("o.ToAlpha() = %s, want %s", pretty.Sprint(gotAlpha), pretty.Sprint(tc.wantAlpha))
		}

		gotBeta, err := o.ToBeta()
		if gotErr := err != nil; gotErr != tc.wantBetaErr {
			t.Fatalf("o.ToBeta() = %v; gotErr = %t, wantErr = %t", err, gotErr, tc.wantBetaErr)
		}
		if !reflect.DeepEqual(gotBeta, tc.wantBeta) {
			t.Errorf("o.ToBeta() = %s, want %s", pretty.Sprint(gotBeta), pretty.Sprint(tc.wantBeta))
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			o := NewResource[st, stA, stB](&cloud.ResourceID{
				ProjectID: "proj-1",
				Resource:  "st",
				Key:       meta.GlobalKey("obj-1"),
			}, &testTrait[st, stA, stB]{})
			if tc.edit != nil {
				err := o.Access(tc.edit)
				if gotErr := err != nil; gotErr != tc.wantEditErr {
					t.Fatalf("Edit = %v, gotErr = %t, want %t", err, gotErr, tc.wantEditErr)
				}
				if err != nil {
					return
				}
			}
			if tc.editAlpha != nil {
				err := o.AccessAlpha(tc.editAlpha)
				if gotErr := err != nil; gotErr != tc.wantEditAlphaErr {
					t.Fatalf("Edit = %v, gotErr = %t, want %t", err, gotErr, tc.wantEditAlphaErr)
				}
				if err != nil {
					return
				}
			}
			if tc.editBeta != nil {
				err := o.AccessBeta(tc.editBeta)
				if gotErr := err != nil; gotErr != tc.wantEditBetaErr {
					t.Fatalf("Edit = %v, gotErr = %t, want %t", err, gotErr, tc.wantEditBetaErr)
				}
				if err != nil {
					return
				}
			}
			check(o, &tc)
		})
	}

	// Check that no-op calls to Edit*() do not result in changes to the output.
	t.Run("idempotent edit", func(t *testing.T) {
		for _, tc := range testCases {
			// Skip test cases where Edit() doesn't succeed.
			if tc.wantEditErr || tc.wantEditAlphaErr || tc.wantEditBetaErr {
				continue
			}

			t.Run(tc.name, func(t *testing.T) {
				o := NewResource[st, stA, stB](&cloud.ResourceID{
					ProjectID: "proj-1",
					Resource:  "st",
					Key:       meta.GlobalKey("obj-1"),
				}, &testTrait[st, stA, stB]{})

				if tc.edit != nil {
					err := o.Access(tc.edit)
					if err != nil {
						t.Fatalf("Edit = %v, want nil", err)
					}
				}
				if tc.editAlpha != nil {
					err := o.AccessAlpha(tc.editAlpha)
					if err != nil {
						t.Fatalf("EditAlpha = %v, want nil", err)
					}
				}
				if tc.editBeta != nil {
					err := o.AccessBeta(tc.editBeta)
					if err != nil {
						t.Fatalf("EditBeta = %v, want nil", err)
					}
				}
				// Force multiple calls to Edit*().
				for i := 0; i < 2; i++ {
					if err := o.Access(func(*st) {}); err != nil {
						t.Errorf("repeated call to Edit failed: %v", err)
					}
					if err := o.AccessAlpha(func(*stA) {}); err != nil {
						t.Errorf("repeated call to EditAlpha failed: %v", err)
					}
					if err := o.AccessBeta(func(*stB) {}); err != nil {
						t.Errorf("repeated call to EditBeta failed: %v", err)
					}
				}
				check(o, &tc)
			})
		}
	})
}

func TestResourceMissingFields(t *testing.T) {
	t.Parallel()

	// Test that the missing fields is correct after a sequence of edits at
	// different API versions.
	type ga struct{ A int }
	type alph struct{ A, B int }
	type beta struct{ A int }
	type vo = resource[ga, alph, beta]

	obj := newTestResource[ga, alph, beta](nil)

	// Set x.B, only available in the Alpha version of the API.
	obj.AccessAlpha(func(x *alph) { x.B = 20 })
	// The following should not overwrite the missing field information of B.
	obj.Access(func(x *ga) { x.A = 10 })
	obj.AccessBeta(func(x *beta) { x.A = 12 })
	obj.AccessAlpha(func(x *alph) { x.A = 15 })

	gaResult, err := obj.ToGA()
	if diff := cmp.Diff(gaResult, &ga{A: 15}); diff != "" {
		t.Errorf("ToGA(); -got,+want: %s", diff)
	}
	if err == nil {
		t.Error("ToGA() = nil, want error")
	}
	aResult, err := obj.ToAlpha()
	if diff := cmp.Diff(aResult, &alph{A: 15, B: 20}); diff != "" {
		t.Errorf("ToAlpha(); -got,+want: %s", diff)
	}
	if err != nil {
		t.Errorf("ToAlpha() = %v, want nil", err)
	}
	bResult, err := obj.ToBeta()
	if diff := cmp.Diff(bResult, &beta{A: 15}); diff != "" {
		t.Errorf("ToBeta(); -got,+want: %s", diff)
	}
	if err == nil {
		t.Error("ToBeta() = nil, want error")
	}
}

func TestResourceMissingMetaFields(t *testing.T) {
	t.Parallel()

	// Test that the missing fields is correct after a sequence of edits at
	// different API versions. Field is specified using a metafield.

	type ga struct {
		A               int
		ForceSendFields []string
	}
	type alph struct {
		A, B            int
		ForceSendFields []string
	}
	type beta struct {
		A               int
		ForceSendFields []string
	}
	type vo = resource[ga, alph, beta]
	obj := newTestResource[ga, alph, beta](nil)

	// Set x.B, only available in the Alpha version of the API.
	obj.AccessAlpha(func(x *alph) { x.ForceSendFields = []string{"B"} })
	// The following should not overwrite the missing field information of B.
	obj.Access(func(x *ga) { x.A = 10 })
	obj.AccessBeta(func(x *beta) { x.A = 12 })
	obj.AccessAlpha(func(x *alph) { x.A = 15 })

	gaResult, err := obj.ToGA()
	if diff := cmp.Diff(gaResult, &ga{A: 15}); diff != "" {
		t.Errorf("ToGA(); -got,+want: %s", diff)
	}
	if err == nil {
		t.Error("ToGA() = nil, want error")
	}
	aResult, err := obj.ToAlpha()
	if diff := cmp.Diff(aResult, &alph{
		A:               15,
		B:               0,
		ForceSendFields: []string{"B"},
	}); diff != "" {
		t.Errorf("ToAlpha(); -got,+want: %s", diff)
	}
	if err != nil {
		t.Errorf("ToAlpha() = %v, want nil", err)
	}
	bResult, err := obj.ToBeta()
	if diff := cmp.Diff(bResult, &beta{A: 15}); diff != "" {
		t.Errorf("ToBeta(); -got,+want: %s", diff)
	}
	if err == nil {
		t.Error("ToBeta() = nil, want error")
	}
}

func TestResourceSetX(t *testing.T) {
	t.Parallel()

	type ga struct{ A int }
	type al struct{ A, B, C int }
	type be struct{ A, B, D int }
	type vo = resource[ga, al, be]

	for _, tc := range []struct {
		name      string
		src       any
		wantGA    *ga
		wantAlpha *al
		wantBeta  *be

		setErr                   bool
		gaErr, alphaErr, betaErr bool
	}{
		{
			name:      "Set",
			src:       &ga{A: 13},
			wantGA:    &ga{A: 13},
			wantAlpha: &al{A: 13},
			wantBeta:  &be{A: 13},
		},
		{
			name:      "SetAlpha",
			src:       &al{A: 10, B: 11, C: 101},
			wantGA:    &ga{A: 10},
			wantAlpha: &al{A: 10, B: 11, C: 101},
			wantBeta:  &be{A: 10, B: 11},
			gaErr:     true,
			betaErr:   true,
		},
		{
			name:      "SetBeta",
			src:       &be{A: 13, B: 14, D: 15},
			wantGA:    &ga{A: 13},
			wantAlpha: &al{A: 13, B: 14},
			wantBeta:  &be{A: 13, B: 14, D: 15},
			gaErr:     true,
			alphaErr:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj := newTestResource[ga, al, be](nil)

			var err error
			switch src := tc.src.(type) {
			case *ga:
				err = obj.Set(src)
			case *al:
				err = obj.SetAlpha(src)
			case *be:
				err = obj.SetBeta(src)
			}
			if gotErr := err != nil; gotErr != tc.setErr {
				t.Errorf("Set*() = %v; gotErr = %t, want %t", err, gotErr, tc.setErr)
			}

			gotGA, err := obj.ToGA()
			if gotErr := err != nil; gotErr != tc.gaErr {
				t.Errorf("ToGA() = %v; gotErr = %t, want %t", err, gotErr, tc.gaErr)
			}
			if diff := cmp.Diff(gotGA, tc.wantGA); diff != "" {
				t.Errorf("ToGA(); -got,+want: %s", diff)
			}
			gotAlpha, err := obj.ToAlpha()
			if gotErr := err != nil; gotErr != tc.alphaErr {
				t.Errorf("ToAlpha() = %v; gotErr = %t, want %t", err, gotErr, tc.alphaErr)
			}
			if diff := cmp.Diff(gotAlpha, tc.wantAlpha); diff != "" {
				t.Errorf("ToAlpha(); -got,+want: %s", diff)
			}
			gotBeta, err := obj.ToBeta()
			if gotErr := err != nil; gotErr != tc.betaErr {
				t.Errorf("ToBeta() = %v; gotErr = %t, want %t", err, gotErr, tc.betaErr)
			}
			if diff := cmp.Diff(gotBeta, tc.wantBeta); diff != "" {
				t.Errorf("ToBeta(); -got,+want: %s", diff)
			}
		})
	}
}

func TestResourceCheckSchema(t *testing.T) {
	t.Parallel()

	type st struct {
		Name            string
		SelfLink        string
		I               int
		NullFields      []string
		ForceSendFields []string
	}
	type stA struct {
		Name            string
		SelfLink        string
		I               int
		A               int
		NullFields      []string
		ForceSendFields []string
	}
	type stB struct {
		Name            string
		SelfLink        string
		I               int
		B               int
		NullFields      []string
		ForceSendFields []string
	}
	type invalid struct {
		I               chan int
		NullFields      []string
		ForceSendFields []string
	}

	type checkSchema interface{ CheckSchema() error }
	for _, tc := range []struct {
		name    string
		res     checkSchema
		wantErr bool
	}{
		{
			name: "valid schema",
			res:  newTestResource[st, stA, stB](nil),
		},
		{
			name:    "invalid schema",
			res:     newTestResource[invalid, stA, stB](nil),
			wantErr: true,
		},
		{
			name:    "invalid schema alpha",
			res:     newTestResource[st, invalid, stB](nil),
			wantErr: true,
		},
		{
			name:    "invalid schema beta",
			res:     newTestResource[st, stA, invalid](nil),
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.res.CheckSchema()
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("CheckSchema() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
		})
	}
}

func TestResourceImpliedVersion(t *testing.T) {
	t.Parallel()

	type st struct {
		I               int
		NullFields      []string
		ForceSendFields []string
	}
	type stA struct {
		I               int
		A               int
		NullFields      []string
		ForceSendFields []string
	}
	type stB struct {
		I               int
		B               int
		NullFields      []string
		ForceSendFields []string
	}

	for _, tc := range []struct {
		name    string
		ga      *st
		alpha   *stA
		beta    *stB
		wantVer meta.Version
		wantErr bool
	}{
		{
			name:    "ver ga",
			ga:      &st{I: 1},
			wantVer: meta.VersionGA,
		},
		{
			name:    "ver alpha",
			alpha:   &stA{I: 1, A: 5},
			wantVer: meta.VersionAlpha,
		},
		{
			name:    "ver beta",
			beta:    &stB{I: 1, B: 7},
			wantVer: meta.VersionBeta,
		},
		{
			name:    "ver alpha",
			ga:      &st{I: 1},
			alpha:   &stA{I: 1, A: 5},
			wantVer: meta.VersionAlpha,
		},
		{
			name:    "ver alpha",
			ga:      &st{I: 1},
			beta:    &stB{I: 1, B: 5},
			wantVer: meta.VersionBeta,
		},
		{
			name:    "ver unknown",
			ga:      &st{I: 1},
			alpha:   &stA{I: 1, A: 5},
			beta:    &stB{I: 1, B: 10},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := newTestResource[st, stA, stB](nil)
			if tc.ga != nil {
				res.Set(tc.ga)
			}
			if tc.alpha != nil {
				res.SetAlpha(tc.alpha)
			}
			if tc.beta != nil {
				res.SetBeta(tc.beta)
			}
			ver, err := res.ImpliedVersion()
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("ImpliedVersion() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
			if err != nil {
				return
			}
			if ver != tc.wantVer {
				t.Errorf("ImpliedVersion() = %v, want %v", ver, tc.wantVer)
			}
		})
	}
}

func TestResourceTypeTrait(t *testing.T) {
	t.Parallel()

	type st struct {
		I int
	}
	type stA struct {
		A int
	}
	type stB struct {
		B int
	}

	tt := TypeTrait[st, stA, stB](&TypeTraitFuncs[st, stA, stB]{
		CopyHelperGAtoAlphaF: func(dest *stA, src *st) error {
			dest.A = src.I + 1
			return nil
		},
		CopyHelperGAtoBetaF: func(dest *stB, src *st) error {
			dest.B = src.I + 2
			return nil
		},
		CopyHelperAlphaToGAF: func(dest *st, src *stA) error {
			dest.I = src.A - 1
			return nil
		},
		CopyHelperAlphaToBetaF: func(dest *stB, src *stA) error {
			dest.B = src.A + 1
			return nil
		},
		CopyHelperBetaToGAF: func(dest *st, src *stB) error {
			dest.I = src.B - 2
			return nil
		},
		CopyHelperBetaToAlphaF: func(dest *stA, src *stB) error {
			dest.A = src.B - 1
			return nil
		},
		FieldTraitsF: func(v meta.Version) *FieldTraits {
			return &FieldTraits{}
		},
	})

	for _, tc := range []struct {
		name  string
		f     func(r Resource[st, stA, stB])
		want  st
		wantA stA
		wantB stB
	}{
		{
			name:  "set field",
			f:     func(r Resource[st, stA, stB]) { r.Access(func(x *st) { x.I = 13 }) },
			want:  st{I: 13},
			wantA: stA{A: 14},
			wantB: stB{B: 15},
		},
		{
			name:  "set field alpha",
			f:     func(r Resource[st, stA, stB]) { r.AccessAlpha(func(x *stA) { x.A = 11 }) },
			want:  st{I: 10},
			wantA: stA{A: 11},
			wantB: stB{B: 12},
		},
		{
			name:  "set field beta",
			f:     func(r Resource[st, stA, stB]) { r.AccessBeta(func(x *stB) { x.B = 12 }) },
			want:  st{I: 10},
			wantA: stA{A: 11},
			wantB: stB{B: 12},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newTestResource(tt)
			tc.f(r)
			g, _ := r.ToGA()
			if diff := cmp.Diff(g, &tc.want); diff != "" {
				t.Errorf("ToGA() -got,+want: %s", diff)
			}
			a, _ := r.ToAlpha()
			if diff := cmp.Diff(a, &tc.wantA); diff != "" {
				t.Errorf("ToAlpha() -got,+want: %s", diff)
			}
			b, _ := r.ToBeta()
			if diff := cmp.Diff(b, &tc.wantB); diff != "" {
				t.Errorf("ToBeta() -got,+want: %s", diff)
			}
		})
	}
}
