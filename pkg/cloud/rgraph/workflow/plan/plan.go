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

package plan

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/algo/actions"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/algo/localplan"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/algo/traversal"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/algo/trclosure"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
)

type Result struct {
	Got     *rgraph.Graph
	Want    *rgraph.Graph
	Actions []exec.Action
}

// Do will plan updates to cloud resources wanted in graph. Returns the set of
// Actions needed to sync to "want".
func Do(ctx context.Context, c cloud.Cloud, want *rgraph.Graph) (*Result, error) {
	w := planner{
		cloud: c,
		want:  want,
	}
	return w.plan(ctx)
}

const errPrefix = "Plan"

type planner struct {
	cloud cloud.Cloud
	got   *rgraph.Graph
	want  *rgraph.Graph
}

func (pl *planner) plan(ctx context.Context) (*Result, error) {
	// Assemble the "got" graph. This will get the current state of any
	// resources and also enumerate any resouces that are currently linked that
	// are not in the "want" graph.
	gotBuilder := pl.want.NewBuilderWithEmptyNodes()

	// Fetch the current resource graph from Cloud.
	// TODO: resource_prefix, ownership due to prefix etc.
	err := trclosure.Do(ctx, pl.cloud, gotBuilder,
		trclosure.OnGetFunc(func(n rnode.Builder) error {
			n.SetOwnership(rnode.OwnershipManaged)
			return nil
		}),
	)
	if err != nil {
		return nil, err
	}

	pl.got, err = gotBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}

	// Figure out what to do with Nodes in "got" that aren't in "want". These
	// are resources that will no longer by referenced in the updated graph.
	for _, gotNode := range pl.got.All() {
		switch {
		case pl.want.Get(gotNode.ID()) != nil:
			// Node exists in "want", don't need to do anything.
		case gotNode.Ownership() == rnode.OwnershipExternal:
			// TODO: clone the node from the "got" graph for "want" unchanged.
		case gotNode.Ownership() == rnode.OwnershipManaged:
			// Nodes that are no longer referenced should be deleted.
			wantNodeBuilder := gotNode.Builder()
			wantNodeBuilder.SetState(rnode.NodeDoesNotExist)
			wantNode, err := wantNodeBuilder.Build()
			if err != nil {
				return nil, err
			}
			err = pl.want.AddTombstone(wantNode)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("%s: node %s has invalid ownership %s", errPrefix, gotNode.ID(), gotNode.Ownership())
		}
	}

	// Compute the local plan for each resource.
	if err := localplan.PlanWantGraph(pl.got, pl.want); err != nil {
		return nil, err
	}

	if err := pl.propagateRecreates(); err != nil {
		return nil, err
	}

	if err := pl.sanityCheck(); err != nil {
		return nil, err
	}

	acts, err := actions.Do(pl.got, pl.want)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}
	return &Result{
		Got:     pl.got,
		Want:    pl.want,
		Actions: acts,
	}, nil
}

// propagateRecreates through inbound references. If a resource needs to be
// recreated, this means any references will also be affected transitively.
func (pl *planner) propagateRecreates() error {
	var recreateNodes []rnode.Node
	for _, n := range pl.want.All() {
		if n.Plan().Op() == rnode.OpRecreate {
			recreateNodes = append(recreateNodes, n)
		}
	}

	done := map[cloud.ResourceMapKey]bool{}
	for _, n := range recreateNodes {
		done[n.ID().MapKey()] = true

		inRefNodes, err := traversal.TransitiveInRefs(pl.want, n)
		if err != nil {
			return err
		}

		for _, inRefNode := range inRefNodes {
			if done[inRefNode.ID().MapKey()] {
				continue
			}
			done[inRefNode.ID().MapKey()] = true

			if inRefNode.Ownership() != rnode.OwnershipManaged {
				return fmt.Errorf("%s: %v planned for recreate, but inRef %v ownership=%s", errPrefix, n.ID(), inRefNode.ID(), inRefNode.Ownership())
			}

			switch inRefNode.Plan().Op() {
			case rnode.OpCreate, rnode.OpRecreate, rnode.OpDelete:
				// Resource is already being created or destroy.
			case rnode.OpNothing, rnode.OpUpdate:
				inRefNode.Plan().Set(rnode.PlanDetails{
					Operation: rnode.OpRecreate,
					Why:       fmt.Sprintf("Dependency %v is being recreated", n.ID()),
				})
			default:
				return fmt.Errorf("%s: inRef %s has invalid op %s, can't propagate recreate", errPrefix, inRefNode.ID(), inRefNode.Plan().Op())
			}
		}
	}
	return nil
}

func (pl *planner) sanityCheck() error {
	for _, n := range pl.want.All() {
		switch n.Plan().Op() {
		case rnode.OpUnknown:
			return fmt.Errorf("%s: node %v has invalid op %s", errPrefix, n.ID(), n.Plan().Op())
		case rnode.OpDelete:
			// If A => B; if B is to be deleted, then A must be deleted.
			for _, refs := range n.InRefs() {
				if inNode := pl.want.Get(refs.To); inNode == nil {
					return fmt.Errorf("%s: inRef from node %v that doesn't exist", errPrefix, refs.From)
				} else if inNode.Plan().Op() != rnode.OpDelete {
					return fmt.Errorf("%s: %v to be deleted, but inRef %v is not", errPrefix, n.ID(), inNode.ID())
				}
			}
		}
	}

	return nil
}
