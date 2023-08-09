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

package ez

import (
	"strings"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/address"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/backendservice"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/fake"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/forwardingrule"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/healthcheck"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/networkendpointgroup"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/targethttpproxy"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/urlmap"
	"google.golang.org/api/compute/v1"
)

var (
	allNodeFactories = []nodeFactory{
		addressFactory{},
		backendServiceFactory{},
		fakeFactory{},
		forwardingRuleFactory{},
		healthCheckFactory{},
		negFactory{},
		targetHttpProxyFactory{},
		urlMapFactory{},
	}
)

func getFactory(name string) nodeFactory {
	for _, nf := range allNodeFactories {
		if nf.match(name) {
			return nf
		}
	}
	panicf("getFactory: invalid name: %q", name)
	panic("not reached")
}

type nodeFactory interface {
	match(name string) bool
	id(g *Graph, n *Node) *cloud.ResourceID
	builder(g *Graph, n *Node) rnode.Builder
}

func getProject(g *Graph, n *Node) string {
	if n.Project != "" {
		return n.Project
	}
	if g.Project != "" {
		return g.Project
	}
	return "test-project"
}

func setCommonOptions(n *Node, b rnode.Builder) {
	b.SetOwnership(rnode.OwnershipManaged)
	switch {
	case n.Options&External != 0:
		b.SetOwnership(rnode.OwnershipExternal)
	case n.Options&Managed != 0:
		b.SetOwnership(rnode.OwnershipManaged)
	}

	b.SetState(rnode.NodeExists)
	switch {
	case n.Options&Exists != 0:
	case n.Options&DoesNotExist != 0:
		b.SetState(rnode.NodeDoesNotExist)
	}
}

type addressFactory struct{}

func (addressFactory) match(name string) bool { return strings.HasPrefix(name, "addr") }

func (addressFactory) id(g *Graph, n *Node) *cloud.ResourceID {
	switch {
	case n.Region == "" && n.Zone == "":
		return address.ID(getProject(g, n), meta.GlobalKey(n.Name))
	case n.Region != "":
		return address.ID(getProject(g, n), meta.RegionalKey(n.Name, n.Region))
	default:
		panicf("addressFactory: invalid scope: %+v", n)
	}
	panic("not reached")
}

func (f addressFactory) builder(g *Graph, n *Node) rnode.Builder {
	id := f.id(g, n)
	b := address.NewBuilder(id)
	setCommonOptions(n, b)

	if b.State() == rnode.NodeExists {
		ma := address.NewMutableAddress(id.ProjectID, id.Key)
		err := ma.Access(func(x *compute.Address) {
			if n.SetupFunc != nil {
				sf, ok := n.SetupFunc.(func(x *compute.Address))
				if !ok {
					panicf("addressFactory: invalid type for SetupFunc: %T", n.SetupFunc)
				}
				sf(x)
			}
		})
		if g.Options&PanicOnAccessErr != 0 && err != nil {
			panicf("addressFactory %s: Access: %v", id, err)
		}
		r, err := ma.Freeze()
		if err != nil {
			panicf("addressFactory: Freeze: %v", err)
		}
		err = b.SetResource(r)
		if err != nil {
			panicf("addressFactory: SetResource: %v", err)
		}
	}
	return b
}

type backendServiceFactory struct{}

func (backendServiceFactory) match(name string) bool { return strings.HasPrefix(name, "bs") }

func (backendServiceFactory) id(g *Graph, n *Node) *cloud.ResourceID {
	switch {
	case n.Region == "" && n.Zone == "":
		return backendservice.ID(getProject(g, n), meta.GlobalKey(n.Name))
	case n.Region != "":
		return backendservice.ID(getProject(g, n), meta.RegionalKey(n.Name, n.Region))
	default:
		panicf("backendServiceFactory: invalid scope: %+v", n)
	}
	panic("not reached")
}

func (f backendServiceFactory) builder(g *Graph, n *Node) rnode.Builder {
	id := f.id(g, n)
	b := backendservice.NewBuilder(id)
	setCommonOptions(n, b)

	if b.State() == rnode.NodeExists {
		ma := backendservice.NewMutableBackendService(id.ProjectID, id.Key)
		err := ma.Access(func(x *compute.BackendService) {
			for _, ref := range n.Refs {
				switch ref.Field {
				case "Backends.Group":
					backend := &compute.Backend{
						Group: g.ids.selfLink(ref.To),
					}
					x.Backends = append(x.Backends, backend)
				case "Healthchecks":
					x.HealthChecks = append(x.HealthChecks, g.ids.selfLink(ref.To))
				default:
					panicf("invalid Ref Field: %q (must be one of [Backends.Group, HealthChecks])", ref.Field)
				}
			}
			if n.SetupFunc != nil {
				sf, ok := n.SetupFunc.(func(x *compute.BackendService))
				if !ok {
					panicf("invalid type for SetupFunc: %T", n.SetupFunc)
				}
				sf(x)
			}
		})
		if g.Options&PanicOnAccessErr != 0 && err != nil {
			panicf("backendServiceFactory %s: Access: %v", id, err)
		}
		r, err := ma.Freeze()
		if err != nil {
			panic(err)
		}
		err = b.SetResource(r)
		if err != nil {
			panic(err)
		}
	}
	return b
}

type fakeFactory struct{}

func (fakeFactory) match(name string) bool { return strings.HasPrefix(name, "fake") }

func (fakeFactory) id(g *Graph, n *Node) *cloud.ResourceID {
	switch {
	case n.Region == "" && n.Zone == "":
		return fake.ID(getProject(g, n), meta.GlobalKey(n.Name))
	case n.Region != "":
		return fake.ID(getProject(g, n), meta.RegionalKey(n.Name, n.Region))
	case n.Zone != "":
		return fake.ID(getProject(g, n), meta.ZonalKey(n.Name, n.Zone))
	default:
		panicf("invalid id: %+v", n)
	}
	panic("not reached")
}

func (f fakeFactory) builder(g *Graph, n *Node) rnode.Builder {
	return fake.NewBuilder(f.id(g, n))
}

type forwardingRuleFactory struct{}

func (forwardingRuleFactory) match(name string) bool { return strings.HasPrefix(name, "fr") }

func (forwardingRuleFactory) id(g *Graph, n *Node) *cloud.ResourceID {
	switch {
	case n.Region == "" && n.Zone == "":
		return forwardingrule.ID(getProject(g, n), meta.GlobalKey(n.Name))
	case n.Region != "":
		return forwardingrule.ID(getProject(g, n), meta.RegionalKey(n.Name, n.Region))
	default:
		panicf("invalid id: %+v", n)
	}
	panic("not reached")
}

func (f forwardingRuleFactory) builder(g *Graph, n *Node) rnode.Builder {
	id := f.id(g, n)
	b := forwardingrule.NewBuilder(id)
	setCommonOptions(n, b)

	if b.State() == rnode.NodeExists {
		ma := forwardingrule.NewMutableForwardingRule(id.ProjectID, id.Key)
		err := ma.Access(func(x *compute.ForwardingRule) {
			for _, ref := range n.Refs {
				switch ref.Field {
				case "IPAddress":
					x.IPAddress = g.ids.selfLink(ref.To)
				case "Target":
					x.Target = g.ids.selfLink(ref.To)
				default:
					panicf("invalid Ref Field: %q (must be one of [IPAddress,Target])", ref.Field)
				}
			}

			if n.SetupFunc != nil {
				sf, ok := n.SetupFunc.(func(x *compute.ForwardingRule))
				if !ok {
					panicf("invalid type for SetupFunc: %T", n.SetupFunc)
				}
				sf(x)
			}
		})
		if g.Options&PanicOnAccessErr != 0 && err != nil {
			panicf("forwardingRuleFactory %s: Access: %v", id, err)
		}
		r, err := ma.Freeze()
		if err != nil {
			panic(err)
		}
		err = b.SetResource(r)
		if err != nil {
			panic(err)
		}
	}
	return b
}

type healthCheckFactory struct{}

func (healthCheckFactory) match(name string) bool { return strings.HasPrefix(name, "hc") }

func (healthCheckFactory) id(g *Graph, n *Node) *cloud.ResourceID {
	switch {
	case n.Region == "" && n.Zone == "":
		return healthcheck.ID(getProject(g, n), meta.GlobalKey(n.Name))
	case n.Region != "":
		return healthcheck.ID(getProject(g, n), meta.RegionalKey(n.Name, n.Region))
	default:
		panicf("invalid id: %+v", n)
	}
	panic("not reached")
}

func (f healthCheckFactory) builder(g *Graph, n *Node) rnode.Builder {
	id := f.id(g, n)
	b := healthcheck.NewBuilder(id)
	setCommonOptions(n, b)

	if b.State() == rnode.NodeExists {
		ma := healthcheck.NewMutableHealthCheck(id.ProjectID, id.Key)
		err := ma.Access(func(x *compute.HealthCheck) {
			if n.SetupFunc != nil {
				sf, ok := n.SetupFunc.(func(x *compute.HealthCheck))
				if !ok {
					panicf("invalid type for SetupFunc: %T", n.SetupFunc)
				}
				sf(x)
			}
		})
		if g.Options&PanicOnAccessErr != 0 && err != nil {
			panicf("healthCheckFactory %s: Access: %v", id, err)
		}
		r, err := ma.Freeze()
		if err != nil {
			panic(err)
		}
		err = b.SetResource(r)
		if err != nil {
			panic(err)
		}
	}
	return b
}

type negFactory struct{}

func (negFactory) match(name string) bool { return strings.HasPrefix(name, "neg") }

func (negFactory) id(g *Graph, n *Node) *cloud.ResourceID {
	switch {
	case n.Zone != "":
		return networkendpointgroup.ID(getProject(g, n), meta.ZonalKey(n.Name, n.Zone))
	default:
		panicf("invalid id: %+v", n)
	}
	panic("not reached")
}

func (f negFactory) builder(g *Graph, n *Node) rnode.Builder {
	id := f.id(g, n)
	b := networkendpointgroup.NewBuilder(id)
	setCommonOptions(n, b)

	if b.State() == rnode.NodeExists {
		ma := networkendpointgroup.NewMutableNetworkEndpointGroup(id.ProjectID, id.Key)
		err := ma.Access(func(x *compute.NetworkEndpointGroup) {
			if n.SetupFunc != nil {
				sf, ok := n.SetupFunc.(func(x *compute.NetworkEndpointGroup))
				if !ok {
					panicf("invalid type for SetupFunc: %T", n.SetupFunc)
				}
				sf(x)
			}
		})
		if g.Options&PanicOnAccessErr != 0 && err != nil {
			panicf("negFactory %s: Access: %v", id, err)
		}
		r, err := ma.Freeze()
		if err != nil {
			panic(err)
		}
		err = b.SetResource(r)
		if err != nil {
			panic(err)
		}
	}
	return b
}

type targetHttpProxyFactory struct{}

func (targetHttpProxyFactory) match(name string) bool { return strings.HasPrefix(name, "thp") }

func (targetHttpProxyFactory) id(g *Graph, n *Node) *cloud.ResourceID {
	switch {
	case n.Region == "" && n.Zone == "":
		return targethttpproxy.ID(getProject(g, n), meta.GlobalKey(n.Name))
	case n.Region != "":
		return targethttpproxy.ID(getProject(g, n), meta.RegionalKey(n.Name, n.Region))
	default:
		panicf("invalid id: %+v", n)
	}
	panic("not reached")
}

func (f targetHttpProxyFactory) builder(g *Graph, n *Node) rnode.Builder {
	id := f.id(g, n)
	b := targethttpproxy.NewBuilder(id)
	setCommonOptions(n, b)

	if b.State() == rnode.NodeExists {
		ma := targethttpproxy.NewMutableTargetHttpProxy(id.ProjectID, id.Key)
		err := ma.Access(func(x *compute.TargetHttpProxy) {
			for _, ref := range n.Refs {
				switch ref.Field {
				case "UrlMap":
					x.UrlMap = g.ids.selfLink(ref.To)
				default:
					panicf("invalid Ref Field: %q (must be one of [UrlMap])", ref.Field)
				}
			}

			if n.SetupFunc != nil {
				sf, ok := n.SetupFunc.(func(x *compute.TargetHttpProxy))
				if !ok {
					panicf("invalid type for SetupFunc: %T", n.SetupFunc)
				}
				sf(x)
			}
		})
		if g.Options&PanicOnAccessErr != 0 && err != nil {
			panicf("targetHttpProxyFactory %s: Access: %v", id, err)
		}
		r, err := ma.Freeze()
		if err != nil {
			panic(err)
		}
		err = b.SetResource(r)
		if err != nil {
			panic(err)
		}
	}
	return b
}

type urlMapFactory struct{}

func (urlMapFactory) match(name string) bool { return strings.HasPrefix(name, "um") }

func (urlMapFactory) id(g *Graph, n *Node) *cloud.ResourceID {
	switch {
	case n.Region == "" && n.Zone == "":
		return urlmap.ID(getProject(g, n), meta.GlobalKey(n.Name))
	case n.Region != "":
		return urlmap.ID(getProject(g, n), meta.RegionalKey(n.Name, n.Region))
	default:
		panicf("invalid id: %+v", n)
	}
	panic("not reached")
}

func (f urlMapFactory) builder(g *Graph, n *Node) rnode.Builder {
	id := f.id(g, n)
	b := urlmap.NewBuilder(id)
	setCommonOptions(n, b)

	if b.State() == rnode.NodeExists {
		ma := urlmap.NewMutableUrlMap(id.ProjectID, id.Key)
		err := ma.Access(func(x *compute.UrlMap) {
			for _, ref := range n.Refs {
				switch ref.Field {
				case "DefaultService":
					x.DefaultService = g.ids.selfLink(ref.To)
				default:
					panicf("invalid Ref Field: %q (must be one of [DefaultService])", ref.Field)
				}
			}

			if n.SetupFunc != nil {
				sf, ok := n.SetupFunc.(func(x *compute.UrlMap))
				if !ok {
					panicf("invalid type for SetupFunc: %T", n.SetupFunc)
				}
				sf(x)
			}
		})
		if g.Options&PanicOnAccessErr != 0 && err != nil {
			panicf("urlMapFactory %s: Access: %v", id, err)
		}
		r, err := ma.Freeze()
		if err != nil {
			panic(err)
		}
		err = b.SetResource(r)
		if err != nil {
			panic(err)
		}
	}
	return b
}
