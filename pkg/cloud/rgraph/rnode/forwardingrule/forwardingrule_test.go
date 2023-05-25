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

package forwardingrule

import (
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"google.golang.org/api/compute/v1"
)

func TestForwardingRuleSchema(t *testing.T) {
	const proj = "proj-1"
	key := meta.GlobalKey("key-1")
	x := NewMutableForwardingRule(proj, key)
	if err := x.CheckSchema(); err != nil {
		t.Fatalf("CheckSchema() = %v, want nil", err)
	}
}

func TestForwardingRuleFieldTraits(t *testing.T) {
	for _, tc := range []struct {
		name     string
		a, b     *compute.ForwardingRule
		wantDiff bool
	}{
		{
			name: "same",
			a: &compute.ForwardingRule{
				Name:   "res-1",
				Target: "ZZZ",
			},
			b: &compute.ForwardingRule{
				Name:   "res-1",
				Target: "ZZZ",
			},
		},
		{
			name: "ignored fields",
			a: &compute.ForwardingRule{
				Name:                "addr-1",
				Target:              "ZZZ",
				Kind:                "zzz",
				Id:                  123,
				CreationTimestamp:   "zzz",
				Region:              "zzz",
				SelfLink:            "zzz",
				PscConnectionId:     123,
				PscConnectionStatus: "zzz",
				BaseForwardingRule:  "zzz",
			},
			b: &compute.ForwardingRule{
				Name:   "addr-1",
				Target: "ZZZ",
			},
		},
		{
			name: "non-ignored fields",
			a: &compute.ForwardingRule{
				Name:   "addr-1",
				Target: "aaa",
			},
			b: &compute.ForwardingRule{
				Name:   "addr-1",
				Target: "bbb",
			},
			wantDiff: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := NewMutableForwardingRule("p1", meta.GlobalKey("addr-1"))
			a.Set(tc.a)
			b := NewMutableForwardingRule("p1", meta.GlobalKey("addr-1"))
			b.Set(tc.b)

			fa, _ := a.Freeze()
			fb, _ := b.Freeze()

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
