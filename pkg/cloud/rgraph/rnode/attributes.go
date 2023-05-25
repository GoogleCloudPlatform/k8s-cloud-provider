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
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
)

// OwnershipStatus of the node in the graph.
type OwnershipStatus string

var (
	// OwnershipUnknown is the initial state of a Node.
	OwnershipUnknown OwnershipStatus = "Unknown"
	// OwnershipManaged means the Node's lifecycle and values are
	// to be planned and sync'd.
	OwnershipManaged OwnershipStatus = "Managed"
	// OwnershipExternal means the Node's lifecycle is not managed
	// by planning. The resource will not be mutated in any way
	// and is present in the graph for read-only purposes.
	OwnershipExternal OwnershipStatus = "External"
)

// NodeState is the state of the node in the Graph.
type NodeState string

const (
	// NodeUnknown is the initial state of the Node.
	NodeUnknown NodeState = "Unknown"
	// NodeExists means the resource exists in the graph.
	NodeExists NodeState = "Exists"
	// NodeDoesNotExist is a tombstone for a Node. It means that
	// the given Node should not exist in the Graph.
	NodeDoesNotExist NodeState = "DoesNotExist"
	// NodeStateError means that the resource could not be fetched
	// from the Cloud.
	NodeStateError NodeState = "Error"
)

// ResourceRef identifies a reference from the resource From in the field Path
// to the resource To.
type ResourceRef struct {
	// From is the object containing the reference.
	From *cloud.ResourceID
	// Path to the field with the value in From.
	Path api.Path
	// To is the resource that is referenced.
	To *cloud.ResourceID
}
