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

package rnode

import (
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
)

// UntypedResource is the type-erased version of Resource.
type UntypedResource interface {
	ResourceID() *cloud.ResourceID
	Version() meta.Version
}

// Node in the resource graph.
type Node interface {
	// ID uniquely identifying this resource.
	ID() *cloud.ResourceID
	// State of the node.
	State() NodeState
	// Ownership of this resource.
	Ownership() OwnershipStatus
	// OutRefs of this resource pointing to other resources.
	OutRefs() []ResourceRef
	// InRefs pointing to this resource.
	InRefs() []ResourceRef
	// Resource is the cloud resource (e.g. the Resource[compute.Address,...]).
	Resource() UntypedResource
	// Builder returns a node builder that has the same attributes and
	// underlying type but has no contents in the resource. This is used to
	// populate a graph for getting the current state from Cloud (i.e. the "got"
	// graph).
	Builder() Builder
	// Diff this node (want) with the state of the Node (got). This computes
	// whether the Sync operation will be an update or recreation.
	Diff(got Node) (*PlanDetails, error)
	// Plan returns the plan for updating this Node.
	Plan() *Plan
	// Actions needed to perform the plan. This will be empty for graphs that
	// have not been planned. "got" is the current state of the Node in the
	// "got" graph.
	Actions(got Node) ([]exec.Action, error)
}

// NodeBase are common non-typed fields for implementing a Node in the graph.
type NodeBase struct {
	id        *cloud.ResourceID
	state     NodeState
	ownership OwnershipStatus
	outRefs   []ResourceRef
	inRefs    []ResourceRef
	plan      Plan
}

func (n *NodeBase) ID() *cloud.ResourceID      { return n.id }
func (n *NodeBase) State() NodeState           { return n.state }
func (n *NodeBase) Ownership() OwnershipStatus { return n.ownership }
func (n *NodeBase) OutRefs() []ResourceRef     { return n.outRefs }
func (n *NodeBase) InRefs() []ResourceRef      { return n.inRefs }
func (n *NodeBase) Plan() *Plan                { return &n.plan }

// InitFromBuilder is an rgraph library internal method for common
// initialization from a Builder.
func (n *NodeBase) InitFromBuilder(b Builder) error {
	n.id = b.ID()
	n.state = b.State()
	n.ownership = b.Ownership()
	outRefs, err := b.OutRefs()
	if err != nil {
		return err
	}
	n.outRefs = outRefs
	n.inRefs = b.inRefs()

	return nil
}
