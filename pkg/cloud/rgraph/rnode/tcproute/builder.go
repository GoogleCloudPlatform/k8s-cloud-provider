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

package tcproute

import (
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"google.golang.org/api/networkservices/v1"
	beta "google.golang.org/api/networkservices/v1beta1"
)

const (
	resourceName = "TcpRoute"
)

// NewBuilder creates builder for tcp route.
func NewBuilder(id *cloud.ResourceID) rnode.Builder {
	b := &builder{}
	b.Defaults(id)
	return b
}

// NewBuilderWithResource creates builder for tcp route
// with predefined resource.
func NewBuilderWithResource(r TcpRoute) rnode.Builder {
	b := &builder{resource: r}
	b.Init(r.ResourceID(), rnode.NodeUnknown, rnode.OwnershipUnknown, r)
	return b
}

type builder struct {
	rnode.BuilderBase
	resource TcpRoute
}

// builder implements node.Builder.
var _ rnode.Builder = (*builder)(nil)

func (b *builder) Resource() rnode.UntypedResource { return b.resource }

func (b *builder) SetResource(u rnode.UntypedResource) error {
	r, ok := u.(TcpRoute)
	if !ok {
		return fmt.Errorf("cannot set TcpRoute from untyped resource, %T", u)
	}
	b.resource = r
	return nil
}

func (b *builder) SyncFromCloud(ctx context.Context, gcp cloud.Cloud) error {
	return rnode.GenericGet[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute](
		ctx, gcp, resourceName, &tcpRouteOps{}, &tcpRouteTypeTrait{}, b)
}

func (b *builder) OutRefs() ([]rnode.ResourceRef, error) {
	if b.resource == nil {
		return nil, nil
	}
	// TODO(kl52752) Add mesh dependency

	var ret []rnode.ResourceRef
	obj, _ := b.resource.ToGA()
	for ruleIdx, rule := range obj.Rules {
		if rule == nil || rule.Action == nil {
			continue
		}
		for destIdx, dest := range rule.Action.Destinations {
			if dest == nil {
				continue
			}
			id, err := cloud.ParseResourceURL(dest.ServiceName)
			if err != nil {
				return nil, fmt.Errorf("tcpRouteNode: %w", err)
			}
			ret = append(ret, rnode.ResourceRef{
				From: b.resource.ResourceID(),
				Path: api.Path{}.Field("Rules").Index(ruleIdx).Field("Action").Field("Destinations").Index(destIdx).Field("ServiceName"),
				To:   id,
			})
		}
	}
	return ret, nil
}

func (b *builder) Build() (rnode.Node, error) {
	if b.State() == rnode.NodeExists && b.resource == nil {
		return nil, fmt.Errorf("TcpRoute %s resource is nil with state %s", b.ID(), b.State())
	}

	ret := &tcpRouteNode{resource: b.resource}
	if err := ret.InitFromBuilder(b); err != nil {
		return nil, err
	}

	return ret, nil
}
