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

const (
	nullFieldsName      = "NullFields"
	forceSendFieldsName = "ForceSendFields"
)

func newMetafieldAccessor(v reflect.Value) (*metafieldAccessor, error) {
	if v.Type().Kind() != reflect.Struct {
		return nil, fmt.Errorf("invalid type: %s", v.Type())
	}
	ret := &metafieldAccessor{}
	if t, ok := v.Type().FieldByName(nullFieldsName); ok && t.Type.Kind() == reflect.Slice && t.Type.Elem().Kind() == reflect.String {
		ret.nullFields = v.FieldByName(nullFieldsName)
	}
	if t, ok := v.Type().FieldByName(forceSendFieldsName); ok && t.Type.Kind() == reflect.Slice && t.Type.Elem().Kind() == reflect.String {
		ret.forceSendFields = v.FieldByName(forceSendFieldsName)
	}
	if !ret.nullFields.IsValid() || !ret.forceSendFields.IsValid() {
		return nil, fmt.Errorf("struct does not have NullField or ForceSendFields")
	}
	return ret, nil
}

type metafieldAccessor struct {
	nullFields      reflect.Value
	forceSendFields reflect.Value
}

func (a *metafieldAccessor) null() map[string]bool {
	ret := map[string]bool{}
	for _, fn := range a.nullFields.Interface().([]string) {
		ret[fn] = true
	}
	return ret
}
func (a *metafieldAccessor) forceSend() map[string]bool {
	ret := map[string]bool{}
	for _, fn := range a.forceSendFields.Interface().([]string) {
		ret[fn] = true
	}
	return ret
}

func (a *metafieldAccessor) inNull(f string) bool {
	for _, x := range a.nullFields.Interface().([]string) {
		if f == x {
			return true
		}
	}
	return false
}

func (a *metafieldAccessor) inForceSend(f string) bool {
	for _, x := range a.forceSendFields.Interface().([]string) {
		if f == x {
			return true
		}
	}
	return false
}
