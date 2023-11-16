/*
Copyright 2018 Google LLC

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

package cloud

import (
	"sync"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
)

// VersionResolverKey stores information needed to fetch the version for resource.
type VersionResolverKey struct {
	Service string
	Scope   meta.Scope
}

// VersionResolver resolves the version for given service in given scope.
// Supported versions GA, Alpha, Beta
// Supported scopes Global, Regional, Zonal
type VersionResolver interface {
	// This allows for plumbing different versions for services in different scope.
	Version(key VersionResolverKey) meta.Version
}

// SimpleVersionResolver resolves all versions to meta.GA.
type SimpleVersionResolver struct{}

// Implements VersionResolver.
func (r SimpleVersionResolver) Version(VersionResolverKey) meta.Version {
	return meta.VersionGA
}

// CustomVersion maps version key and value for loading custom versions.
type CustomVersion struct {
	Key     VersionResolverKey
	Version meta.Version
}

// CustomResolver enables setting version per service and scope.
// If the version key does not exist in versions map GA version is returned.
// This class is thread safety.
type customResolver struct {
	// guards versions map
	lock sync.Mutex

	versions map[VersionResolverKey]meta.Version
}

// NewCustomResolver creates customResolver with custom versions.
func NewCustomResolver(versions ...CustomVersion) *customResolver {
	cr := customResolver{
		versions: make(map[VersionResolverKey]meta.Version),
	}
	for _, customVersion := range versions {
		cr.versions[customVersion.Key] = customVersion.Version
	}
	return &cr
}

// LoadVersions enables adding custom versions to existing resolver.
// If a custom version already exists in the map it will be overridden.
func (r *customResolver) LoadVersions(versions ...CustomVersion) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.versions == nil {
		r.versions = make(map[VersionResolverKey]meta.Version)
	}

	for _, customVersion := range versions {
		r.versions[customVersion.Key] = customVersion.Version
	}
}

// Implements VersionResolver.
func (r *customResolver) Version(key VersionResolverKey) meta.Version {
	r.lock.Lock()
	defer r.lock.Unlock()
	ver, ok := r.versions[key]
	if ok {
		return ver
	}
	return meta.VersionGA
}
