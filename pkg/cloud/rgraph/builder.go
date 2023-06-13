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
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
)

const (
	builderErrPrefix = "Builder"
)

// NewBuilder returns a new Graph Builder.
func NewBuilder() *Builder {
	return &Builder{
		nodes: map[cloud.ResourceMapKey]rnode.Builder{},
	}
}

// Builder builds resource Graphs.
type Builder struct {
	nodes map[cloud.ResourceMapKey]rnode.Builder
}

func (g *Builder) All() []rnode.Builder {
	var ret []rnode.Builder
	for _, nb := range g.nodes {
		ret = append(ret, nb)
	}
	return ret
}

// Add a node to the resource graph.
func (g *Builder) Add(node rnode.Builder) { g.nodes[node.ID().MapKey()] = node }

// Get the node named by id from the graph. Returns nil if the node does not
// exist.
func (g *Builder) Get(id *cloud.ResourceID) rnode.Builder { return g.nodes[id.MapKey()] }

// Build a Graph for planning from the nodes.
func (g *Builder) Build() (*Graph, error) {
	if err := g.computeInRefs(); err != nil {
		return nil, err
	}
	if err := g.validate(); err != nil {
		return nil, err
	}

	newGraph := newGraph()
	for _, nb := range g.nodes {
		newNode, err := nb.Build()
		if err != nil {
			return nil, err
		}
		newGraph.add(newNode)
	}

	return newGraph, nil
}

// MustBuild panics if the Graph cannot be built. This should ONLY be used in
// unit tests.
func (g *Builder) MustBuild() *Graph {
	ret, err := g.Build()
	if err != nil {
		panic(fmt.Sprintf("MustBuild: %v", err))
	}
	return ret
}

// computeInRefs calculates the inbound references to a resource from all of the
// nodes in the graph.
func (g *Builder) computeInRefs() error {
	for _, fromNode := range g.nodes {
		refs, err := fromNode.OutRefs()
		if err != nil {
			return fmt.Errorf("computeInRefs: %w", err)
		}
		for _, ref := range refs {
			toNode, ok := g.nodes[ref.To.MapKey()]
			if !ok {
				return fmt.Errorf("%s: missing outRef: %s points to %s which isn't in the graph", builderErrPrefix, fromNode.ID(), ref.To)
			}
			toNode.AddInRef(ref)
		}
	}
	return nil
}

// validate the graph.
func (g *Builder) validate() error {
	for _, n := range g.nodes {
		// No nodes have OwnershipUnknown
		if n.Ownership() == rnode.OwnershipUnknown {
			return fmt.Errorf("%s: node %s has ownership %s", builderErrPrefix, n.ID(), n.Ownership())
		}
		// ResourceID is not mismatched
		resource := n.Resource()
		if resource != nil && !resource.ResourceID().Equal(n.ID()) {
			return fmt.Errorf("%s: node and resource id mismatch (node=%v, id=%v)", builderErrPrefix, n.ID(), resource.ResourceID())
		}
	}
	// All resources have their dependencies in the graph if they are OwnershipManaged.
	for _, n := range g.nodes {
		if n.Ownership() != rnode.OwnershipManaged {
			continue
		}
		deps, err := n.OutRefs()
		if err != nil {
			return err
		}
		for _, d := range deps {
			if _, ok := g.nodes[d.To.MapKey()]; !ok {
				return fmt.Errorf("%s: missing outRef: %v points to %v which isn't in the graph", builderErrPrefix, n.ID(), d.To)
			}
		}
	}

	return nil
}
