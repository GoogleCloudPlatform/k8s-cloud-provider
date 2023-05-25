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

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/address"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/fake"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/targethttpproxy"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/compute/v1"
)

func TestBuilderSetResource(t *testing.T) {
	id := ID("proj", meta.GlobalKey("fr"))

	newR := func() rnode.UntypedResource {
		mr := NewMutableForwardingRule(id.ProjectID, id.Key)
		r, _ := mr.Freeze()
		return r
	}

	for _, tc := range []struct {
		name    string
		r       rnode.UntypedResource
		wantErr bool
	}{
		{
			name: "ok",
			r:    newR(),
		},
		{
			name:    "wrong type",
			r:       fake.Fake(nil), // this will fail to cast.
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder(id)
			err := b.SetResource(tc.r)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Errorf("SetResource() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
		})
	}
}

func TestOutRefs(t *testing.T) {
	id := ID("proj", meta.GlobalKey("fr"))
	addrID := address.ID("proj", meta.GlobalKey("addr"))
	targetID := targethttpproxy.ID("proj", meta.GlobalKey("tp"))

	for _, tc := range []struct {
		name string
		f    func(*compute.ForwardingRule)

		wantErr bool
		want    []rnode.ResourceRef
	}{
		{
			name: "numeric ip address",
			f: func(x *compute.ForwardingRule) {
				x.IPAddress = "1.2.3.4"
			},
		},
		{
			name: "address resource",
			f: func(x *compute.ForwardingRule) {
				x.IPAddress = addrID.SelfLink(meta.VersionGA)
			},
			want: []rnode.ResourceRef{
				{From: id, To: addrID, Path: api.Path{}.Pointer().Field("IPAddress")},
			},
		},
		{
			name: "target",
			f: func(x *compute.ForwardingRule) {
				x.Target = targetID.SelfLink(meta.VersionGA)
			},
			want: []rnode.ResourceRef{
				{From: id, To: targetID, Path: api.Path{}.Pointer().Field("Target")},
			},
		},
		{
			name: "target and address",
			f: func(x *compute.ForwardingRule) {
				x.IPAddress = addrID.SelfLink(meta.VersionGA)
				x.Target = targetID.SelfLink(meta.VersionGA)
			},
			want: []rnode.ResourceRef{
				{From: id, To: addrID, Path: api.Path{}.Pointer().Field("IPAddress")},
				{From: id, To: targetID, Path: api.Path{}.Pointer().Field("Target")},
			},
		},
		{
			name: "garbage IP",
			f: func(x *compute.ForwardingRule) {
				x.IPAddress = "garbage"
			},
			wantErr: true,
		},
		{
			name: "garbage target",
			f: func(x *compute.ForwardingRule) {
				x.Target = "garbage"
			},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mr := NewMutableForwardingRule(id.ProjectID, id.Key)
			mr.Access(tc.f)
			r, _ := mr.Freeze()
			b := NewBuilderWithResource(r)

			got, err := b.OutRefs()
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("XXX")
			} else if gotErr {
				return
			}
			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("OutRefs diff = -got,+want: %s", diff)
			}
		})
	}
}
