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

package testutils

import (
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/address"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/backendservice"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/forwardingrule"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/healthcheck"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/networkendpointgroup"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/targethttpproxy"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/tcproute"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/urlmap"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/networkservices/v1"
)

// ResourceBuilder is a convenience wrapper for tests. Do not use this in production.
type ResourceBuilder struct {
	Project string
	Name    string
	Region  string
	Zone    string
}

func (b *ResourceBuilder) Key() *meta.Key {
	switch {
	case b.Region == "" && b.Zone == "":
		return meta.GlobalKey(b.Name)
	case b.Region != "":
		return meta.RegionalKey(b.Name, b.Region)
	case b.Zone != "":
		return meta.ZonalKey(b.Name, b.Zone)
	}
	panic(fmt.Sprintf("missing fields: %+v", *b))
}

func (b *ResourceBuilder) P(project string) *ResourceBuilder {
	ret := *b
	ret.Project = project
	return &ret
}

func (b *ResourceBuilder) N(name string) *ResourceBuilder {
	ret := *b
	ret.Name = name
	return &ret
}

func (b *ResourceBuilder) R(region string) *ResourceBuilder {
	ret := *b
	ret.Region = region
	return &ret
}

func (b *ResourceBuilder) Z(zone string) *ResourceBuilder {
	ret := *b
	ret.Zone = zone
	return &ret
}

func (b *ResourceBuilder) DefaultRegion() *ResourceBuilder { return b.R("us-central1") }
func (b *ResourceBuilder) DefaultZone() *ResourceBuilder   { return b.Z("us-central1-b") }

func (b *ResourceBuilder) Address() *AddressBuilder               { return &AddressBuilder{*b} }
func (b *ResourceBuilder) BackendService() *BackendServiceBuilder { return &BackendServiceBuilder{*b} }
func (b *ResourceBuilder) ForwardingRule() *ForwardingRuleBuilder { return &ForwardingRuleBuilder{*b} }
func (b *ResourceBuilder) HealthCheck() *HealthCheckBuilder       { return &HealthCheckBuilder{*b} }
func (b *ResourceBuilder) NetworkEndpointGroup() *NetworkEndpointGroupBuilder {
	return &NetworkEndpointGroupBuilder{*b}
}
func (b *ResourceBuilder) TargetHttpProxy() *TargetHttpProxyBuilder {
	return &TargetHttpProxyBuilder{*b}
}
func (b *ResourceBuilder) UrlMap() *UrlMapBuilder { return &UrlMapBuilder{*b} }

type AddressBuilder struct{ ResourceBuilder }

func (b *AddressBuilder) ID() *cloud.ResourceID { return address.ID(b.Project, b.Key()) }
func (b *AddressBuilder) SelfLink() string      { return b.ID().SelfLink(meta.VersionGA) }
func (b *AddressBuilder) Resource() address.Mutable {
	return address.New(b.Project, b.Key())
}

func (b *AddressBuilder) Build(f func(*compute.Address)) rnode.Builder {
	m := b.Resource()
	if f != nil {
		m.Access(f)
	}
	r, _ := m.Freeze()
	nb := address.NewBuilderWithResource(r)
	nb.SetOwnership(rnode.OwnershipManaged)
	nb.SetState(rnode.NodeExists)
	return nb
}

type BackendServiceBuilder struct{ ResourceBuilder }

func (b *BackendServiceBuilder) ID() *cloud.ResourceID {
	return backendservice.ID(b.Project, b.Key())
}
func (b *BackendServiceBuilder) SelfLink() string { return b.ID().SelfLink(meta.VersionGA) }
func (b *BackendServiceBuilder) Resource() backendservice.Mutable {
	return backendservice.New(b.Project, b.Key())
}

func (b *BackendServiceBuilder) Build(f func(*compute.BackendService)) rnode.Builder {
	m := b.Resource()
	if f != nil {
		m.Access(f)
	}
	r, _ := m.Freeze()
	nb := backendservice.NewBuilderWithResource(r)
	nb.SetOwnership(rnode.OwnershipManaged)
	nb.SetState(rnode.NodeExists)
	return nb
}

type ForwardingRuleBuilder struct{ ResourceBuilder }

func (b *ForwardingRuleBuilder) ID() *cloud.ResourceID {
	return forwardingrule.ID(b.Project, b.Key())
}
func (b *ForwardingRuleBuilder) SelfLink() string { return b.ID().SelfLink(meta.VersionGA) }
func (b *ForwardingRuleBuilder) Resource() forwardingrule.Mutable {
	return forwardingrule.New(b.Project, b.Key())
}

func (b *ForwardingRuleBuilder) Build(f func(*compute.ForwardingRule)) rnode.Builder {
	m := b.Resource()
	if f != nil {
		m.Access(f)
	}
	r, _ := m.Freeze()
	nb := forwardingrule.NewBuilderWithResource(r)
	nb.SetOwnership(rnode.OwnershipManaged)
	nb.SetState(rnode.NodeExists)
	return nb
}

type HealthCheckBuilder struct{ ResourceBuilder }

func (b *HealthCheckBuilder) ID() *cloud.ResourceID { return healthcheck.ID(b.Project, b.Key()) }
func (b *HealthCheckBuilder) SelfLink() string      { return b.ID().SelfLink(meta.VersionGA) }
func (b *HealthCheckBuilder) Resource() healthcheck.Mutable {
	return healthcheck.New(b.Project, b.Key())
}

func (b *HealthCheckBuilder) Build(f func(*compute.HealthCheck)) rnode.Builder {
	m := b.Resource()
	if f != nil {
		m.Access(f)
	}
	r, _ := m.Freeze()
	nb := healthcheck.NewBuilderWithResource(r)
	nb.SetOwnership(rnode.OwnershipManaged)
	nb.SetState(rnode.NodeExists)
	return nb
}

type NetworkEndpointGroupBuilder struct{ ResourceBuilder }

func (b *NetworkEndpointGroupBuilder) ID() *cloud.ResourceID {
	return networkendpointgroup.ID(b.Project, b.Key())
}
func (b *NetworkEndpointGroupBuilder) SelfLink() string { return b.ID().SelfLink(meta.VersionGA) }
func (b *NetworkEndpointGroupBuilder) Resource() networkendpointgroup.Mutable {
	return networkendpointgroup.New(b.Project, b.Key())
}

func (b *NetworkEndpointGroupBuilder) Build(f func(*compute.NetworkEndpointGroup)) rnode.Builder {
	m := b.Resource()
	if f != nil {
		m.Access(f)
	}
	r, _ := m.Freeze()
	nb := networkendpointgroup.NewBuilderWithResource(r)
	nb.SetOwnership(rnode.OwnershipManaged)
	nb.SetState(rnode.NodeExists)
	return nb
}

type TargetHttpProxyBuilder struct{ ResourceBuilder }

func (b *TargetHttpProxyBuilder) ID() *cloud.ResourceID {
	return targethttpproxy.ID(b.Project, b.Key())
}
func (b *TargetHttpProxyBuilder) SelfLink() string { return b.ID().SelfLink(meta.VersionGA) }
func (b *TargetHttpProxyBuilder) Resource() targethttpproxy.Mutable {
	return targethttpproxy.New(b.Project, b.Key())
}

func (b *TargetHttpProxyBuilder) Build(f func(*compute.TargetHttpProxy)) rnode.Builder {
	m := b.Resource()
	if f != nil {
		m.Access(f)
	}
	r, _ := m.Freeze()
	nb := targethttpproxy.NewBuilderWithResource(r)
	nb.SetOwnership(rnode.OwnershipManaged)
	nb.SetState(rnode.NodeExists)
	return nb
}

type UrlMapBuilder struct{ ResourceBuilder }

func (b *UrlMapBuilder) ID() *cloud.ResourceID { return urlmap.ID(b.Project, b.Key()) }
func (b *UrlMapBuilder) SelfLink() string      { return b.ID().SelfLink(meta.VersionGA) }
func (b *UrlMapBuilder) Resource() urlmap.Mutable {
	return urlmap.New(b.Project, b.Key())
}

func (b *UrlMapBuilder) Build(f func(*compute.UrlMap)) rnode.Builder {
	m := b.Resource()
	if f != nil {
		m.Access(f)
	}
	r, _ := m.Freeze()
	nb := urlmap.NewBuilderWithResource(r)
	nb.SetOwnership(rnode.OwnershipManaged)
	nb.SetState(rnode.NodeExists)
	return nb
}

type TcpRouteBuilder struct{ ResourceBuilder }

func (b *TcpRouteBuilder) ID() *cloud.ResourceID { return tcproute.ID(b.Project, b.Key()) }
func (b *TcpRouteBuilder) SelfLink() string      { return b.ID().SelfLink(meta.VersionGA) }
func (b *TcpRouteBuilder) Resource() tcproute.Mutable {
	return tcproute.New(b.Project, b.Key())
}

func (b *TcpRouteBuilder) Build(f func(*networkservices.TcpRoute)) rnode.Builder {
	m := b.Resource()
	if f != nil {
		m.Access(f)
	}
	r, _ := m.Freeze()
	nb := tcproute.NewBuilderWithResource(r)
	nb.SetOwnership(rnode.OwnershipManaged)
	nb.SetState(rnode.NodeExists)
	return nb
}
