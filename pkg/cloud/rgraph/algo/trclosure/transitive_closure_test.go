package trclosure

import (
	"context"
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/fake"
	"github.com/google/go-cmp/cmp"
	"k8s.io/klog/v2"
)

func TestTransitiveClosure(t *testing.T) {
	// No t.Parallel() due to use of fake.Mocks.Add().
	const project = "proj1"
	mockCloud := cloud.NewMockGCE(nil) // TODO: project router

	addNode := func(from string, toList []string, opts ...func(*fake.Builder)) *fake.Builder {
		id := fake.ID(project, meta.GlobalKey(from))
		ret := fake.NewBuilder(id)
		ret.SetOwnership(rnode.OwnershipManaged)
		ret.SetState(rnode.NodeExists)

		for _, opt := range opts {
			opt(ret)
		}
		for _, to := range toList {
			ret.FakeOutRefs = append(ret.FakeOutRefs, rnode.ResourceRef{
				From: ret.ID(),
				To:   fake.ID(project, meta.GlobalKey(to)),
			})
		}
		if fake.Mocks.Add(ret) {
			// We panic here to catch buggy test setup early.
			panic(fmt.Sprintf("duplicate fake.Mocks.Add(%s)", ret.ID()))
		}
		klog.Infof("fake.Mocks.Add(%s)", ret.ID())
		return ret
	}
	addNodeToGraph := func(g *rgraph.Builder, from string, toList []string, opts ...func(*fake.Builder)) {
		fakeNode := addNode(from, toList, opts...)
		g.Add(fakeNode)
	}

	type wantGraph struct {
		ids []string
	}

	checkGraph := func(t *testing.T, g *rgraph.Builder, wg *wantGraph) {
		t.Helper()

		got := map[string]bool{}
		want := map[string]bool{}

		// TODO: enhance the kinds of things that are being checked.
		for _, n := range g.All() {
			got[n.ID().String()] = true
		}
		for _, id := range wg.ids {
			fid := fake.ID(project, meta.GlobalKey(id))
			want[fid.String()] = true
		}

		if diff := cmp.Diff(got, want); diff != "" {
			t.Fatalf("checkGraph: -got,+want: %s", diff)
		}
	}

	nodeIsExternal := func(b *fake.Builder) { b.SetOwnership(rnode.OwnershipExternal) }

	for _, tc := range []struct {
		name    string
		cloud   cloud.Cloud
		graph   func() *rgraph.Builder
		want    wantGraph
		wantErr bool
	}{
		{
			name:  "empty graph",
			cloud: mockCloud,
			graph: func() *rgraph.Builder { return rgraph.NewBuilder() },
		},
		{
			name:  "single node",
			cloud: mockCloud,
			graph: func() *rgraph.Builder {
				g := rgraph.NewBuilder()
				addNodeToGraph(g, "a", nil)
				return g
			},
			want: wantGraph{ids: []string{"a"}},
		},
		{
			name:  "singleton nodes",
			cloud: mockCloud,
			graph: func() *rgraph.Builder {
				g := rgraph.NewBuilder()
				addNodeToGraph(g, "a", nil)
				addNodeToGraph(g, "b", nil)
				return g
			},
			want: wantGraph{ids: []string{"a", "b"}},
		},
		// Notation: [x] means that x is a starting node in the graph.
		{
			// Note: this is impossible, but we handle this case.
			name:  "[a] -> a",
			cloud: mockCloud,
			graph: func() *rgraph.Builder {
				g := rgraph.NewBuilder()
				addNodeToGraph(g, "a", []string{"a"})
				return g
			},
			want: wantGraph{ids: []string{"a"}},
		},
		{
			name:  "[a] -> b",
			cloud: mockCloud,
			graph: func() *rgraph.Builder {
				g := rgraph.NewBuilder()
				addNodeToGraph(g, "a", []string{"b"})
				return g
			},
			want: wantGraph{ids: []string{"a", "b"}},
		},
		{
			name:  "[a] -> b; [a] -> c",
			cloud: mockCloud,
			graph: func() *rgraph.Builder {
				g := rgraph.NewBuilder()
				addNodeToGraph(g, "a", []string{"b", "c"})
				return g
			},
			want: wantGraph{ids: []string{"a", "b", "c"}},
		},
		{
			name:  "[a] -> b -> c",
			cloud: mockCloud,
			graph: func() *rgraph.Builder {
				g := rgraph.NewBuilder()
				addNodeToGraph(g, "a", []string{"b"})
				addNode("b", []string{"c"})
				return g
			},
			want: wantGraph{ids: []string{"a", "b", "c"}},
		},
		{
			name:  "[a] -> b; [a] -> c -> d",
			cloud: mockCloud,
			graph: func() *rgraph.Builder {
				g := rgraph.NewBuilder()
				addNodeToGraph(g, "a", []string{"b", "c"})
				addNode("c", []string{"d"})
				return g
			},
			want: wantGraph{ids: []string{"a", "b", "c", "d"}},
		},
		{
			name:  "diamond ([a]->b;[a]->c;b->d;c->d)",
			cloud: mockCloud,
			graph: func() *rgraph.Builder {
				g := rgraph.NewBuilder()
				addNodeToGraph(g, "a", []string{"b", "c"})
				addNode("b", []string{"d"})
				addNode("c", []string{"d"})
				return g
			},
			want: wantGraph{ids: []string{"a", "b", "c", "d"}},
		},
		{
			name:  "complex graph 1 ([a]->b->c->d;c->a;b->d)",
			cloud: mockCloud,
			graph: func() *rgraph.Builder {
				g := rgraph.NewBuilder()
				addNodeToGraph(g, "a", []string{"b"})
				addNode("b", []string{"c", "d"})
				addNode("c", []string{"a", "d"})
				addNode("d", []string{})
				return g
			},
			want: wantGraph{ids: []string{"a", "b", "c", "d"}},
		},
		{
			name:  "don't traverse external nodes",
			cloud: mockCloud,
			graph: func() *rgraph.Builder {
				g := rgraph.NewBuilder()
				addNodeToGraph(g, "a", []string{"b"})
				addNode("b", []string{"c"}, nodeIsExternal)
				return g
			},
			want: wantGraph{ids: []string{"a", "b"}}, // "c" should not be traversed.
		},
		{
			name:  "don't traverse external nodes (diamond)",
			cloud: mockCloud,
			graph: func() *rgraph.Builder {
				g := rgraph.NewBuilder()
				addNodeToGraph(g, "a", []string{"b", "c"})
				addNode("b", []string{"d", "e"}, nodeIsExternal)
				addNode("c", []string{"d"}) // "d" should be traversed because of "c->d".
				addNode("d", []string{})
				addNode("e", []string{}) // "e" should not be traversed.
				return g
			},
			want: wantGraph{ids: []string{"a", "b", "c", "d"}},
		},
		{
			name:  "don't traverse external nodes (complex)",
			cloud: mockCloud,
			graph: func() *rgraph.Builder {
				g := rgraph.NewBuilder()
				addNodeToGraph(g, "a", []string{"b", "c"})
				addNode("b", []string{"d", "e"}, nodeIsExternal)
				addNode("c", []string{"f"})
				addNode("d", []string{})
				addNode("e", []string{})
				addNode("f", []string{})
				return g
			},
			want: wantGraph{ids: []string{"a", "b", "c", "f"}},
		},
		{
			name:  "don't traverse external nodes (cycles)",
			cloud: mockCloud,
			graph: func() *rgraph.Builder {
				g := rgraph.NewBuilder()
				addNodeToGraph(g, "a", []string{"b", "c"})
				addNode("b", []string{"d", "e"}, nodeIsExternal)
				addNode("d", []string{"a"})
				addNode("e", []string{"b"})
				return g
			},
			want: wantGraph{ids: []string{"a", "b", "c"}},
		},
		// TODO(bowei): add error test cases.
	} {
		t.Run(tc.name, func(t *testing.T) {
			// No t.Parallel() due to use of fake.Mocks.Add().
			fake.Mocks.Clear()
			g := tc.graph()
			err := Do(context.Background(), tc.cloud, g)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Fatalf("Do() = %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
			if gotErr {
				return
			}
			checkGraph(t, g, &tc.want)
		})
	}
}
