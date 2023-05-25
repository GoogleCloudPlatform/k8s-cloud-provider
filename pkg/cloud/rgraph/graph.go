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

func newGraph() *Graph {
	return &Graph{
		nodes: map[cloud.ResourceMapKey]rnode.Node{},
	}
}

// Graph of cloud resources. The set of nodes in the graph cannot change -- use
// the Builder to manipulate the set of resource nodes.
type Graph struct {
	nodes map[cloud.ResourceMapKey]rnode.Node
}

// All of the nodes in the Graph.
func (g *Graph) All() []rnode.Node {
	var ret []rnode.Node
	for _, n := range g.nodes {
		ret = append(ret, n)
	}
	return ret
}

// Get returns the Node named by id. Returns nil if the resource does not exist
// in the Graph.
func (g *Graph) Get(id *cloud.ResourceID) rnode.Node {
	return g.nodes[id.MapKey()]
}

// NewBuilderWithEmptyNodes creates a graph Builder with the same set of nodes
// but with no resource values. This is used to create a Builder that can be
// sync'ed with the cloud.
func (g *Graph) NewBuilderWithEmptyNodes() *Builder {
	builder := NewBuilder()
	for _, n := range g.nodes {
		builder.Add(n.Builder())
	}
	return builder
}

// AddTombstone adds a node to represent the non-existance of a resource.
func (g *Graph) AddTombstone(n rnode.Node) error {
	if n.State() != rnode.NodeDoesNotExist {
		return fmt.Errorf("graph: invalid tombstone (want state %s, but got %s)", rnode.NodeDoesNotExist, n.State())
	}
	g.nodes[n.ID().MapKey()] = n
	return nil
}

// add a note to the graph. This is package internal on purpose.
func (g *Graph) add(n rnode.Node) {
	g.nodes[n.ID().MapKey()] = n
}
