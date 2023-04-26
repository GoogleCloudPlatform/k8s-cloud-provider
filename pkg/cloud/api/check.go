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
)

// checkPostAccess validates the fields for consistency. See the error messages
// below for the properties being checked.
func checkPostAccess(traits *FieldTraits, v reflect.Value) error {
	acc := newAcceptorFuncs()
	acc.onStructF = func(p Path, v reflect.Value) (bool, error) {
		var nullFields, forceSendFields []string
		if t, ok := v.Type().FieldByName("NullFields"); ok && t.Type.Kind() == reflect.Slice && t.Type.Elem().Kind() == reflect.String {
			nullFields = v.FieldByName("NullFields").Interface().([]string)
		}
		if t, ok := v.Type().FieldByName("ForceSendFields"); ok && t.Type.Kind() == reflect.Slice && t.Type.Elem().Kind() == reflect.String {
			forceSendFields = v.FieldByName("ForceSendFields").Interface().([]string)
		}
		inNull := func(f string) bool {
			for _, x := range nullFields {
				if f == x {
					return true
				}
			}
			return false
		}
		inForceSend := func(f string) bool {
			for _, x := range forceSendFields {
				if f == x {
					return true
				}
			}
			return false
		}
		for i := 0; i < v.NumField(); i++ {
			ft := v.Type().Field(i)
			if ft.Name == "NullFields" || ft.Name == "ForceSendFields" {
				continue
			}
			fType := traits.fieldType(p.Field(ft.Name))
			fv := v.Field(i)
			fp := p.Field(ft.Name)

			switch fType {
			case FieldTypeSystem:
				if !fv.IsZero() {
					return false, fmt.Errorf("%s has a non-zero value (%v) but is a System field", fv.Interface(), fp)
				}
			case FieldTypeOutputOnly:
				if !fv.IsZero() {
					return false, fmt.Errorf("%s has a non-zero value (%v) but is an OutputOnly field", fv.Interface(), fp)
				}
			case FieldTypeOrdinary:
				switch {
				case fv.IsZero() && !inNull(ft.Name) && !inForceSend(ft.Name):
					return false, fmt.Errorf("%s is zero value but not in a NullFields or ForceSendFields %v %t", fp, fv.Interface(), fv.IsZero())
				case !fv.IsZero() && inNull(ft.Name):
					return false, fmt.Errorf("%s is non-nil and also in NullFields", fp)
				}
			case FieldTypeAllowZeroValue:
				continue
			default:
				return false, fmt.Errorf("invalid FieldType: %q", fType)
			}
		}
		return true, nil
	}
	return visit(v, acc)
}
