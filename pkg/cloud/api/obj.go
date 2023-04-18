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

// ConversionContext gives which version => version the error occurred on.
type ConversionContext int

const (
	GAToAlphaConversion ConversionContext = iota
	GAToBetaConversion
	AlphaToGAConversion
	AlphaToBetaConversion
	BetaToGAConversion
	BetaToAlphaConversion
	conversionContextCount
)

// ConversionError is returned from To*() methods. Inspect this error to get
// more details on what did not convert.
type ConversionError struct {
	// MissingFields is a list of field values that were set but did not
	// translate to the version requested.
	MissingFields []MissingField
}

func (e *ConversionError) hasErr() bool {
	return len(e.MissingFields) > 0
}

func (e *ConversionError) Error() string {
	return fmt.Sprintf("ConversionError: missing fields %v", e.MissingFields)
}

type MissingField struct {
	Context ConversionContext
	Path    Path
	Value   any
}

type conversionErrors struct {
	missingFields []missingFieldOnCopy
}

// VersionedObject wraps the standard GA, Alpha, Beta versions of a GCP
// resource. By accessing the object using Access(), AccessAlpha().
// AccessBeta(), VersionedObject will ensure that common fields between the
// versions of the object are in sync.
type VersionedObject[GA any, Alpha any, Beta any] struct {
	copierOptions []copierOption

	ga    GA
	alpha Alpha
	beta  Beta

	errors [conversionContextCount]conversionErrors
}

// cycleCheck that there are no cycles where a struct type appears 2+ times on
// the same path. Our algorithms requires special handling for recursive
// structures.
func cycleCheck(p Path, t reflect.Type, seen []string) error {
	switch t.Kind() {
	case reflect.Slice:
		return cycleCheck(p.Index(0), t.Elem(), seen)
	case reflect.Pointer:
		return cycleCheck(p.Pointer(), t.Elem(), seen)
	case reflect.Map:
		// Use "x" as the placeholder for the map key in the Path for debugging
		// output purposes.
		return cycleCheck(p.MapIndex("x"), t.Elem(), seen)
	case reflect.Struct:
		typeName := fmt.Sprintf("%s/%s", t.PkgPath(), t.Name())
		for _, seenTypeName := range seen {
			if typeName == seenTypeName {
				return fmt.Errorf("recursive type found at %s: %s", p, typeName)
			}
		}
		// Add this struct type to the list of types seen on this path.
		seen = append(seen, fmt.Sprintf("%s/%s", t.PkgPath(), t.Name()))
		for i := 0; i < t.NumField(); i++ {
			if err := cycleCheck(p.Field(t.Field(i).Name), t.Field(i).Type, seen); err != nil {
				return err
			}
		}
	}
	return nil
}

// typeCheck the type is something we can handle.
func typeCheck(p Path, t reflect.Type) error {
	// valid_type => basic | ...
	if isBasicT(t) {
		return nil
	}
	switch t.Kind() {
	case reflect.Pointer:
		if err := typeCheck(p, t.Elem()); err != nil {
			return err
		}
	case reflect.Struct:
		// struct => {all fields are valid_type}
		for i := 0; i < t.NumField(); i++ {
			tf := t.Field(i)
			if err := typeCheck(p.Field(tf.Name), tf.Type); err != nil {
				return err
			}
		}
	case reflect.Slice:
		// slice => {elements => valid_type}
		if err := typeCheck(p.Index(0), t.Elem()); err != nil {
			return err
		}
	case reflect.Map:
		// map => key is basic type; value is valid_type
		if !isBasicT(t.Key()) {
			return fmt.Errorf("map key must be basic type %s: %v", p.Pointer(), t)
		}
		// Supported value types.
		if !isBasicT(t.Elem()) {
			switch t.Elem().Kind() {
			case reflect.Slice, reflect.Struct:
			default:
				return fmt.Errorf("unsupported value type %s: %v", p, t)
			}
			// Use "x" as the placeholder for the map key in the Path for debugging
			// output purposes.
			if err := typeCheck(p.MapIndex("x"), t.Elem()); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unsupported type %s: %v", p, t)
	}
	return nil
}

func versionedObjectCheck(t reflect.Type) error {
	// Run cycleCheck first, other checks will blow up if there are cycles.
	if err := cycleCheck(Path{}, t, []string{}); err != nil {
		return err
	}
	if err := typeCheck(Path{}, t); err != nil {
		return err
	}
	return nil
}

// CheckSchema should be called in init() to ensure that the resource being
// wrapped by VersionedObject meets the assumptions we are making for this the
// transformations to work.
func (u *VersionedObject[GA, Alpha, Beta]) CheckSchema() error {
	err := versionedObjectCheck(reflect.TypeOf(u.ga))
	if err != nil {
		return err
	}
	err = versionedObjectCheck(reflect.TypeOf(u.alpha))
	if err != nil {
		return err
	}
	err = versionedObjectCheck(reflect.TypeOf(u.beta))
	if err != nil {
		return err
	}
	return nil
}

// Access the mutable object.
func (u *VersionedObject[GA, Alpha, Beta]) Access(f func(x *GA)) error {
	f(&u.ga)

	src := reflect.ValueOf(&u.ga)

	c := newCopier(u.copierOptions...)
	err := c.do(reflect.ValueOf(&u.alpha), src)
	if err != nil {
		return err
	}
	u.errors[GAToAlphaConversion].missingFields = c.missing

	c = newCopier(u.copierOptions...)
	err = c.do(reflect.ValueOf(&u.beta), src)
	if err != nil {
		return err
	}
	u.errors[GAToBetaConversion].missingFields = c.missing

	return nil
}

// AccessAlpha object.
func (u *VersionedObject[GA, Alpha, Beta]) AccessAlpha(f func(x *Alpha)) error {
	f(&u.alpha)
	src := reflect.ValueOf(&u.alpha)

	c := newCopier(u.copierOptions...)
	err := c.do(reflect.ValueOf(&u.ga), src)
	if err != nil {
		return err
	}
	u.errors[AlphaToGAConversion].missingFields = c.missing

	c = newCopier(u.copierOptions...)
	err = c.do(reflect.ValueOf(&u.beta), src)
	if err != nil {
		return err
	}
	u.errors[AlphaToBetaConversion].missingFields = c.missing

	return nil
}

// AccessBeta object.
func (u *VersionedObject[GA, Alpha, Beta]) AccessBeta(f func(x *Beta)) error {
	f(&u.beta)
	src := reflect.ValueOf(&u.beta)

	c := newCopier(u.copierOptions...)
	err := c.do(reflect.ValueOf(&u.ga), src)
	if err != nil {
		return err
	}
	u.errors[BetaToGAConversion].missingFields = c.missing

	c = newCopier(u.copierOptions...)
	err = c.do(reflect.ValueOf(&u.alpha), src)
	if err != nil {
		return err
	}
	u.errors[BetaToAlphaConversion].missingFields = c.missing

	return nil
}

// ToGA returns the GA version of this object. Use error.As ConversionError to
// get the specific details.
func (u *VersionedObject[GA, Alpha, Beta]) ToGA() (*GA, error) {
	var errs ConversionError
	for _, cc := range []ConversionContext{AlphaToGAConversion, BetaToGAConversion} {
		for _, mf := range u.errors[cc].missingFields {
			errs.MissingFields = append(errs.MissingFields, MissingField{
				Context: cc,
				Path:    mf.Path,
				Value:   mf.Value,
			})
		}
	}
	if errs.hasErr() {
		return &u.ga, &errs
	}
	return &u.ga, nil
}

// ToAlpha returns the Alpha version of this object. Use error.As
// ConversionError to get the specific details.
func (u *VersionedObject[GA, Alpha, Beta]) ToAlpha() (*Alpha, error) {
	var errs ConversionError
	for _, cc := range []ConversionContext{GAToAlphaConversion, BetaToAlphaConversion} {
		for _, mf := range u.errors[cc].missingFields {
			errs.MissingFields = append(errs.MissingFields, MissingField{
				Context: cc,
				Path:    mf.Path,
				Value:   mf.Value,
			})
		}
	}
	if errs.hasErr() {
		return &u.alpha, &errs
	}
	return &u.alpha, nil
}

// ToBeta returns the Beta version of this object. Use error.As ConversionError
// to get the specific details.
func (u *VersionedObject[GA, Alpha, Beta]) ToBeta() (*Beta, error) {
	var errs ConversionError
	for _, cc := range []ConversionContext{GAToBetaConversion, AlphaToBetaConversion} {
		for _, mf := range u.errors[cc].missingFields {
			errs.MissingFields = append(errs.MissingFields, MissingField{
				Context: cc,
				Path:    mf.Path,
				Value:   mf.Value,
			})
		}
	}
	if errs.hasErr() {
		return &u.beta, &errs
	}
	return &u.beta, nil
}
