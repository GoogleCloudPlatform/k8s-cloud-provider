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

package traversal

import (
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/fake"
	"github.com/google/go-cmp/cmp"
)

const (
	project = "proj"
)

// parseGraph converts a Graphviz-like syntax into a Graph for testing.
//
// # Example
//
//   - "a" is a graph with a single node "a".
//   - "a->b;c->b" is a graph with edges (a,b), (c,b).
//   - "a -> b -> c; b -> d" is a graph with the following OutRef edges: (a,b),
//     (b,c), (b,d).
func parseGraph(t *testing.T, s string) *rgraph.Graph {
	b := rgraph.NewBuilder()

	paths := strings.Split(s, ";")
	for _, pathStr := range paths {
		path := strings.Split(pathStr, "->")

		if len(path) == 0 || len(path) == 1 && path[0] == "" {
			continue
		}

		// Singletons are added as nodes with no outRefs.
		if len(path) == 1 {
			nodeName := strings.TrimSpace(path[0])
			t.Logf("node %q", nodeName)
			nodeID := fake.ID(project, meta.GlobalKey(nodeName))
			node := b.Get(nodeID)
			if node == nil {
				node = fake.NewBuilder(nodeID)
				node.SetOwnership(rnode.OwnershipManaged)
				b.Add(node)
			}
			continue
		}

		for i := 1; i < len(path); i++ {
			FromName := strings.TrimSpace(path[i-1])
			ToName := strings.TrimSpace(path[i])
			t.Logf("edge: %q -> %q", FromName, ToName)
			FromID := fake.ID(project, meta.GlobalKey(FromName))
			ToID := fake.ID(project, meta.GlobalKey(ToName))
			// Add nodes if they don't exist.

			from := b.Get(FromID)
			if from == nil {
				from = fake.NewBuilder(FromID)
				from.SetOwnership(rnode.OwnershipManaged)
				b.Add(from)
			}
			to := b.Get(ToID)
			if to == nil {
				to = fake.NewBuilder(ToID)
				to.SetOwnership(rnode.OwnershipManaged)
				b.Add(to)
			}
			fakeFrom := from.(*fake.Builder)
			fakeFrom.FakeOutRefs = append(fakeFrom.FakeOutRefs, rnode.ResourceRef{
				From: FromID,
				To:   ToID,
			})
		}
	}

	ret, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	return ret
}

func TestConnectedSubgraph(t *testing.T) {
	for _, tc := range []struct {
		name    string
		start   string
		graph   string
		want    []string
		wantErr bool
	}{
		{
			name:    "error: empty graph",
			wantErr: true,
		},
		{
			name:    "error: node not in graph",
			graph:   "a",
			wantErr: true,
		},
		{
			name:  "one node",
			graph: "a",
			start: "a",
			want:  []string{"a"},
		},
		{
			name:  "one node, disconnected",
			graph: "a; b->c; c->d",
			start: "a",
			want:  []string{"a"},
		},
		{
			name:  "outref one hop",
			graph: "a->b; c",
			start: "a",
			want:  []string{"a", "b"},
		},
		{
			name:  "outref two hops",
			graph: "a->b->c; d",
			start: "a",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "outref many hops",
			graph: "a->b->c->d->e",
			start: "a",
			want:  []string{"a", "b", "c", "d", "e"},
		},
		{
			name:  "v graph 1",
			graph: "a->b;c->b",
			start: "a",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "v graph 2",
			graph: "a->b;c->b",
			start: "b",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "inref one hop",
			graph: "a->b",
			start: "b",
			want:  []string{"a", "b"},
		},
		{
			name:  "inref two hops",
			graph: "a->b->c",
			start: "c",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "inref many hops",
			graph: "a->b->c->d->e",
			start: "e",
			want:  []string{"a", "b", "c", "d", "e"},
		},
		{
			name:  "tree root",
			graph: "a->b; a->c",
			start: "a",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "tree leaf",
			graph: "a->b; a->c",
			start: "b",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "tree big root",
			graph: "a->b; a->c->d; c->e",
			start: "a",
			want:  []string{"a", "b", "c", "d", "e"},
		},
		{
			name:  "tree big middle",
			graph: "a->b; a->c->d; c->e",
			start: "c",
			want:  []string{"a", "b", "c", "d", "e"},
		},
		{
			name:  "tree big leaf",
			graph: "a->b; a->c->d; c->e",
			start: "e",
			want:  []string{"a", "b", "c", "d", "e"},
		},
		{
			name:  "cycle one",
			graph: "a->a",
			start: "a",
			want:  []string{"a"},
		},
		{
			name:  "cycle two",
			graph: "a->b;b->a",
			start: "a",
			want:  []string{"a", "b"},
		},
		{
			name:  "cycle many",
			graph: "a->b;b->c;c->d;d->a",
			start: "a",
			want:  []string{"a", "b", "c", "d"},
		},
		{
			name:  "complex",
			graph: "a->b->c; b->c; c->a; c->d->e",
			start: "a",
			want:  []string{"a", "b", "c", "d", "e"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := parseGraph(t, tc.graph)

			var startNode rnode.Node
			if tc.start == "" {
				// Create sentinel node.
				startID := fake.ID(project, meta.GlobalKey("sentinel"))
				nb := fake.NewBuilder(startID)
				var err error
				startNode, err = nb.Build()
				if err != nil {
					t.Fatal(err)
				}
			} else {
				startID := fake.ID(project, meta.GlobalKey(tc.start))
				startNode = g.Get(startID)
			}
			nodes, err := ConnectedSubgraph(g, startNode)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("ConnectedSubgraph() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}

			got := map[string]struct{}{}
			for _, n := range nodes {
				got[n.ID().String()] = struct{}{}
			}
			want := map[string]struct{}{}
			for _, w := range tc.want {
				want[fake.ID(project, meta.GlobalKey(w)).String()] = struct{}{}
			}

			if diff := cmp.Diff(got, want); diff != "" {
				t.Fatalf("Diff() -got+want: %s", diff)
			}
		})
	}
}

func TestTransitiveRefs(t *testing.T) {
	for _, tc := range []struct {
		name    string
		start   string
		graph   string
		want    []string
		wantErr bool
	}{
		{
			name:    "empty graph",
			wantErr: true,
		},
		{
			name:  "one node",
			graph: "a",
			start: "a",
			want:  []string{"a"},
		},
		{
			name:  "no outrefs",
			graph: "a->b",
			start: "a",
			want:  []string{"a"},
		},
		{
			name:  "one hop",
			graph: "a->b",
			start: "b",
			want:  []string{"a", "b"},
		},
		{
			name:  "many hops",
			graph: "a->b->c->d",
			start: "c",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "fan out",
			graph: "a->b->c->d; e->c; f->a",
			start: "c",
			want:  []string{"a", "b", "c", "e", "f"},
		},
		{
			name:  "cycle 1",
			graph: "a->a",
			start: "a",
			want:  []string{"a"},
		},
		{
			name:  "cycle 3",
			graph: "a->b; b->c; c->a",
			start: "b",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "cycle 3",
			graph: "a->b; b->c; c->a",
			start: "b",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "complex",
			graph: "a->b->c; b->d; e->a; d->c",
			start: "b",
			want:  []string{"a", "b", "e"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := parseGraph(t, tc.graph)

			var startNode rnode.Node
			if tc.start == "" {
				// Create sentinel node.
				startID := fake.ID(project, meta.GlobalKey("sentinel"))
				nb := fake.NewBuilder(startID)
				var err error
				startNode, err = nb.Build()
				if err != nil {
					t.Fatal(err)
				}
			} else {
				startID := fake.ID(project, meta.GlobalKey(tc.start))
				startNode = g.Get(startID)
			}
			nodes, err := TransitiveInRefs(g, startNode)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("TransitiveInRefs() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}

			got := map[string]struct{}{}
			for _, n := range nodes {
				got[n.ID().String()] = struct{}{}
			}
			want := map[string]struct{}{}
			for _, w := range tc.want {
				want[fake.ID(project, meta.GlobalKey(w)).String()] = struct{}{}
			}

			if diff := cmp.Diff(got, want); diff != "" {
				t.Fatalf("Diff() -got+want: %s", diff)
			}
		})
	}
}
