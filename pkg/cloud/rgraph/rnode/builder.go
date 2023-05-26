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
	"context"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
)

// Builder is a Node in the graph Builder.
type Builder interface {
	// ID uniquely identifying this resource.
	ID() *cloud.ResourceID

	// State of the node.
	State() NodeState
	// SetState of the node.
	SetState(state NodeState)

	// Ownership of this resource.
	Ownership() OwnershipStatus
	// SetOwnership of this resource.
	SetOwnership(os OwnershipStatus)

	// Resource (cloud type) for this Node.
	Resource() UntypedResource
	// SetResource to a new value.
	SetResource(UntypedResource) error

	// Version of the resource. This is used when fetching the
	// resource from the Cloud.
	Version() meta.Version

	// OutRefs parses the outgoing references of the Resource.
	OutRefs() ([]ResourceRef, error)
	// AddInRef to this node Builder.
	AddInRef(ref ResourceRef)

	// SyncFromCloud downloads the resource from the Cloud. This
	// may result in one or more blocking calls to the GCE APIs.
	SyncFromCloud(ctx context.Context, cl cloud.Cloud) error

	// Build the node, converting this to a Node in a Graph.
	Build() (Node, error)

	// inRefs that have been computed so far. This method is
	// package private; this value is not accurate until it has
	// been computed from a complete set of nodes in the graph
	// Builder.
	inRefs() []ResourceRef
}

// BuilderBase implements the non-type specific fields.
type BuilderBase struct {
	id        *cloud.ResourceID
	state     NodeState
	ownership OwnershipStatus
	version   meta.Version

	curInRefs []ResourceRef
}

func (b *BuilderBase) ID() *cloud.ResourceID           { return b.id }
func (b *BuilderBase) State() NodeState                { return b.state }
func (b *BuilderBase) SetState(state NodeState)        { b.state = state }
func (b *BuilderBase) Ownership() OwnershipStatus      { return b.ownership }
func (b *BuilderBase) SetOwnership(os OwnershipStatus) { b.ownership = os }
func (b *BuilderBase) Version() meta.Version           { return b.version }

func (b *BuilderBase) AddInRef(ref ResourceRef) { b.curInRefs = append(b.curInRefs, ref) }
func (b *BuilderBase) inRefs() []ResourceRef    { return b.curInRefs }

// Defaults sets the default values for a empty Builder node.
func (b *BuilderBase) Defaults(id *cloud.ResourceID) {
	b.id = id
	b.state = NodeUnknown
	b.ownership = OwnershipUnknown
	b.version = meta.VersionGA
}

// Init the values of the BuilderBase.
func (b *BuilderBase) Init(
	id *cloud.ResourceID,
	state NodeState,
	ownership OwnershipStatus,
	resource UntypedResource,
) {
	b.id = id
	b.state = state
	b.ownership = ownership
	if resource == nil {
		b.version = meta.VersionGA
	} else {
		b.version = resource.Version()
	}
}
