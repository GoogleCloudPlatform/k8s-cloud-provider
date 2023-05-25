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

package address

import (
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"google.golang.org/api/compute/v1"
)

func TestAddressSchema(t *testing.T) {
	const proj = "proj-1"
	key := meta.GlobalKey("key-1")
	x := NewMutableAddress(proj, key)
	if err := x.CheckSchema(); err != nil {
		t.Fatalf("CheckSchema() = %v, want nil", err)
	}
}

func TestAddressFieldTraits(t *testing.T) {
	for _, tc := range []struct {
		name     string
		a, b     *compute.Address
		wantDiff bool
	}{
		{
			name: "same",
			a: &compute.Address{
				Name:    "addr-1",
				Address: "1.2.3.4",
			},
			b: &compute.Address{
				Name:    "addr-1",
				Address: "1.2.3.4",
			},
		},
		{
			name: "ignored fields",
			a: &compute.Address{
				Name:              "addr-1",
				Address:           "1.2.3.4",
				Kind:              "zzz",
				Id:                123,
				CreationTimestamp: "zzz",
				Status:            "IN_USE",
				Region:            "zzz",
				SelfLink:          "zzz",
				Users:             []string{"zzz"},
			},
			b: &compute.Address{
				Name:    "addr-1",
				Address: "1.2.3.4",
			},
		},
		{
			name: "non-ignored fields",
			a: &compute.Address{
				Name:        "addr-1",
				Address:     "1.2.3.4",
				NetworkTier: "PREMIUM",
			},
			b: &compute.Address{
				Name:        "addr-1",
				Address:     "1.2.3.4",
				NetworkTier: "STANDARD",
			},
			wantDiff: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := NewMutableAddress("p1", meta.GlobalKey("addr-1"))
			a.Set(tc.a)
			b := NewMutableAddress("p1", meta.GlobalKey("addr-1"))
			b.Set(tc.b)

			fa, err := a.Freeze()
			if err != nil {
				t.Fatalf("a.Freeze() = %v, want nil", err)
			}
			fb, err := b.Freeze()
			if err != nil {
				t.Fatalf("b.Freeze() = %v, want nil", err)
			}

			r, err := fa.Diff(fb)
			if err != nil {
				t.Fatalf("Diff() = %v, want nil", err)
			}
			if r.HasDiff() != tc.wantDiff {
				t.Errorf("result = %+v, HasDiff() = %t, want %t", r, r.HasDiff(), tc.wantDiff)
			}
		})
	}
}
