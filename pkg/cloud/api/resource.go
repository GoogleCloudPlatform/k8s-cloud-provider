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

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
)

// Resource is read-only view into the resource. A Resource
// has a definitive Version.
type Resource[GA any, Alpha any, Beta any] interface {
	// Version of the resource. This cannot be indeterminant.
	Version() meta.Version
	// ResourceID fully qualitfied name of the resource.
	ResourceID() *cloud.ResourceID

	// Convert to the concrete types.
	ToGA() (*GA, error)
	ToAlpha() (*Alpha, error)
	ToBeta() (*Beta, error)

	// Diff obtains the difference between this resource and
	// other, taking into account the versions of the resources
	// being compared. Cross Alpha and Beta comparisons are not
	// currently supported.
	Diff(other Resource[GA, Alpha, Beta]) (*DiffResult, error)

	// Clone returns an exact structural copy of this resource.
	// Clone() Resource[GA, Alpha, Beta] XXX
}

type resource[GA any, Alpha any, Beta any] struct {
	x   *mutableResource[GA, Alpha, Beta]
	ver meta.Version
}

// Implements Resource.
func (obj *resource[GA, Alpha, Beta]) Version() meta.Version         { return obj.ver }
func (obj *resource[GA, Alpha, Beta]) ResourceID() *cloud.ResourceID { return obj.x.ResourceID() }
func (obj *resource[GA, Alpha, Beta]) ToGA() (*GA, error)            { return obj.x.ToGA() }
func (obj *resource[GA, Alpha, Beta]) ToAlpha() (*Alpha, error)      { return obj.x.ToAlpha() }
func (obj *resource[GA, Alpha, Beta]) ToBeta() (*Beta, error)        { return obj.x.ToBeta() }

// Diff implements Resource.
func (obj *resource[GA, Alpha, Beta]) Diff(other Resource[GA, Alpha, Beta]) (*DiffResult, error) {
	switch {
	// Comparisons between the same versions don't need conversions.
	//
	// cmp(GA, GA)
	case obj.Version() == meta.VersionGA && other.Version() == meta.VersionGA:
		aObj, _ := obj.ToGA()
		bObj, _ := other.ToGA()
		return diff(aObj, bObj, obj.x.typeTrait.FieldTraits(meta.VersionGA))
	// cmp(Alpha, Alpha)
	case obj.Version() == meta.VersionAlpha && other.Version() == meta.VersionAlpha:
		aObj, _ := obj.ToAlpha()
		bObj, _ := other.ToAlpha()
		return diff(aObj, bObj, obj.x.typeTrait.FieldTraits(meta.VersionAlpha))
	// cmp(Beta, Beta)
	case obj.Version() == meta.VersionBeta && other.Version() == meta.VersionBeta:
		aObj, _ := obj.ToBeta()
		bObj, _ := other.ToBeta()
		return diff(aObj, bObj, obj.x.typeTrait.FieldTraits(meta.VersionBeta))

	// GA => Alpha, GA => Beta should be safe and supported with a conversion.
	//
	// cmp(GA, Alpha), cmp(Alpha, GA): convert to Alpha, then compare.
	case obj.Version() == meta.VersionGA && other.Version() == meta.VersionAlpha:
		fallthrough
	case obj.Version() == meta.VersionAlpha && other.Version() == meta.VersionGA:
		aObj, err := obj.ToAlpha()
		if err != nil {
			return nil, fmt.Errorf("Resource.Diff: %s", err)
		}
		bObj, err := other.ToAlpha()
		if err != nil {
			return nil, fmt.Errorf("Resource.Diff: %s", err)
		}
		return diff(aObj, bObj, obj.x.typeTrait.FieldTraits(meta.VersionAlpha))
	// cmp(GA, Beta), cmp(Beta, GA): convert to Beta, then compare.
	case obj.Version() == meta.VersionGA && other.Version() == meta.VersionBeta:
		fallthrough
	case obj.Version() == meta.VersionBeta && other.Version() == meta.VersionGA:
		aObj, err := obj.ToBeta()
		if err != nil {
			return nil, fmt.Errorf("Resource.Diff: %s", err)
		}
		bObj, err := other.ToBeta()
		if err != nil {
			return nil, fmt.Errorf("Resource.Diff: %s", err)
		}
		return diff(aObj, bObj, obj.x.typeTrait.FieldTraits(meta.VersionBeta))

	// Comparison between Alpha/Beta is not supported right now. This probably
	// can work with some manual conversion logic.
	case obj.Version() == meta.VersionAlpha && other.Version() == meta.VersionBeta:
		return nil, fmt.Errorf("cross alpha/beta diff not supported")
	case obj.Version() == meta.VersionBeta && other.Version() == meta.VersionAlpha:
		return nil, fmt.Errorf("cross beta/alpha diff not supported")
	}

	return nil, fmt.Errorf("invalid versions (got a.Version=%s, b.Version=%s)", obj.Version(), other.Version())
}

/*
func (obj *Resource[GA, Alpha, Beta]) Clone() Resource[GA, Alpha, Beta] {
	return &Resource[GA, Alpha, Beta]{
		x:   obj.Clone(),
		ver: obj.ver,
	}
}
*/
