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

package networkendpointgroup

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

func NewBuilder(id *cloud.ResourceID) rnode.Builder {
	b := &builder{}
	b.Defaults(id)
	return b
}

func NewBuilderWithResource(r NetworkEndpointGroup) rnode.Builder {
	b := &builder{resource: r}
	b.Init(r.ResourceID(), rnode.NodeUnknown, rnode.OwnershipUnknown, r)
	return b
}

type builder struct {
	rnode.BuilderBase
	resource NetworkEndpointGroup
}

// builder implements node.Builder.
var _ rnode.Builder = (*builder)(nil)

func (b *builder) Resource() rnode.UntypedResource { return b.resource }

func (b *builder) SetResource(u rnode.UntypedResource) error {
	r, ok := u.(NetworkEndpointGroup)
	if !ok {
		return fmt.Errorf("XXX")
	}
	b.resource = r
	return nil
}

func (b *builder) SyncFromCloud(ctx context.Context, gcp cloud.Cloud) error {
	return rnode.GenericGet[compute.NetworkEndpointGroup, alpha.NetworkEndpointGroup, beta.NetworkEndpointGroup](
		ctx, gcp, "NetworkEndpointGroup", &ops{}, &typeTrait{}, b)
}

func (b *builder) OutRefs() ([]rnode.ResourceRef, error) {
	// No references.
	return nil, nil
}

func (b *builder) Build() (rnode.Node, error) {
	if b.State() == rnode.NodeExists && b.resource == nil {
		return nil, fmt.Errorf("NetworkEndpointGroup %s resource is nil with state %s", b.ID(), b.State())
	}

	ret := &networkEndpointGroupNode{resource: b.resource}
	if err := ret.InitFromBuilder(b); err != nil {
		return nil, err
	}

	return ret, nil
}
