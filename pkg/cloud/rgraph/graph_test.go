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

package rgraph

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/fake"
	"github.com/google/go-cmp/cmp"
)

type topology struct {
	nodes map[string]struct{}
	edges edgeMap
}

type edgeMap map[string]map[string]struct{}

func addToEdgeMap(m edgeMap, a, b string) {
	if m[a] == nil {
		m[a] = map[string]struct{}{}
	}
	m[a][b] = struct{}{}
}

func parseTopology(s string) *topology {
	t := &topology{
		nodes: map[string]struct{}{},
		edges: edgeMap{},
	}
	runs := strings.Split(s, ";")
	for _, run := range runs {
		var prev string
		for _, n := range strings.Split(run, "->") {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			t.nodes[n] = struct{}{}
			if prev != "" {
				if t.edges[prev] == nil {
					t.edges[prev] = map[string]struct{}{}
				}
				t.edges[prev][n] = struct{}{}
			}
			prev = n
		}
	}
	return t
}

// This also tests Builder.
func TestGraph(t *testing.T) {
	ids := make([]*cloud.ResourceID, 10)
	for i := 0; i < len(ids); i++ {
		ids[i] = &cloud.ResourceID{Resource: "fake", Key: meta.GlobalKey(fmt.Sprintf("r%d", i))}
	}

	for _, tc := range []struct {
		name         string
		setup        func(b *Builder)
		topology     string
		wantBuildErr bool
	}{
		{
			name: "one node",
			setup: func(b *Builder) {
				b.Add(fake.NewBuilder(ids[0]))
				b.Get(ids[0]).SetOwnership(rnode.OwnershipManaged)
			},
			topology: "r0",
		},
		{
			name: "r0 -> r1",
			setup: func(b *Builder) {
				b0 := fake.NewBuilder(ids[0])
				b0.FakeOutRefs = append(b0.FakeOutRefs, rnode.ResourceRef{From: ids[0], To: ids[1]})
				b.Add(b0)
				b.Add(fake.NewBuilder(ids[1]))

				b.Get(ids[0]).SetOwnership(rnode.OwnershipManaged)
				b.Get(ids[1]).SetOwnership(rnode.OwnershipManaged)
			},
			topology: "r0 -> r1",
		},
		{
			name: "r0 -> r1; r0 -> r2",
			setup: func(b *Builder) {
				b0 := fake.NewBuilder(ids[0])
				b0.FakeOutRefs = append(b0.FakeOutRefs, rnode.ResourceRef{From: ids[0], To: ids[1]})
				b0.FakeOutRefs = append(b0.FakeOutRefs, rnode.ResourceRef{From: ids[0], To: ids[2]})
				b.Add(b0)

				b.Add(fake.NewBuilder(ids[1]))
				b.Add(fake.NewBuilder(ids[2]))

				b.Get(ids[0]).SetOwnership(rnode.OwnershipManaged)
				b.Get(ids[1]).SetOwnership(rnode.OwnershipManaged)
				b.Get(ids[2]).SetOwnership(rnode.OwnershipManaged)
			},
			topology: "r0 -> r1; r0 -> r2",
		},
		{
			name: "diamond",
			setup: func(b *Builder) {
				b0 := fake.NewBuilder(ids[0])
				b0.FakeOutRefs = append(b0.FakeOutRefs, rnode.ResourceRef{From: ids[0], To: ids[1]})
				b0.FakeOutRefs = append(b0.FakeOutRefs, rnode.ResourceRef{From: ids[0], To: ids[2]})
				b.Add(b0)

				b1 := fake.NewBuilder(ids[1])
				b1.FakeOutRefs = append(b0.FakeOutRefs, rnode.ResourceRef{From: ids[1], To: ids[3]})
				b.Add(b1)

				b2 := fake.NewBuilder(ids[2])
				b2.FakeOutRefs = append(b0.FakeOutRefs, rnode.ResourceRef{From: ids[2], To: ids[3]})
				b.Add(b2)

				b.Add(fake.NewBuilder(ids[3]))

				b.Get(ids[0]).SetOwnership(rnode.OwnershipManaged)
				b.Get(ids[1]).SetOwnership(rnode.OwnershipManaged)
				b.Get(ids[2]).SetOwnership(rnode.OwnershipManaged)
				b.Get(ids[3]).SetOwnership(rnode.OwnershipManaged)
			},
			topology: "r0 -> r1; r0 -> r2; r1 -> r3; r2 -> r3",
		},
		{
			name: "unknown ownership",
			setup: func(b *Builder) {
				b.Add(fake.NewBuilder(ids[0]))
			},
			topology:     "r0",
			wantBuildErr: true,
		},
		{
			name: "points to object not in graph",
			setup: func(b *Builder) {
				b0 := fake.NewBuilder(ids[0])
				b0.FakeOutRefs = append(b0.FakeOutRefs, rnode.ResourceRef{From: ids[0], To: ids[1]})
				b.Add(b0)
				b.Get(ids[0]).SetOwnership(rnode.OwnershipManaged)
			},
			topology:     "r0 -> r1",
			wantBuildErr: true,
		},
		{
			name: "outRef parse errors",
			setup: func(b *Builder) {
				b0 := fake.NewBuilder(ids[0])
				b0.FakeOutRefs = append(b0.FakeOutRefs, rnode.ResourceRef{From: ids[0], To: ids[1]})
				b0.OutRefsErr = errors.New("injected")
				b.Add(b0)
				b.Add(fake.NewBuilder(ids[1]))

				b.Get(ids[0]).SetOwnership(rnode.OwnershipManaged)
				b.Get(ids[1]).SetOwnership(rnode.OwnershipManaged)
			},
			topology:     "r0 -> r1",
			wantBuildErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := NewBuilder()
			tc.setup(b)

			topo := parseTopology(tc.topology)
			t.Logf("topo = %+v", topo)

			gotNodes := map[string]struct{}{}
			gotEdges := edgeMap{}

			for _, n := range b.All() {
				nget := b.Get(n.ID())
				if nget == nil {
					t.Errorf("b.Get(%s) = nil, want non-nil", n.ID())
				}
				gotNodes[n.ID().Key.Name] = struct{}{}
				refs, err := n.OutRefs()
				if err == nil {
					for _, ref := range refs {
						from := ref.From.Key.Name
						to := ref.To.Key.Name
						t.Logf("edge %s -> %s", from, to)
						addToEdgeMap(gotEdges, from, to)
					}
				}
			}

			g, err := b.Build()
			if err == nil {
				b.MustBuild()
			}

			gotErr := err != nil
			if gotErr != tc.wantBuildErr {
				t.Fatalf("b.Build() = %v, gotErr = %t, want %v", err, gotErr, tc.wantBuildErr)
			}
			if gotErr {
				return
			}

			if diff := cmp.Diff(gotNodes, topo.nodes); diff != "" {
				t.Errorf("Diff(gotNodes, topo.nodes) = -got,+want: %s", diff)
			}
			if diff := cmp.Diff(gotEdges, topo.edges); diff != "" {
				t.Errorf("Diff(gotEdges, topo.edges) = -got,+want: %s", diff)
			}

			gotNodes = map[string]struct{}{}
			gotInEdges := edgeMap{}
			gotOutEdges := edgeMap{}

			for _, n := range g.All() {
				nget := g.Get(n.ID())
				if nget == nil {
					t.Errorf("g.Get(%s) = nil, want non-nil", n.ID())
				}
				gotNodes[n.ID().Key.Name] = struct{}{}
				for _, ref := range n.InRefs() {
					from := ref.From.Key.Name
					to := ref.To.Key.Name
					t.Logf("in edge %s -> %s", from, to)
					addToEdgeMap(gotInEdges, from, to)
				}
				for _, ref := range n.OutRefs() {
					from := ref.From.Key.Name
					to := ref.To.Key.Name
					t.Logf("out edge %s -> %s", from, to)
					addToEdgeMap(gotOutEdges, from, to)
				}
			}
			if diff := cmp.Diff(gotNodes, topo.nodes); diff != "" {
				t.Errorf("Diff(gotNodes, topo.nodes) = -got,+want: %s", diff)
			}
			if diff := cmp.Diff(gotInEdges, topo.edges); diff != "" {
				t.Errorf("Diff(gotInEdges, topo.edges) = -got,+want: %s", diff)
			}
			if diff := cmp.Diff(gotOutEdges, topo.edges); diff != "" {
				t.Errorf("Diff(gotOutEdges, topo.edges) = -got,+want: %s", diff)
			}
		})
	}
}

func TestGraphNewBuilder(t *testing.T) {
	ids := make([]*cloud.ResourceID, 10)
	for i := 0; i < len(ids); i++ {
		ids[i] = &cloud.ResourceID{Resource: "fake", Key: meta.GlobalKey(fmt.Sprintf("r%d", i))}
	}

	b := NewBuilder()
	b0 := fake.NewBuilder(ids[0])
	b0.FakeOutRefs = append(b0.FakeOutRefs, rnode.ResourceRef{From: ids[0], To: ids[1]})
	b.Add(b0)
	b.Add(fake.NewBuilder(ids[1]))

	b.Get(ids[0]).SetOwnership(rnode.OwnershipManaged)
	b.Get(ids[1]).SetOwnership(rnode.OwnershipManaged)

	g := b.MustBuild()
	b = g.NewBuilderWithEmptyNodes()

	got := map[string]struct{}{}
	for _, n := range g.All() {
		got[n.ID().Key.Name] = struct{}{}
	}
	if diff := cmp.Diff(got, map[string]struct{}{
		"r0": {},
		"r1": {},
	}); diff != "" {
		t.Errorf("Diff() -got,+want: %s", diff)
	}
}

func TestGraphAddTombstone(t *testing.T) {
	ids := make([]*cloud.ResourceID, 10)
	for i := 0; i < len(ids); i++ {
		ids[i] = &cloud.ResourceID{Resource: "fake", Key: meta.GlobalKey(fmt.Sprintf("r%d", i))}
	}

	b := NewBuilder()
	b0 := fake.NewBuilder(ids[0])
	b0.FakeOutRefs = append(b0.FakeOutRefs, rnode.ResourceRef{From: ids[0], To: ids[1]})
	b.Add(b0)
	b.Add(fake.NewBuilder(ids[1]))

	b.Get(ids[0]).SetOwnership(rnode.OwnershipManaged)
	b.Get(ids[1]).SetOwnership(rnode.OwnershipManaged)

	g := b.MustBuild()
	tombstoneb := fake.NewBuilder(ids[2])
	tombstoneb.SetState(rnode.NodeDoesNotExist)
	tombstone, err := tombstoneb.Build()
	if err != nil {
		t.Fatalf("tombstoneb.Build() = %v", err)
	}
	if err := g.AddTombstone(tombstone); err != nil {
		t.Fatalf("g.AddTombstone() = %v, want nil", err)
	}

	got := map[string]struct{}{}
	for _, n := range g.All() {
		got[n.ID().Key.Name] = struct{}{}
	}
	if diff := cmp.Diff(got, map[string]struct{}{
		"r0": {},
		"r1": {},
		"r2": {},
	}); diff != "" {
		t.Errorf("Diff() -got,+want: %s", diff)
	}

	// Cannot add non-tombstone Nodes.
	tombstoneb.SetState(rnode.NodeExists)
	tombstone, _ = tombstoneb.Build()
	if err := g.AddTombstone(tombstone); err == nil {
		t.Fatalf("g.AddTombstone() = nil, want error")
	}
}
