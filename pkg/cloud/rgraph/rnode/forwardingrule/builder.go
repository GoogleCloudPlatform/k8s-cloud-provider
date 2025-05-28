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

package forwardingrule

import (
	"context"
	"fmt"
	"net"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/all"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

func init() { all.RegisterBuilder(resourcePlural, NewBuilder) }

func NewBuilder(id *cloud.ResourceID) rnode.Builder {
	b := &builder{}
	b.Defaults(id)
	return b
}

func NewBuilderWithResource(r Resource) rnode.Builder {
	b := &builder{resource: r}
	b.Init(r.ResourceID(), rnode.NodeUnknown, rnode.OwnershipUnknown, r)
	return b
}

type builder struct {
	rnode.BuilderBase
	resource Resource
}

// builder implements node.Builder.
var _ rnode.Builder = (*builder)(nil)

func (b *builder) Resource() rnode.UntypedResource { return b.resource }

func (b *builder) SetResource(u rnode.UntypedResource) error {
	r, ok := u.(Resource)
	if !ok {
		return fmt.Errorf("SetResource: invalid type: %T, want ForwardingRule", u)
	}
	b.resource = r
	return nil
}

func (b *builder) SyncFromCloud(ctx context.Context, gcp cloud.Cloud) error {
	return rnode.GenericGet[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule](
		ctx, gcp, "ForwardingRule", &ops{}, &TypeTrait{}, b)
}

func (b *builder) OutRefs() ([]rnode.ResourceRef, error) {
	if b.resource == nil {
		return nil, nil
	}

	var ret []rnode.ResourceRef
	// Ignore conversion errors as the fields we care about are all available in GA.
	obj, _ := b.resource.ToGA()

	// IPAddress
	if obj.IPAddress != "" {
		if ip := net.ParseIP(obj.IPAddress); ip != nil {
			// Numeric IP address. This is an emphemeral address that does't
			// have a resource associated with it.
		} else {
			id, err := cloud.ParseResourceURL(obj.IPAddress)
			if err != nil {
				return nil, fmt.Errorf("ForwardingRuleNode IPAddress: %w", err)
			}
			ret = append(ret, rnode.ResourceRef{
				From: b.resource.ResourceID(),
				Path: api.Path{}.Pointer().Field("IPAddress"),
				To:   id,
			})
		}
	}

	// .BackendService, .Target
	for _, fieldSpec := range []struct {
		name string
		val  string
	}{
		{"BackendService", obj.BackendService},
		{"Target", obj.Target},
	} {
		if fieldSpec.val == "" {
			continue
		}
		id, err := cloud.ParseResourceURL(fieldSpec.val)
		if err != nil {
			return nil, fmt.Errorf("ForwardingRuleNode %s: %w", fieldSpec.name, err)
		}
		ret = append(ret, rnode.ResourceRef{
			From: b.resource.ResourceID(),
			Path: api.Path{}.Pointer().Field(fieldSpec.name),
			To:   id,
		})
	}

	return ret, nil
}

func (b *builder) Build() (rnode.Node, error) {
	if b.State() == rnode.NodeExists && b.resource == nil {
		return nil, fmt.Errorf("ForwardingRule %s resource is nil with state %s", b.ID(), b.State())
	}

	ret := &forwardingRuleNode{resource: b.resource}
	if err := ret.InitFromBuilder(b); err != nil {
		return nil, err
	}

	return ret, nil
}
