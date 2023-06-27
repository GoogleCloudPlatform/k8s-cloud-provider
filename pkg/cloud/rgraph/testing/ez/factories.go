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
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/fake"
)

var (
	allNodeFactories = []nodeFactory{
		fakeFactory{},
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
