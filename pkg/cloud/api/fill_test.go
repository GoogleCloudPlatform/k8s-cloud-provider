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

func TestFill(t *testing.T) {
	type inner struct {
		I int
	}

	type st struct {
		I   int
		S   string
		PS  *string
		IS  inner
		PIS *inner
	}

	var s st
	err := Fill(&s)
	if err != nil {
		t.Fatalf("Fill() = %v, want nil", err)
	}

	// Check that there are no zeros.
	switch {
	case s.I == 0:
		fallthrough
	case s.S == "":
		fallthrough
	case s.PS == nil:
		fallthrough
	case *s.PS == "":
		fallthrough
	case s.IS.I == 0:
		fallthrough
	case s.PIS == nil:
		fallthrough
	case s.PIS.I == 0:
		t.Errorf("Fill(), s = %s, had zero fields", pretty.Sprint(s))
	}
}
