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

package address

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/all"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

func init() { all.RegisterBuilder(resourcePlural, NewBuilder) }

func NewBuilder(id *cloud.ResourceID) rnode.Builder {
	b := &builder{}
	b.BuilderBase.Defaults(id)
	return b
}

func NewBuilderWithResource(r Address) rnode.Builder {
	b := &builder{resource: r}
	b.Init(r.ResourceID(), rnode.NodeExists, rnode.OwnershipUnknown, r)
	return b
}

// Unimplemented fields and methods:
// - .Labels, setLabels(). Impacts CreateAction, Diff and Update.

type builder struct {
	rnode.BuilderBase
	resource Address
}

// builder implements node.Builder.
var _ rnode.Builder = (*builder)(nil)

func (b *builder) Resource() rnode.UntypedResource { return b.resource }

func (b *builder) SetResource(u rnode.UntypedResource) error {
	r, ok := u.(Address)
	if !ok {
		return fmt.Errorf("XXX")
	}
	b.resource = r
	return nil
}

func (b *builder) SyncFromCloud(ctx context.Context, gcp cloud.Cloud) error {
	return rnode.GenericGet[compute.Address, alpha.Address, beta.Address](ctx, gcp, "Address", &ops{}, &typeTrait{}, b)
}

func (b *builder) OutRefs() ([]rnode.ResourceRef, error) {
	// Address does not have any outgoing resource references.
	return nil, nil
}

func (b *builder) Build() (rnode.Node, error) {
	if b.State() == rnode.NodeExists && b.resource == nil {
		return nil, fmt.Errorf("Address %s resource is nil with state %s", b.ID(), b.State())
	}

	ret := &addressNode{resource: b.resource}
	if err := ret.InitFromBuilder(b); err != nil {
		return nil, err
	}

	return ret, nil
}
