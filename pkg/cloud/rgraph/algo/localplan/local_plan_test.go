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

package localplan

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/algo/graphviz"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/fake"
)

func TestLocalPlan(t *testing.T) {
	const project = "project-1"
	makeID := func(i int) *cloud.ResourceID {
		return fake.ID(project, meta.GlobalKey(fmt.Sprintf("fake-%d", i)))
	}
	newNodeWithValue := func(i int, v string) rnode.Builder {
		id := makeID(i)
		nb := fake.NewBuilder(id)
		mr := fake.New(project, id.Key)
		mr.Access(func(x *fake.FakeResource) { x.Value = v })
		r, _ := mr.Freeze()
		nb.SetResource(r)
		return nb
	}
	newNode := func(i int) rnode.Builder {
		return newNodeWithValue(i, "")
	}

	for _, tc := range []struct {
		name         string
		setupBuilder func(gotb, wantb *rgraph.Builder)
		setupGraph   func(got, want *rgraph.Graph)
		wantErr      bool
		wantPlan     map[string]rnode.Operation
	}{
		{
			name: "empty graph",
		},
		{
			name: "node exist with no diff (nop)",
			setupBuilder: func(gotb, wantb *rgraph.Builder) {
				node := newNode(0)
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeExists)
				gotb.Add(node)

				node = newNode(0)
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeExists)
				wantb.Add(node)
			},
			wantPlan: map[string]rnode.Operation{
				makeID(0).String(): rnode.OpNothing,
			},
		},
		{
			name: "node does not exist (nop)",
			setupBuilder: func(gotb, wantb *rgraph.Builder) {
				node := newNode(0)
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeDoesNotExist)
				gotb.Add(node)

				node = newNode(0)
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeDoesNotExist)
				wantb.Add(node)
			},
			wantPlan: map[string]rnode.Operation{
				makeID(0).String(): rnode.OpNothing,
			},
		},
		{
			name: "node is not managed (nop)",
			setupBuilder: func(gotb, wantb *rgraph.Builder) {
				node := newNodeWithValue(0, "abc")
				node.SetOwnership(rnode.OwnershipExternal)
				node.SetState(rnode.NodeExists)
				gotb.Add(node)
				// Node contents are different, but as this is unmanaged,
				// there will no planned action.
				node = newNodeWithValue(0, "def")
				node.SetOwnership(rnode.OwnershipExternal)
				node.SetState(rnode.NodeExists)
				wantb.Add(node)
			},
			wantPlan: map[string]rnode.Operation{
				makeID(0).String(): rnode.OpNothing,
			},
		},
		{
			name: "delete resource (1 -> 0 node)",
			setupBuilder: func(gotb, wantb *rgraph.Builder) {
				node := newNode(0)
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeExists)
				gotb.Add(node)

				node = newNode(0)
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeDoesNotExist)
				wantb.Add(node)
			},
			wantPlan: map[string]rnode.Operation{
				makeID(0).String(): rnode.OpDelete,
			},
		},
		{
			name: "create resource (0 -> 1 node)",
			setupBuilder: func(gotb, wantb *rgraph.Builder) {
				node := newNode(0)
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeDoesNotExist)
				gotb.Add(node)

				node = newNode(0)
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeExists)
				wantb.Add(node)
			},
			wantPlan: map[string]rnode.Operation{
				makeID(0).String(): rnode.OpCreate,
			},
		},
		{
			name: "update node",
			setupBuilder: func(gotb, wantb *rgraph.Builder) {
				node := newNodeWithValue(0, "abc")
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeExists)
				gotb.Add(node)

				node = newNodeWithValue(0, "def")
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeExists)
				wantb.Add(node)
			},
			wantPlan: map[string]rnode.Operation{
				makeID(0).String(): rnode.OpUpdate,
			},
		},
		{
			name: "multiple nodes",
			setupBuilder: func(gotb, wantb *rgraph.Builder) {
				// nop
				node := newNode(0)
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeExists)
				gotb.Add(node)

				node = newNode(0)
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeExists)
				wantb.Add(node)
				// delete
				node = newNode(1)
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeExists)
				gotb.Add(node)

				node = newNode(1)
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeDoesNotExist)
				wantb.Add(node)
				// create
				node = newNode(2)
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeDoesNotExist)
				gotb.Add(node)

				node = newNode(2)
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeExists)
				wantb.Add(node)
				// update
				node = newNodeWithValue(3, "abc")
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeExists)
				gotb.Add(node)

				node = newNodeWithValue(3, "def")
				node.SetOwnership(rnode.OwnershipManaged)
				node.SetState(rnode.NodeExists)
				wantb.Add(node)
			},
			wantPlan: map[string]rnode.Operation{
				makeID(0).String(): rnode.OpNothing,
				makeID(1).String(): rnode.OpDelete,
				makeID(2).String(): rnode.OpCreate,
				makeID(3).String(): rnode.OpUpdate,
			},
		},
		{
			name: "error: node in got but not in want",
			setupBuilder: func(gotb, wantb *rgraph.Builder) {
				node := newNode(0)
				node.SetOwnership(rnode.OwnershipManaged)
				gotb.Add(node)
			},
			wantErr: true,
		},
		{
			name: "error: node in want but not in got",
			setupBuilder: func(gotb, wantb *rgraph.Builder) {
				node := newNode(0)
				node.SetOwnership(rnode.OwnershipManaged)
				wantb.Add(node)
			},
			wantErr: true,
		},
		{
			name: "error: invalid state for planning",
			setupBuilder: func(gotb, wantb *rgraph.Builder) {
				node := newNode(0)
				node.SetOwnership(rnode.OwnershipManaged)
				gotb.Add(node)

				node = newNode(0)
				node.SetOwnership(rnode.OwnershipManaged)
				wantb.Add(node)
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

			err = PlanWantGraph(got, want)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("Do() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
			if err != nil {
				return
			}

			t.Logf("got = \n%s", graphviz.Do(got))
			t.Logf("want = \n%s", graphviz.Do(want))

			for _, node := range want.All() {
				op, ok := tc.wantPlan[node.ID().String()]
				if !ok {
					t.Fatalf("node %s in graph, but not in wantPlan", node.ID())
				}
				delete(tc.wantPlan, node.ID().String())

				if op != node.Plan().Op() {
					t.Fatalf("node %s, got op=%s, want %s", node.ID(), node.Plan().Op(), op)
				}
			}

			for k, op := range tc.wantPlan {
				t.Errorf("node %s (op=%s) in wantPlan but not in the graph", k, op)
			}
		})
	}
}
