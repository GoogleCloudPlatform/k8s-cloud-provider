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

package actions

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/fake"
)

func TestActions(t *testing.T) {
	id := fake.ID("project-1", meta.GlobalKey("fake-1"))

	newNode := func() rnode.Builder {
		nb := fake.NewBuilder(id)
		nb.SetOwnership(rnode.OwnershipManaged)
		return nb
	}

	for _, tc := range []struct {
		name         string
		setupBuilder func(gotb, wantb *rgraph.Builder)
		setupGraph   func(got, want *rgraph.Graph)
		wantErr      bool
	}{
		{
			name: "success",
			setupBuilder: func(gotb, wantb *rgraph.Builder) {
				gotb.Add(newNode())
				wantb.Add(newNode())
			},
			setupGraph: func(got, want *rgraph.Graph) {
				// Set the planned action to an update.
				fakeNode := want.Get(id)
				fakeNode.Plan().Set(rnode.PlanDetails{
					Operation: rnode.OpUpdate,
					Why:       "test plan",
				})
			},
		},
		{
			name: "node in want does not exist in got",
			setupBuilder: func(gotb, wantb *rgraph.Builder) {
				wantb.Add(newNode())
			},
			wantErr: true,
		},
		{
			name: "invalid plan",
			setupBuilder: func(gotb, wantb *rgraph.Builder) {
				gotb.Add(newNode())
				wantb.Add(newNode())
			},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotb := rgraph.NewBuilder()
			wantb := rgraph.NewBuilder()

			if tc.setupBuilder != nil {
				tc.setupBuilder(gotb, wantb)
			}

			got, err := gotb.Build()
			if err != nil {
				t.Fatalf("gotb.Build() = _, %v, want nil", err)
			}
			want, err := wantb.Build()
			if err != nil {
				t.Fatalf("wantb.Build() = _, %v, want nil", err)
			}

			if tc.setupGraph != nil {
				tc.setupGraph(got, want)
			}

			actions, err := Do(got, want)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("Do() = _, %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
			if err != nil {
				return
			}

			if len(actions) != 1 || !strings.HasPrefix(actions[0].String(), "EventAction") {
				t.Errorf("actions = %v, want [EventAction...]", actions)
			}

			t.Log(actions)
		})
	}
}
