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

	"github.com/google/go-cmp/cmp"
)

func TestFill(t *testing.T) {
	type inner struct {
		I int
	}

	type st struct {
		I    int
		S    string
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
	}

	var s st
	err := Fill(&s)
	if err != nil {
		t.Fatalf("Fill() = %v, want nil", err)
	}

	zzzStr := "ZZZ"

	want := st{
		I:    111,
		S:    "ZZZ",
		PS:   &zzzStr,
		IS:   inner{I: 111},
		PIS:  &inner{I: 111},
		LS:   []string{"ZZZ"},
		LSt:  []inner{{I: 111}},
		LPS:  []*string{&zzzStr},
		LPSt: []*inner{&inner{I: 111}},
		M:    map[string]inner{"ZZZ": {I: 111}},
		MP:   map[string]*inner{"ZZZ": &inner{I: 111}},
		MLS:  map[string][]string{"ZZZ": {"ZZZ"}},
	}
	if diff := cmp.Diff(s, want); diff != "" {
		t.Errorf("Fill(); -got,+want = %s", diff)
	}
}
