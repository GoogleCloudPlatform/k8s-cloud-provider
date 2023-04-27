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
	"sort"
)

type missingFieldOnCopy struct {
	Path  Path
	Value any
}

// copierOption are options that customize the behavior of the internal copier.
type copierOption func(*copier)

// copierLogS configures logging for the copier to use the given func f.
func copierLogS(f func(msg string, kv ...any)) copierOption {
	return func(c *copier) { c.logSFn = f }
}

func newCopier(opts ...copierOption) *copier {
	c := &copier{}
	for _, o := range opts {
		o(c)
	}
	return c
}

type copier struct {
	// logSFn is an optional structured log function, matching the
	// signature from klog/v2.
	logSFn func(msg string, kv ...any)

	missing []missingFieldOnCopy
}

func (c *copier) logS(msg string, kv ...any) {
	if c.logSFn == nil {
		return
	}
	c.logSFn(msg, kv...)
}

func (c *copier) do(dest, src reflect.Value) error {
	return c.doValues(Path{}, dest, src)
}

func (c *copier) doValues(p Path, dest, src reflect.Value) error {
	switch {
	case isBasicV(dest) && isBasicV(src):
		return c.doBasic(p, dest, src)
	case src.Type().Kind() == reflect.Pointer && dest.Type().Kind() == reflect.Pointer:
		return c.doPointer(p, dest, src)
	case src.Type().Kind() == reflect.Slice && dest.Type().Kind() == reflect.Slice:
		return c.doSlice(p, dest, src)
	case src.Type().Kind() == reflect.Struct && dest.Type().Kind() == reflect.Struct:
		return c.doStruct(p, dest, src)
	case src.Type().Kind() == reflect.Map && dest.Type().Kind() == reflect.Map:
		return c.doMap(p, dest, src)
	}
	return fmt.Errorf("copyValues: incompatible types: src %T, dest %T", src.Interface(), dest.Interface())
}

func (c *copier) doBasic(p Path, dest, src reflect.Value) error {
	if !isBasicV(dest) || !isBasicV(src) || dest.Type().Kind() != src.Type().Kind() {
		return fmt.Errorf("copyBasic: mismatched types: src %s, dest %s", src.Type(), dest.Type())
	}
	if !dest.CanSet() {
		return fmt.Errorf("cannot set dest (%s)", p)
	}
	dest.Set(src)
	c.logS("copyBasic", "path", p, "value", dest.Interface())
	return nil
}

func (c *copier) doPointer(p Path, dest, src reflect.Value) error {
	if dest.Type().Kind() != reflect.Pointer || src.Type().Kind() != reflect.Pointer {
		return fmt.Errorf("copyPointer: invalid types: src %T, dest %T", src.Interface(), dest.Interface())
	}
	if src.IsZero() {
		c.logS("copyPointer zero", "path", p)
		if !dest.CanSet() {
			return fmt.Errorf("cannot set dest (%s)", p)
		}
		dest.Set(reflect.Zero(dest.Type()))
		return nil
	}
	if dest.IsZero() {
		if !dest.CanAddr() {
			return fmt.Errorf("copyPointer: dest is nil and not addressable: src %T, dest %T", src.Interface(), dest.Interface())
		}
		dest.Set(reflect.New(dest.Type().Elem()))
		c.logS("copyPointer", "path", p, "value", dest.Interface())
	}
	return c.doValues(p.Pointer(), dest.Elem(), src.Elem())
}

func (c *copier) doSlice(p Path, dest, src reflect.Value) error {
	if dest.Type().Kind() != reflect.Slice || src.Type().Kind() != reflect.Slice {
		return fmt.Errorf("copySlice: invalid type (dest: %T, src: %T)", dest.Interface(), src.Interface())
	}
	if src.IsZero() {
		dest.Set(reflect.Zero(dest.Type()))
		c.logS("copySlice zero", "path", p)
		return nil
	}

	newSlice := reflect.MakeSlice(dest.Type(), src.Len(), src.Len())
	for i := 0; i < src.Len(); i++ {
		if err := c.doValues(p.Index(i), newSlice.Index(i), src.Index(i)); err != nil {
			return err
		}
	}
	c.logS("copySlice", "path", p, "value", newSlice)

	if !dest.CanSet() {
		return fmt.Errorf("cannot set dest (%s)", p)
	}
	dest.Set(newSlice)

	return nil
}

func (c *copier) doStruct(p Path, dest, src reflect.Value) error {
	if dest.Kind() != reflect.Struct || src.Kind() != reflect.Struct {
		return fmt.Errorf("copyStruct: invalid type (dest: %T, src: %T)", dest.Interface(), src.Interface())
	}
	// Copy over fields that are present in both src and dest. Fields in dest
	// that don't exist in src are left alone.
	for i := 0; i < src.Type().NumField(); i++ {
		srcFieldT := src.Type().Field(i)
		fieldName := srcFieldT.Name
		destField := dest.FieldByName(fieldName)
		_, ok := dest.Type().FieldByName(fieldName)

		if !ok {
			// Only non-zero fields are counted towards
			// the missing fields. Fields explicitly named
			// in NullFields or ForceSendFields are
			// handled by copyMetaFields() below.
			if !src.Field(i).IsZero() {
				c.missing = append(c.missing, missingFieldOnCopy{
					Path:  p.Field(fieldName),
					Value: src.Field(i).Interface(),
				})
				c.logS("copyStruct missing field", "path", p, "fieldName", fieldName)
			}
			continue
		}

		// ServerResponse should be skipped.
		if (p.Equal(Path{}) || p.Equal(Path{}.Pointer())) && fieldName == "ServerResponse" {
			continue
		}

		if fieldName == "NullFields" || fieldName == "ForceSendFields" {
			err := c.doMetaFields(p.Field(fieldName), destField, src.Field(i), dest, src)
			if err != nil {
				return err
			}
			continue
		}

		c.logS("copyStruct", "path", p, "fieldName", fieldName)
		if err := c.doValues(p.Field(fieldName), destField, src.Field(i)); err != nil {
			return err
		}
	}
	return nil
}

func (c *copier) doMap(p Path, dest, src reflect.Value) error {
	if dest.Type().Kind() != reflect.Map || src.Type().Kind() != reflect.Map {
		return fmt.Errorf("copyMap: invalid type (dest: %T, src: %T)", dest.Interface(), src.Interface())
	}

	if src.IsZero() {
		if !dest.CanSet() {
			return fmt.Errorf("cannot set dest (%s)", p)
		}
		dest.Set(reflect.Zero(dest.Type()))
		c.logS("copyMap zero", "path", p)
		return nil
	}

	dkt := dest.Type().Key()
	dvt := dest.Type().Elem()
	skt := src.Type().Key()
	svt := src.Type().Elem()

	if !basicT(dkt) || !basicT(skt) {
		return fmt.Errorf("copyMap: keys are not basic types (dest: %T, src: %T)", dest.Interface(), src.Interface())
	}
	if dkt.Kind() != skt.Kind() {
		return fmt.Errorf("copyMap: keys do not match (dest: %T, src: %T)", dest.Interface(), src.Interface())
	}
	if dvt.Kind() != svt.Kind() {
		return fmt.Errorf("copyMap: values type must match (dest: %T, src: %T)", dest.Interface(), src.Interface())
	}

	newMap := reflect.MakeMapWithSize(dest.Type(), src.Len())

	for _, sk := range src.MapKeys() {
		sv := src.MapIndex(sk)
		switch {
		case basicT(dvt) && basicT(svt):
			c.logS("copyMap basic", "path", p.MapIndex(sk.Interface()), "value", sv.Interface())
			newMap.SetMapIndex(sk, sv)
		case svt.Kind() == reflect.Struct:
			pdv := reflect.New(dvt)
			if err := c.doValues(p.MapIndex(sk.Interface()), pdv.Elem(), sv); err != nil {
				return err
			}
			newMap.SetMapIndex(sk, pdv.Elem())
		case svt.Kind() == reflect.Slice:
			dv := reflect.New(dvt).Elem()
			dv.Grow(sv.Len())
			if err := c.doValues(p.MapIndex(sk.Interface()), dv, sv); err != nil {
				return err
			}
			newMap.SetMapIndex(sk, dv)
		default:
			return fmt.Errorf("unsupported map types (dest: %T, src: %T)", dest.Interface(), src.Interface())
		}
	}

	if !dest.CanSet() {
		return fmt.Errorf("cannot set dest (%s)", p)
	}
	dest.Set(newMap)

	return nil
}

// copyMetaFields copies over the contents of metafields such as
// "ForceSendFields". This may not be a straightforward copy when
// there are references in version-specific API fields. For example:
//
//	type Obj struct { Field int }
//	type ObjBeta struct { Field int; BetaField int }
//
// The user may do the following:
//
//	EditBeta(func(x *ObjBeta) { x.ForceSendFields = []string{"BetaField"} }
//	Edit(func(x *Obj) { x.ForceSendFields = []string{"Field"} }
//
// We want to preserve the presence of "BetaField" for the beta
// version of Obj, even though the field does not exist in the GA
// version of the API. In the above example, we will end up with:
//
//	// BetaField is marked as "missing" if the user uses ToGA().
//	Obj.ForceSendFields == []string{"Field"}
//	ObjBeta.ForceSendFields == []string{"Field", "BetaField"}
func (c *copier) doMetaFields(p Path, destField, srcField, destStruct, srcStruct reflect.Value) error {
	if !isSliceOfStringV(destField) || !isSliceOfStringV(srcField) {
		return fmt.Errorf("copyMetaFields: invalid type (destField: %T, srcField: %T)", destField.Interface(), srcField.Interface())
	}

	destMetaFields := destField.Interface().([]string)

	// MetaFields already present in the destination list.
	exists := map[string]bool{}
	for _, v := range destMetaFields {
		exists[v] = true
	}

	c.logS("copyMetaFields dest", "path", p, "destFields", exists)

	for _, fn := range srcField.Interface().([]string) {
		if _, ok := srcStruct.Type().FieldByName(fn); !ok {
			return fmt.Errorf("copyMetaFields: %s refers to field %q that doesn't exist (type %T)", p, fn, srcStruct.Interface())
		}
		_, destHasField := destStruct.Type().FieldByName(fn)
		// We only need to add to destMetaFields if it exists
		// in the dest struct and hasn't already been added to
		// the list.
		if destHasField && !exists[fn] {
			destMetaFields = append(destMetaFields, fn)
			c.logS("copyMetaFields add", "path", p, "fieldName", fn)
		} else if !destHasField {
			// Record that the metafield referenced a
			// field that didn't exist on the dest
			// version.
			c.missing = append(c.missing, missingFieldOnCopy{
				Path:  p.Field(fn),
				Value: srcField.Interface(),
			})
			c.logS("copyMetaFields missing field", "path", p, "fieldName", fn)
		}
	}

	if !destField.CanSet() {
		return fmt.Errorf("cannot set destField (%s)", p)

	}

	sort.Strings(destMetaFields)
	destField.Set(reflect.ValueOf(destMetaFields))

	return nil
}
