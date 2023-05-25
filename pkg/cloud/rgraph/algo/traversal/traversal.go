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
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/algo"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
)

// ConnectedSubgraph returns the set of Nodes (inclusive of starting node) that
// are connected by inbound (InRef) and outbound (OutRef) references.
func ConnectedSubgraph(g *rgraph.Graph, n rnode.Node) ([]rnode.Node, error) {
	if g.Get(n.ID()) == nil {
		return nil, fmt.Errorf("starting node %s not in graph", n.ID())
	}

	done := map[cloud.ResourceMapKey]rnode.Node{}

	var work algo.Queue[rnode.Node]
	work.Add(n)

	for !work.Empty() {
		cur := work.Pop()
		done[cur.ID().MapKey()] = cur

		refs := cur.OutRefs()
		for _, ref := range refs {
			if _, ok := done[ref.To.MapKey()]; ok {
				continue
			}
			to := g.Get(ref.To)
			if to == nil {
				return nil, fmt.Errorf("invalid graph: to node %s not in graph", to.ID())
			}
			work.Add(to)
		}
		refs = cur.InRefs()
		for _, ref := range refs {
			if _, ok := done[ref.From.MapKey()]; ok {
				continue
			}
			from := g.Get(ref.From)
			if from == nil {
				return nil, fmt.Errorf("invalid graph: from node %s not in graph", from.ID())
			}
			work.Add(from)
		}
	}

	var ret []rnode.Node
	for _, node := range done {
		ret = append(ret, node)
	}

	return ret, nil
}

// TransitiveInRefs returns the set of Nodes (inclusive of the starting node)
// that point into the node. For example, for graph A => B => C; D => B, this
// will return [A, B, D] for B.
func TransitiveInRefs(g *rgraph.Graph, n rnode.Node) ([]rnode.Node, error) {
	if g.Get(n.ID()) == nil {
		return nil, fmt.Errorf("starting node %s not in graph", n.ID())
	}

	var work algo.Queue[rnode.Node]
	work.Add(n)

	done := map[cloud.ResourceMapKey]rnode.Node{}

	for !work.Empty() {
		cur := work.Pop()
		done[cur.ID().MapKey()] = cur

		refs := cur.InRefs()
		for _, ref := range refs {
			if _, ok := done[ref.From.MapKey()]; ok {
				continue
			}
			from := g.Get(ref.From)
			if from == nil {
				return nil, fmt.Errorf("invalid graph: from node %s not in graph", from.ID())
			}
			work.Add(from)
		}
	}

	var ret []rnode.Node
	for _, node := range done {
		ret = append(ret, node)
	}

	return ret, nil
}
