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
	"errors"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
)

func TestEqualResourceID(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		a *ResourceID
		b *ResourceID
	}{
		{
			a: &ResourceID{"some-gce-project", meta.APIGroupCompute, "projects", nil},
			b: &ResourceID{"some-gce-project", meta.APIGroupCompute, "projects", nil},
		},
		{
			a: &ResourceID{"", meta.APIGroupCompute, "networks", meta.GlobalKey("my-net")},
			b: &ResourceID{"", meta.APIGroupCompute, "networks", meta.GlobalKey("my-net")},
		},
		{
			a: &ResourceID{"some-gce-project", meta.APIGroupCompute, "projects", meta.GlobalKey("us-central1")},
			b: &ResourceID{"some-gce-project", meta.APIGroupCompute, "projects", meta.GlobalKey("us-central1")},
		},
		{
			a: nil,
			b: nil,
		},
	} {
		if !tc.a.Equal(tc.b) {
			t.Errorf("%v.Equal(%v) = false, want true", tc.a, tc.b)
		}
	}

	for _, tc := range []struct {
		a *ResourceID
		b *ResourceID
	}{
		{
			a: &ResourceID{"some-gce-project", meta.APIGroupCompute, "projects", nil},
			b: &ResourceID{"some-other-project", meta.APIGroupCompute, "projects", nil},
		},
		{
			a: &ResourceID{"some-gce-project", meta.APIGroupCompute, "projects", nil},
			b: &ResourceID{"some-gce-project", meta.APIGroupCompute, "projects", meta.GlobalKey("us-central1")},
		},
		{
			a: &ResourceID{"some-gce-project", meta.APIGroupCompute, "networks", meta.GlobalKey("us-central1")},
			b: &ResourceID{"some-gce-project", meta.APIGroupCompute, "projects", meta.GlobalKey("us-central1")},
		},
		{
			a: &ResourceID{"some-gce-project", meta.APIGroupCompute, "projects", meta.GlobalKey("us-central1")},
			b: nil,
		},
	} {
		if tc.a.Equal(tc.b) {
			t.Errorf("%v.Equal(%v) = true, want false", tc.a, tc.b)
		}
	}
}

func TestResourceIDString(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		id   *ResourceID
		want string
	}{
		{
			id:   &ResourceID{"proj1", meta.APIGroupNetworkServices, "res1", meta.GlobalKey("key1")},
			want: "networkservices/res1:proj1/key1",
		},
		{
			id:   &ResourceID{"proj1", meta.APIGroupCompute, "res1", meta.RegionalKey("key1", "us-central1")},
			want: "compute/res1:proj1/us-central1/key1",
		},
		{
			id:   &ResourceID{"proj1", meta.APIGroupCompute, "res1", meta.ZonalKey("key1", "us-central1-c")},
			want: "compute/res1:proj1/us-central1-c/key1",
		},
	} {
		got := tc.id.String()
		if got != tc.want {
			t.Errorf("String() = %q, want %q (id = %+v)", got, tc.want, tc.id)
		}
	}
}

func TestParseResourceURL(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		in string
		r  *ResourceID
	}{
		{
			"https://www.googleapis.com/compute/v1/projects/some-gce-project",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "projects", nil},
		},
		{
			"https://compute.googleapis.com/v1/projects/some-gce-project",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "projects", nil},
		},
		{
			"https://www.googleapis.com/compute/v1/projects/some-gce-project/regions/us-central1",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "regions", meta.GlobalKey("us-central1")},
		},
		{
			"https://compute.googleapis.com/v1/projects/some-gce-project/regions/us-central1",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "regions", meta.GlobalKey("us-central1")},
		},
		{
			"https://www.googleapis.com/networkservices/v1/projects/some-gce-project/regions/us-central1",
			&ResourceID{"some-gce-project", meta.APIGroupNetworkServices, "regions", meta.GlobalKey("us-central1")},
		},
		{
			"https://networkservices.googleapis.com/v1/projects/some-gce-project/regions/us-central1",
			&ResourceID{"some-gce-project", meta.APIGroupNetworkServices, "regions", meta.GlobalKey("us-central1")},
		},
		{
			"https://www.googleapis.com/compute/v1/projects/some-gce-project/zones/us-central1-b",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "zones", meta.GlobalKey("us-central1-b")},
		},
		{
			"https://compute.googleapis.com/v1/projects/some-gce-project/zones/us-central1-b",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "zones", meta.GlobalKey("us-central1-b")},
		},
		{
			"https://www.googleapis.com/compute/v1/projects/some-gce-project/global/operations/operation-1513289952196-56054460af5a0-b1dae0c3-9bbf9dbf",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "operations", meta.GlobalKey("operation-1513289952196-56054460af5a0-b1dae0c3-9bbf9dbf")},
		},
		{
			"https://compute.googleapis.com/v1/projects/some-gce-project/global/operations/operation-1513289952196-56054460af5a0-b1dae0c3-9bbf9dbf",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "operations", meta.GlobalKey("operation-1513289952196-56054460af5a0-b1dae0c3-9bbf9dbf")},
		},
		{
			"https://www.googleapis.com/compute/alpha/projects/some-gce-project/regions/us-central1/addresses/my-address",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "addresses", meta.RegionalKey("my-address", "us-central1")},
		},
		{
			"https://compute.googleapis.com/alpha/projects/some-gce-project/regions/us-central1/addresses/my-address",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "addresses", meta.RegionalKey("my-address", "us-central1")},
		},
		{
			"https://www.googleapis.com/compute/v1/projects/some-gce-project/zones/us-central1-c/instances/instance-1",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "instances", meta.ZonalKey("instance-1", "us-central1-c")},
		},
		{
			"https://compute.googleapis.com/v1/projects/some-gce-project/zones/us-central1-c/instances/instance-1",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "instances", meta.ZonalKey("instance-1", "us-central1-c")},
		},
		{
			"http://localhost:3990/compute/beta/projects/some-gce-project/global/operations/operation-1513289952196-56054460af5a0-b1dae0c3-9bbf9dbf",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "operations", meta.GlobalKey("operation-1513289952196-56054460af5a0-b1dae0c3-9bbf9dbf")},
		},
		{
			"http://localhost:3990/compute/alpha/projects/some-gce-project/regions/dev-central1/addresses/my-address",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "addresses", meta.RegionalKey("my-address", "dev-central1")},
		},
		{
			"http://localhost:3990/networkservices/alpha/projects/some-gce-project/regions/dev-central1/addresses/my-address",
			&ResourceID{"some-gce-project", meta.APIGroupNetworkServices, "addresses", meta.RegionalKey("my-address", "dev-central1")},
		},
		{
			"http://localhost:3990/compute/v1/projects/some-gce-project/zones/dev-central1-std/instances/instance-1",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "instances", meta.ZonalKey("instance-1", "dev-central1-std")},
		},
		{
			"projects/some-gce-project",
			&ResourceID{"some-gce-project", "", "projects", nil},
		},
		{
			"projects/some-gce-project/regions/us-central1",
			&ResourceID{"some-gce-project", "", "regions", meta.GlobalKey("us-central1")},
		},
		{
			"projects/some-gce-project/zones/us-central1-b",
			&ResourceID{"some-gce-project", "", "zones", meta.GlobalKey("us-central1-b")},
		},
		{
			"projects/some-gce-project/global/operations/operation-1513289952196-56054460af5a0-b1dae0c3-9bbf9dbf",
			&ResourceID{"some-gce-project", "", "operations", meta.GlobalKey("operation-1513289952196-56054460af5a0-b1dae0c3-9bbf9dbf")},
		},
		{
			"projects/some-gce-project/regions/us-central1/addresses/my-address",
			&ResourceID{"some-gce-project", "", "addresses", meta.RegionalKey("my-address", "us-central1")},
		},
		{
			"projects/some-gce-project/zones/us-central1-c/instances/instance-1",
			&ResourceID{"some-gce-project", "", "instances", meta.ZonalKey("instance-1", "us-central1-c")},
		},
		{
			"global/networks/my-network",
			&ResourceID{"", "", "networks", meta.GlobalKey("my-network")},
		},
		{
			"regions/us-central1/subnetworks/my-subnet",
			&ResourceID{"", "", "subnetworks", meta.RegionalKey("my-subnet", "us-central1")},
		},
		{
			"zones/us-central1-c/instances/instance-1",
			&ResourceID{"", "", "instances", meta.ZonalKey("instance-1", "us-central1-c")},
		},
		{
			"https://compute.googleapis.com/compute/v1/projects/some-gce-project/regions/us-central1/backendServices/bs1",
			&ResourceID{"some-gce-project", meta.APIGroupCompute, "backendServices", meta.RegionalKey("bs1", "us-central1")},
		},
	} {
		t.Run(tc.in, func(t *testing.T) {
			r, err := ParseResourceURL(tc.in)
			if err != nil {
				t.Errorf("Error from ParseResourceURL(%q) = %+v, %v; want _, nil", tc.in, r, err)
				return
			}
			if !r.Equal(tc.r) {
				t.Errorf("Unexpected output from ParseResourceURL(%q) = %+v, nil; want %+v, nil", tc.in, r, tc.r)
			}
		})
	}

	// Malformed URLs.
	for _, tc := range []string{
		"",
		"/",
		"/a",
		"/a/b",
		"/a/b/c",
		"/a/b/c/d",
		"/a/b/c/d/e",
		"/a/b/c/d/e/f",
		"https://www.googleapis.com/compute/v1/projects/some-gce-project/global",
		"projects/some-gce-project/global",
		"projects/some-gce-project/global/foo",
		"projects/some-gce-project/global/foo/bar/baz",
		"projects/some-gce-project/regions/us-central1/res",
		"projects/some-gce-project/zones/us-central1-c/res",
		"projects/some-gce-project/zones/us-central1-c/res/name/extra",
	} {
		r, err := ParseResourceURL(tc)
		if err == nil {
			t.Errorf("ParseResourceURL(%q) = %+v, %v, want _, error", tc, r, err)
		}
	}
}

type A struct {
	A, B, C string
}

type B struct {
	A, B, D string
}

type E struct{}

func (*E) MarshalJSON() ([]byte, error) {
	return nil, errors.New("injected error")
}

func TestCopyVisJSON(t *testing.T) {
	t.Parallel()

	var b B
	srcA := &A{"aa", "bb", "cc"}
	err := copyViaJSON(&b, srcA)
	if err != nil {
		t.Errorf(`copyViaJSON(&b, %+v) = %v, want nil`, srcA, err)
	} else {
		expectedB := B{"aa", "bb", ""}
		if b != expectedB {
			t.Errorf("b == %+v, want %+v", b, expectedB)
		}
	}

	var a A
	srcB := &B{"aaa", "bbb", "ccc"}
	err = copyViaJSON(&a, srcB)
	if err != nil {
		t.Errorf(`copyViaJSON(&a, %+v) = %v, want nil`, srcB, err)
	} else {
		expectedA := A{"aaa", "bbb", ""}
		if a != expectedA {
			t.Errorf("a == %+v, want %+v", a, expectedA)
		}
	}

	if err := copyViaJSON(&a, &E{}); err == nil {
		t.Errorf("copyViaJSON(&a, &E{}) = nil, want error")
	}
}

func TestResourceIdSelfLink(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		resourceID *ResourceID
		ver        meta.Version
		want       string
	}{
		{
			&ResourceID{"proj1", meta.APIGroupNetworkServices, "res1", meta.GlobalKey("key1")},
			meta.VersionGA,
			"https://www.googleapis.com/networkservices/v1/projects/proj1/global/res1/key1",
		},
		{
			&ResourceID{"proj1", meta.APIGroupCompute, "res1", meta.GlobalKey("key1")},
			meta.VersionAlpha,
			"https://www.googleapis.com/compute/alpha/projects/proj1/global/res1/key1",
		},
		{
			&ResourceID{"proj1", "", "res1", meta.GlobalKey("key1")},
			meta.VersionAlpha,
			"https://www.googleapis.com/compute/alpha/projects/proj1/global/res1/key1",
		},
	} {
		if link := tc.resourceID.SelfLink(tc.ver); link != tc.want {
			t.Errorf("ResourceID{%+v}.SelfLink(%v) = %v, want %q", tc.resourceID, tc.ver, link, tc.want)
		}
	}
}

func TestSelfLink(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		ver      meta.Version
		project  string
		resource string
		key      *meta.Key
		want     string
	}{
		{
			meta.VersionAlpha,
			"proj1",
			"addresses",
			meta.RegionalKey("key1", "us-central1"),
			"https://www.googleapis.com/compute/alpha/projects/proj1/regions/us-central1/addresses/key1",
		},
		{
			meta.VersionBeta,
			"proj3",
			"disks",
			meta.ZonalKey("key2", "us-central1-b"),
			"https://www.googleapis.com/compute/beta/projects/proj3/zones/us-central1-b/disks/key2",
		},
		{
			meta.VersionGA,
			"proj4",
			"urlMaps",
			meta.GlobalKey("key3"),
			"https://www.googleapis.com/compute/v1/projects/proj4/global/urlMaps/key3",
		},
		{
			meta.VersionGA,
			"proj4",
			"projects",
			nil,
			"https://www.googleapis.com/compute/v1/projects/proj4",
		},
		{
			meta.VersionGA,
			"proj4",
			"regions",
			meta.GlobalKey("us-central1"),
			"https://www.googleapis.com/compute/v1/projects/proj4/regions/us-central1",
		},
		{
			meta.VersionGA,
			"proj4",
			"zones",
			meta.GlobalKey("us-central1-a"),
			"https://www.googleapis.com/compute/v1/projects/proj4/zones/us-central1-a",
		},
	} {
		if link := SelfLink(tc.ver, tc.project, tc.resource, tc.key); link != tc.want {
			t.Errorf("SelfLink(%v, %q, %q, %v) = %v, want %q", tc.ver, tc.project, tc.resource, tc.key, link, tc.want)
		}
	}
}

func TestSelfLinkWithGroup(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		apiGroup meta.APIGroup
		ver      meta.Version
		project  string
		resource string
		key      *meta.Key
		want     string
	}{
		{
			meta.APIGroupCompute,
			meta.VersionAlpha,
			"proj1",
			"addresses",
			meta.RegionalKey("key1", "us-central1"),
			"https://www.googleapis.com/compute/alpha/projects/proj1/regions/us-central1/addresses/key1",
		},
		{
			meta.APIGroupCompute,
			meta.VersionBeta,
			"proj3",
			"disks",
			meta.ZonalKey("key2", "us-central1-b"),
			"https://www.googleapis.com/compute/beta/projects/proj3/zones/us-central1-b/disks/key2",
		},
		{
			meta.APIGroupCompute,
			meta.VersionGA,
			"proj4",
			"urlMaps",
			meta.GlobalKey("key3"),
			"https://www.googleapis.com/compute/v1/projects/proj4/global/urlMaps/key3",
		},
		{
			meta.APIGroupCompute,
			meta.VersionGA,
			"proj4",
			"projects",
			nil,
			"https://www.googleapis.com/compute/v1/projects/proj4",
		},
		{
			meta.APIGroupCompute,
			meta.VersionGA,
			"proj4",
			"regions",
			meta.GlobalKey("us-central1"),
			"https://www.googleapis.com/compute/v1/projects/proj4/regions/us-central1",
		},
		{
			meta.APIGroupCompute,
			meta.VersionGA,
			"proj4",
			"zones",
			meta.GlobalKey("us-central1-a"),
			"https://www.googleapis.com/compute/v1/projects/proj4/zones/us-central1-a",
		},
		{
			meta.APIGroupNetworkServices,
			meta.VersionGA,
			"proj4",
			"tcproutes",
			meta.ZonalKey("key2", "us-central1-a"),
			"https://www.googleapis.com/networkservices/v1/projects/proj4/zones/us-central1-a/tcproutes/key2",
		},
		{
			meta.APIGroup("foo"),
			meta.VersionGA,
			"proj4",
			"tcproutes",
			meta.ZonalKey("key1", "us-central1-a"),
			"https://www.googleapis.com/invalid-apigroup/v1/projects/proj4/zones/us-central1-a/tcproutes/key1",
		},
	} {
		if link := SelfLinkWithGroup(tc.apiGroup, tc.ver, tc.project, tc.resource, tc.key); link != tc.want {
			t.Errorf("SelfLinkWithGroup(%v, %v, %q, %q, %v) = %v, want %q", tc.apiGroup, tc.ver, tc.project, tc.resource, tc.key, link, tc.want)
		}
	}
}

// Test that SelfLink() returns the overridden api domain.
// This test is not run in parallel since it modifies global vars.
func TestSelfLinkWithSetAPIDomain(t *testing.T) {
	// Reset domain.
	defer func() { SetAPIDomain("https://www.googleapis.com") }()

	for _, tc := range []struct {
		ver      meta.Version
		domain   string
		project  string
		resource string
		key      *meta.Key
		want     string
	}{
		{
			meta.VersionAlpha,
			"http://www.foo.com",
			"proj1",
			"addresses",
			meta.RegionalKey("key1", "us-central1"),
			"http://www.foo.com/compute/alpha/projects/proj1/regions/us-central1/addresses/key1",
		},
		{
			meta.VersionBeta,
			"www.bar.com",
			"proj3",
			"disks",
			meta.ZonalKey("key2", "us-central1-b"),
			"www.bar.com/compute/beta/projects/proj3/zones/us-central1-b/disks/key2",
		},
		{
			meta.VersionGA,
			"baz.com",
			"proj4",
			"urlMaps",
			meta.GlobalKey("key3"),
			"baz.com/compute/v1/projects/proj4/global/urlMaps/key3",
		},
		{
			meta.VersionGA,
			"https://foo.bar",
			"proj4",
			"projects",
			nil,
			"https://foo.bar/compute/v1/projects/proj4",
		},
		{
			meta.VersionGA,
			"https://www.foo.com",
			"proj4",
			"regions",
			meta.GlobalKey("us-central1"),
			"https://www.foo.com/compute/v1/projects/proj4/regions/us-central1",
		},
		{
			meta.VersionGA,
			"http://foo.com",
			"proj4",
			"zones",
			meta.GlobalKey("us-central1-a"),
			"http://foo.com/compute/v1/projects/proj4/zones/us-central1-a",
		},
	} {
		SetAPIDomain(tc.domain)

		if link := SelfLink(tc.ver, tc.project, tc.resource, tc.key); link != tc.want {
			t.Errorf("SelfLink(%v, %q, %q, %v) = %v, want %q", tc.ver, tc.project, tc.resource, tc.key, link, tc.want)
		}
	}
}

func TestAggregatedListKey(t *testing.T) {
	for _, tc := range []struct {
		key          *meta.Key
		expectOutput string
	}{
		{
			key:          meta.ZonalKey("a", "zone1"),
			expectOutput: "zones/zone1",
		},
		{
			key:          meta.RegionalKey("a", "region1"),
			expectOutput: "regions/region1",
		},
		{
			key:          meta.GlobalKey("a"),
			expectOutput: "global",
		},
	} {
		if tc.expectOutput != aggregatedListKey(tc.key) {
			t.Errorf("expect output %q, but got %q", tc.expectOutput, aggregatedListKey(tc.key))
		}
	}
}
