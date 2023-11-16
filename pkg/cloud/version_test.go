/*
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
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
)

func TestCustomResolver(t *testing.T) {
	customVersions := []CustomVersion{
		{
			Key:     VersionResolverKey{Scope: meta.GlobalScope, Service: "BackendService"},
			Version: meta.VersionAlpha,
		},
		{
			Key:     VersionResolverKey{Scope: meta.RegionalScope, Service: "HealthCheck"},
			Version: meta.VersionBeta,
		},
		{
			Key:     VersionResolverKey{Scope: meta.ZonalScope, Service: "Address"},
			Version: meta.VersionAlpha,
		},
	}
	cr := NewCustomResolver(customVersions...)

	for _, tc := range []struct {
		desc string
		key  VersionResolverKey
		want meta.Version
	}{
		{
			desc: "BackendService in Global scope has custom version",
			key:  VersionResolverKey{Scope: meta.GlobalScope, Service: "BackendService"},
			want: meta.VersionAlpha,
		},
		{
			desc: "BackendService in Zonal scope has default version",
			key:  VersionResolverKey{Scope: meta.ZonalScope, Service: "BackendService"},
			want: meta.VersionGA,
		},
		{
			desc: "BackendService in Regional scope has default version",
			key:  VersionResolverKey{Scope: meta.RegionalScope, Service: "BackendService"},
			want: meta.VersionGA,
		},
		{
			desc: "HealthCheck in Global scope has default version",
			key:  VersionResolverKey{Scope: meta.GlobalScope, Service: "HealthCheck"},
			want: meta.VersionGA,
		},
		{
			desc: "HealthCheck in Zonal scope has default version",
			key:  VersionResolverKey{Scope: meta.ZonalScope, Service: "HealthCheck"},
			want: meta.VersionGA,
		},
		{
			desc: "HealthCheck in Regional scope has custom version",
			key:  VersionResolverKey{Scope: meta.RegionalScope, Service: "HealthCheck"},
			want: meta.VersionBeta,
		},
		{
			desc: "Address in Global scope has default version",
			key:  VersionResolverKey{Scope: meta.GlobalScope, Service: "Address"},
			want: meta.VersionGA,
		},
		{
			desc: "Address in Zonal scope has custom version",
			key:  VersionResolverKey{Scope: meta.ZonalScope, Service: "Address"},
			want: meta.VersionAlpha,
		},
		{
			desc: "Address in Regional scope has custom version",
			key:  VersionResolverKey{Scope: meta.RegionalScope, Service: "Address"},
			want: meta.VersionGA,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			tc := tc
			t.Parallel()
			got := cr.Version(tc.key)
			if got != tc.want {
				t.Errorf("%v != %v", got, tc.want)
			}
		})
	}
}

func TestLazyInitForCustomResolver(t *testing.T) {
	customVersions := []CustomVersion{
		{
			Key:     VersionResolverKey{Scope: meta.GlobalScope, Service: "BackendService"},
			Version: meta.VersionAlpha,
		},
	}

	cr := NewCustomResolver()
	cr.LoadVersions(customVersions...)

	for _, tc := range []struct {
		desc string
		key  VersionResolverKey
		want meta.Version
	}{
		{
			desc: "BackendService in Global scope has custom version",
			key:  VersionResolverKey{Scope: meta.GlobalScope, Service: "BackendService"},
			want: meta.VersionAlpha,
		},
		{
			desc: "BackendService in Zonal scope has default version",
			key:  VersionResolverKey{Scope: meta.ZonalScope, Service: "BackendService"},
			want: meta.VersionGA,
		},
		{
			desc: "BackendService in Regional scope has default version",
			key:  VersionResolverKey{Scope: meta.RegionalScope, Service: "BackendService"},
			want: meta.VersionGA,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			tc := tc
			t.Parallel()
			got := cr.Version(tc.key)
			if got != tc.want {
				t.Errorf("%v != %v", got, tc.want)
			}
		})
	}
}
