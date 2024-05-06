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

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
)

// TypeTrait allows for specialization of the behavior for operations involving
// resources.
type TypeTrait[GA any, Alpha any, Beta any] interface {
	// CopyHelpers are hooks called after the generic copy operation is
	// complete. The func is given the post-copy src and dest struct and is free
	// to modify dest to complete the copy operation.
	CopyHelperGAtoAlpha(dest *Alpha, src *GA) error
	CopyHelperGAtoBeta(dest *Beta, src *GA) error
	CopyHelperAlphaToGA(dest *GA, src *Alpha) error
	CopyHelperAlphaToBeta(dest *Beta, src *Alpha) error
	CopyHelperBetaToGA(dest *GA, src *Beta) error
	CopyHelperBetaToAlpha(dest *Alpha, src *Beta) error

	// FieldTraits returns the field traits for the version given.
	FieldTraits(meta.Version) *FieldTraits
}

// BaseTypeTrait is a TypeTrait that has no effect. This can be embedded to
// reduce verbosity when creating a custom TypeTrait.
type BaseTypeTrait[GA any, Alpha any, Beta any] struct{}

// Implements TypeTrait.
func (*BaseTypeTrait[GA, Alpha, Beta]) CopyHelperGAtoAlpha(dest *Alpha, src *GA) error { return nil }
func (*BaseTypeTrait[GA, Alpha, Beta]) CopyHelperGAtoBeta(dest *Beta, src *GA) error   { return nil }
func (*BaseTypeTrait[GA, Alpha, Beta]) CopyHelperAlphaToGA(dest *GA, src *Alpha) error { return nil }
func (*BaseTypeTrait[GA, Alpha, Beta]) CopyHelperAlphaToBeta(dest *Beta, src *Alpha) error {
	return nil
}
func (*BaseTypeTrait[GA, Alpha, Beta]) CopyHelperBetaToGA(dest *GA, src *Beta) error { return nil }
func (*BaseTypeTrait[GA, Alpha, Beta]) CopyHelperBetaToAlpha(dest *Alpha, src *Beta) error {
	return nil
}
func (*BaseTypeTrait[GA, Alpha, Beta]) FieldTraits(meta.Version) *FieldTraits { return &FieldTraits{} }

// NewFieldTraits creates a default traits.
func NewFieldTraits() *FieldTraits {
	return &FieldTraits{
		fields: []fieldTrait{
			{
				path:  Path{}.Pointer().Field("ServerResponse"),
				fType: FieldTypeSystem,
			},
		},
	}
}

// TypeTraitFuncs is a TypeTrait that takes func instead of defining an interface.
type TypeTraitFuncs[GA any, Alpha any, Beta any] struct {
	CopyHelperGAtoAlphaF   func(dest *Alpha, src *GA) error
	CopyHelperGAtoBetaF    func(dest *Beta, src *GA) error
	CopyHelperAlphaToGAF   func(dest *GA, src *Alpha) error
	CopyHelperAlphaToBetaF func(dest *Beta, src *Alpha) error
	CopyHelperBetaToGAF    func(dest *GA, src *Beta) error
	CopyHelperBetaToAlphaF func(dest *Alpha, src *Beta) error
	FieldTraitsF           func(meta.Version) *FieldTraits
}

// Implements TypeTrait.
func (f *TypeTraitFuncs[GA, Alpha, Beta]) CopyHelperGAtoAlpha(dest *Alpha, src *GA) error {
	if f.CopyHelperGAtoAlphaF == nil {
		return nil
	}
	return f.CopyHelperGAtoAlphaF(dest, src)
}
func (f *TypeTraitFuncs[GA, Alpha, Beta]) CopyHelperGAtoBeta(dest *Beta, src *GA) error {
	if f.CopyHelperGAtoBetaF == nil {
		return nil
	}
	return f.CopyHelperGAtoBetaF(dest, src)
}
func (f *TypeTraitFuncs[GA, Alpha, Beta]) CopyHelperAlphaToGA(dest *GA, src *Alpha) error {
	if f.CopyHelperAlphaToGAF == nil {
		return nil
	}
	return f.CopyHelperAlphaToGAF(dest, src)
}
func (f *TypeTraitFuncs[GA, Alpha, Beta]) CopyHelperAlphaToBeta(dest *Beta, src *Alpha) error {
	if f.CopyHelperAlphaToBetaF == nil {
		return nil
	}
	return f.CopyHelperAlphaToBetaF(dest, src)
}
func (f *TypeTraitFuncs[GA, Alpha, Beta]) CopyHelperBetaToGA(dest *GA, src *Beta) error {
	if f.CopyHelperBetaToGAF == nil {
		return nil
	}
	return f.CopyHelperBetaToGAF(dest, src)
}
func (f *TypeTraitFuncs[GA, Alpha, Beta]) CopyHelperBetaToAlpha(dest *Alpha, src *Beta) error {
	if f.CopyHelperBetaToAlphaF == nil {
		return nil
	}
	return f.CopyHelperBetaToAlphaF(dest, src)
}
func (f *TypeTraitFuncs[GA, Alpha, Beta]) FieldTraits(v meta.Version) *FieldTraits {
	if f.FieldTraitsF == nil {
		return &FieldTraits{}
	}
	return f.FieldTraitsF(v)
}

// FieldTraits are the features and behavior for fields in the resource.
type FieldTraits struct {
	fields []fieldTrait
}

type fieldTrait struct {
	path  Path
	fType FieldType
}

// FieldType of the field.
type FieldType string

const (
	// FieldTypeOrdinary is a ordinary field. It will be compared by value in a
	// diff. It can be zero-value without being in a metafield.
	FieldTypeOrdinary FieldType = "Ordinary"
	// FieldTypeSystem fields are internal infrastructure related fields. These are never
	// copied or diff'd.
	FieldTypeSystem FieldType = "System"
	// FieldTypeOutputOnly are fields that are status set by the server. These
	// should never be set by the client.
	FieldTypeOutputOnly FieldType = "OutputOnly"
	// FieldTypeAllowZeroValue is an ordinary field that can be zero-value
	// without being in a metafield. This is used for testing. TODO(kl52752)
	// remove this field when all resources are migrating to
	// FieldTypeNonZeroValue.
	FieldTypeAllowZeroValue FieldType = "AllowZeroValue"
	// FieldTypeNonZeroValue is a field that's value must be non-zero or
	// specified in a meta-field. It will be compared by value in a diff.
	FieldTypeNonZeroValue FieldType = "NonZeroValue"
)

// CheckSchema validates that the traits are valid and match the schema of the
// given type.
func (dt *FieldTraits) CheckSchema(t reflect.Type) error {
	for _, f := range dt.fields {
		if f.path[len(f.path)-1][0] != pathField {
			return fmt.Errorf("CheckSchema: path %s is not a field reference", f.path)
		}
		_, err := f.path.ResolveType(t)
		if err != nil {
			return fmt.Errorf("CheckSchema: %w", err)
		}
	}
	return nil
}

func (dt *FieldTraits) add(p Path, t FieldType) {
	dt.fields = append(dt.fields, fieldTrait{path: p, fType: t})
}

// OutputOnly specifies the type of the given path.
func (dt *FieldTraits) OutputOnly(p Path) { dt.add(p, FieldTypeOutputOnly) }

// System specifies the type of the given path.
func (dt *FieldTraits) System(p Path) { dt.add(p, FieldTypeSystem) }

// AllowZeroValue specifies the type of the given path.
func (dt *FieldTraits) AllowZeroValue(p Path) { dt.add(p, FieldTypeAllowZeroValue) }

// NonZeroValue specifies the type of the given path.
func (dt *FieldTraits) NonZeroValue(p Path) { dt.add(p, FieldTypeNonZeroValue) }

// Clone create an exact copy of the traits.
func (dt *FieldTraits) Clone() *FieldTraits {
	return &FieldTraits{
		fields: append([]fieldTrait{}, dt.fields...),
	}
}

func (dt *FieldTraits) fieldType(p Path) FieldType { return dt.fieldTrait(p).fType }

func (dt *FieldTraits) fieldTrait(p Path) fieldTrait {
	// TODO(bowei): this can be made very efficient with a tree, early bailout
	// etc.. We will go with a very inefficient implimentation for now.
	for _, f := range dt.fields {
		if p.HasPrefix(f.path) {
			return f
		}
	}
	return fieldTrait{
		path:  p,
		fType: FieldTypeOrdinary,
	}
}
