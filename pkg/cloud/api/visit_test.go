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

func TestVisit(t *testing.T) {
	trace := map[string]string{}
	recordFn := func(p Path, v reflect.Value) (bool, error) {
		trace[p.String()] = fmt.Sprint(v)
		return true, nil
	}

	type nestedS struct {
		I int
	}

	type testS struct {
		I  int
		M  map[string]string
		N  nestedS
		PI *int
		S  string
		SL []nestedS
	}

	s := testS{
		I:  13,
		M:  map[string]string{"a": "b"},
		N:  nestedS{I: 15},
		PI: new(int),
		S:  "xyz",
		SL: []nestedS{{I: 20}, {I: 21}},
	}

	err := visit(reflect.ValueOf(s), acceptorFromFunc(recordFn))
	if err != nil {
		t.Fatalf("visit() = %v, want nil", err)
	}

	const ignoreValue = "ignore-value"
	want := map[string]string{
		"":        ignoreValue,
		".I":      "13",
		".M":      ignoreValue,
		".M:a":    "b",
		".N":      "{15}",
		".N.I":    "15",
		".PI":     ignoreValue,
		".PI*":    "0",
		".S":      "xyz",
		".SL!0":   "{20}",
		".SL!0.I": "20",
		".SL!1":   "{21}",
		".SL!1.I": "21",
		".SL":     "[{20} {21}]",
	}

	// overwrite the ignored values
	for k, v := range want {
		if v == ignoreValue {
			if _, ok := trace[k]; !ok {
				t.Errorf("trace[%q] doesn't exist", k)
			}
			trace[k] = ignoreValue
		}
	}
	if diff := cmp.Diff(trace, want); diff != "" {
		t.Errorf("visit trace diff: -trace,+want: %s", diff)
	}
}
