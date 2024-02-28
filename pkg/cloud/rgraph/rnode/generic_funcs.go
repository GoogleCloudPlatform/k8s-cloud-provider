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

package rnode

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"google.golang.org/api/googleapi"
	"k8s.io/klog/v2"
)

// GenericOps are a typed dispatch for (API version, scope) for CRUD verbs. Set
// the field to nil if the operation is not supported.
type GenericOps[GA any, Alpha any, Beta any] interface {
	GetFuncs(gcp cloud.Cloud) *GetFuncs[GA, Alpha, Beta]
	CreateFuncs(gcp cloud.Cloud) *CreateFuncs[GA, Alpha, Beta]
	UpdateFuncs(gcp cloud.Cloud) *UpdateFuncs[GA, Alpha, Beta]
	DeleteFuncs(gcp cloud.Cloud) *DeleteFuncs[GA, Alpha, Beta]
}

// GetFuncsByScope dispatches the operation by the appropriate scope. Set the
// field to nil if the scope is not supported.
type GetFuncsByScope[T any] struct {
	Global   func(context.Context, *meta.Key, ...cloud.Option) (*T, error)
	Regional func(context.Context, *meta.Key, ...cloud.Option) (*T, error)
	Zonal    func(context.Context, *meta.Key, ...cloud.Option) (*T, error)
}

// Do the operation.
func (s *GetFuncsByScope[T]) Do(ctx context.Context, key *meta.Key, options ...cloud.Option) (*T, error) {
	klog.Infof("get %s", key)
	switch {
	case key.Type() == meta.Global && s.Global != nil:
		return s.Global(ctx, key, options...)
	case key.Type() == meta.Regional && s.Regional != nil:
		return s.Regional(ctx, key, options...)
	case key.Type() == meta.Zonal && s.Zonal != nil:
		return s.Zonal(ctx, key, options...)
	}
	return nil, fmt.Errorf("unsupported scope (key=%s)", key)
}

type GetFuncs[GA any, Alpha any, Beta any] struct {
	GA    GetFuncsByScope[GA]
	Alpha GetFuncsByScope[Alpha]
	Beta  GetFuncsByScope[Beta]
}

func (f *GetFuncs[GA, Alpha, Beta]) Do(
	ctx context.Context,
	ver meta.Version,
	id *cloud.ResourceID,
	tt api.TypeTrait[GA, Alpha, Beta],
) (api.Resource[GA, Alpha, Beta], error) {
	current := api.NewResource(id, tt)
	switch ver {
	case meta.VersionGA:
		raw, err := f.GA.Do(ctx, id.Key, cloud.ForceProjectID(id.ProjectID))
		if err != nil {
			return nil, err
		}
		if err := current.Set(raw); err != nil {
			return nil, err
		}
	case meta.VersionAlpha:
		raw, err := f.Alpha.Do(ctx, id.Key, cloud.ForceProjectID(id.ProjectID))
		if err != nil {
			return nil, err
		}
		if err := current.SetAlpha(raw); err != nil {
			return nil, err
		}
	case meta.VersionBeta:
		raw, err := f.Beta.Do(ctx, id.Key, cloud.ForceProjectID(id.ProjectID))
		if err != nil {
			return nil, err
		}
		if err := current.SetBeta(raw); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("getFuncs.do unsupported version %q", ver)
	}
	return current.Freeze()
}

type CreateFuncsByScope[T any] struct {
	Global   func(context.Context, *meta.Key, *T, ...cloud.Option) error
	Regional func(context.Context, *meta.Key, *T, ...cloud.Option) error
	Zonal    func(context.Context, *meta.Key, *T, ...cloud.Option) error
}

func (s *CreateFuncsByScope[T]) Do(ctx context.Context, key *meta.Key, x *T, options ...cloud.Option) error {
	// TODO: Context logging
	// TODO: span
	switch {
	case key.Type() == meta.Global && s.Global != nil:
		return s.Global(ctx, key, x, options...)
	case key.Type() == meta.Regional && s.Regional != nil:
		return s.Regional(ctx, key, x, options...)
	case key.Type() == meta.Zonal && s.Zonal != nil:
		return s.Zonal(ctx, key, x, options...)
	}
	return fmt.Errorf("unsupported scope (key = %s)", key)
}

type CreateFuncs[GA any, Alpha any, Beta any] struct {
	GA    CreateFuncsByScope[GA]
	Alpha CreateFuncsByScope[Alpha]
	Beta  CreateFuncsByScope[Beta]
}

func (f *CreateFuncs[GA, Alpha, Beta]) Do(
	ctx context.Context,
	id *cloud.ResourceID,
	r api.Resource[GA, Alpha, Beta],
) error {
	// TODO: Context logging
	// TODO: span
	switch r.Version() {
	case meta.VersionGA:
		raw, err := r.ToGA()
		if err != nil {
			return err
		}
		err = f.GA.Do(ctx, id.Key, raw, cloud.ForceProjectID(id.ProjectID))
		if err != nil {
			return err
		}
		return nil
	case meta.VersionAlpha:
		raw, err := r.ToAlpha()
		if err != nil {
			return err
		}
		err = f.Alpha.Do(ctx, id.Key, raw, cloud.ForceProjectID(id.ProjectID))
		if err != nil {
			return err
		}
		return nil
	case meta.VersionBeta:
		raw, err := r.ToBeta()
		if err != nil {
			return err
		}
		err = f.Beta.Do(ctx, id.Key, raw, cloud.ForceProjectID(id.ProjectID))
		if err != nil {
			return err
		}
		return nil
	}
	return fmt.Errorf("createFuncs.do unsupported version %q", r.Version())
}

type UpdateFuncsByScope[T any] struct {
	Global   func(context.Context, *meta.Key, *T, ...cloud.Option) error
	Regional func(context.Context, *meta.Key, *T, ...cloud.Option) error
	Zonal    func(context.Context, *meta.Key, *T, ...cloud.Option) error
}

func (s *UpdateFuncsByScope[T]) Do(ctx context.Context, key *meta.Key, x *T, options ...cloud.Option) error {
	switch {
	case key.Type() == meta.Global && s.Global != nil:
		return s.Global(ctx, key, x, options...)
	case key.Type() == meta.Regional && s.Regional != nil:
		return s.Regional(ctx, key, x, options...)
	case key.Type() == meta.Zonal && s.Zonal != nil:
		return s.Zonal(ctx, key, x, options...)
	}
	return fmt.Errorf("unsupported scope (key = %s)", key)
}

const (
	// Resource does not have a .Fingerprint field. Note: this
	// means that the resource is technically not compliant with
	// API conventions but these exceptions occur throughout the
	// GCE APIs and we have to work around them.
	UpdateFuncsNoFingerprint = 1 << iota
)

type UpdateFuncs[GA any, Alpha any, Beta any] struct {
	GA    UpdateFuncsByScope[GA]
	Alpha UpdateFuncsByScope[Alpha]
	Beta  UpdateFuncsByScope[Beta]

	Options int
}

func fingerprintField(v reflect.Value) (reflect.Value, error) {
	typeCheck := func(v reflect.Value) error {
		t := v.Type()
		if !(t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.Struct) {
			return fmt.Errorf("fingerprintField: invalid type %T", v.Interface())
		}
		if fv := v.Elem().FieldByName("Fingerprint"); !fv.IsValid() || fv.Kind() != reflect.String {
			return fmt.Errorf("fingerprintField: no Fingerprint field (%T)", v.Interface())
		}
		return nil
	}
	if err := typeCheck(v); err != nil {
		return reflect.Value{}, err
	}
	return v.Elem().FieldByName("Fingerprint"), nil
}

func (f *UpdateFuncs[GA, Alpha, Beta]) Do(
	ctx context.Context,
	fingerprint string,
	id *cloud.ResourceID,
	desired api.Resource[GA, Alpha, Beta],
) error {
	// TODO: Context logging
	// TODO: span
	switch desired.Version() {
	case meta.VersionGA:
		raw, err := desired.ToGA()
		if err != nil {
			return err
		}
		if f.Options&UpdateFuncsNoFingerprint == 0 {
			// TODO: we need to make sure this is the right way to do this as it
			// modifies the Resource. Patch fingerprint for the update.
			if fv, err := fingerprintField(reflect.ValueOf(raw)); err != nil {
				return err
			} else {
				klog.Infof("Set fingerprint to %s", fingerprint)
				fv.Set(reflect.ValueOf(fingerprint))
			}
		}
		err = f.GA.Do(ctx, id.Key, raw, cloud.ForceProjectID(id.ProjectID))
		if err != nil {
			return err
		}
		return nil

	case meta.VersionAlpha:
		raw, err := desired.ToAlpha()
		if err != nil {
			return err
		}
		if f.Options&UpdateFuncsNoFingerprint == 0 {
			if fv, err := fingerprintField(reflect.ValueOf(raw)); err != nil {
				return err
			} else {
				fv.Set(reflect.ValueOf(fingerprint))
			}
		}
		err = f.Alpha.Do(ctx, id.Key, raw, cloud.ForceProjectID(id.ProjectID))
		if err != nil {
			return err
		}
		return nil

	case meta.VersionBeta:
		raw, err := desired.ToBeta()
		if err != nil {
			return err
		}
		if f.Options&UpdateFuncsNoFingerprint == 0 {
			if fv, err := fingerprintField(reflect.ValueOf(raw)); err != nil {
				return err
			} else {
				fv.Set(reflect.ValueOf(fingerprint))
			}
		}
		err = f.Beta.Do(ctx, id.Key, raw, cloud.ForceProjectID(id.ProjectID))
		if err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("updateFuncs.do unsupported version %q", desired.Version())
}

type DeleteFuncsByScope[T any] struct {
	Global   func(context.Context, *meta.Key, ...cloud.Option) error
	Regional func(context.Context, *meta.Key, ...cloud.Option) error
	Zonal    func(context.Context, *meta.Key, ...cloud.Option) error
}

func (s *DeleteFuncsByScope[T]) Do(ctx context.Context, id *cloud.ResourceID, options ...cloud.Option) error {
	key := id.Key
	switch {
	case key.Type() == meta.Global && s.Global != nil:
		return s.Global(ctx, key, options...)
	case key.Type() == meta.Regional && s.Regional != nil:
		return s.Regional(ctx, key, options...)
	case key.Type() == meta.Zonal && s.Zonal != nil:
		return s.Zonal(ctx, key, options...)
	}
	return fmt.Errorf("unsupported scope (key = %s)", key)
}

type DeleteFuncs[GA any, Alpha any, Beta any] struct {
	GA    DeleteFuncsByScope[GA]
	Alpha DeleteFuncsByScope[Alpha]
	Beta  DeleteFuncsByScope[Beta]
}

func (f *DeleteFuncs[GA, Alpha, Beta]) Do(ctx context.Context, id *cloud.ResourceID) error {
	// TODO: Context logging
	// TODO: span
	return f.GA.Do(ctx, id, cloud.ForceProjectID(id.ProjectID))
}

func isErrorCode(err error, code int) bool {
	var gerr *googleapi.Error
	if !errors.As(err, &gerr) {
		return false
	}
	return gerr.Code == code
}

func isErrorNotFound(err error) bool { return isErrorCode(err, 404) }

func GenericGet[GA any, Alpha any, Beta any](
	ctx context.Context,
	gcp cloud.Cloud,
	resourceName string,
	ops GenericOps[GA, Alpha, Beta],
	typeTrait api.TypeTrait[GA, Alpha, Beta],
	b Builder,
) error {
	// TODO: this method needs some audits, it may be incomplete.

	if b.Version() == "" {
		// TODO: handle this by returning an error.
		panic("XXX")
	}
	r, err := ops.GetFuncs(gcp).Do(ctx, b.Version(), b.ID(), typeTrait)

	switch {
	case isErrorNotFound(err):
		b.SetState(NodeDoesNotExist)
		return nil // Not found is not an error condition.

	case err != nil:
		b.SetState(NodeStateError)
		return fmt.Errorf("genericGet %s: %w", resourceName, err)

	default:
		b.SetState(NodeExists)
		b.SetResource(r)
		return nil
	}
}
