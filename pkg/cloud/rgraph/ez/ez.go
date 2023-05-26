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

// Package ez is a utility to create complex resource graphs for testing from a
// concise description by use of naming conventions and default values.
//
// This package should only be used for testing. There is little error handling;
// most errors will result in a panic() to reduce the verbosity of the code.
package ez

import (
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
)

// GraphOption are flags controlling the state of the Graphs. Options
// should be joined via a bitwise OR "|".
type GraphOption int

const (
	// PanicOnAccessErr causes panic() if Accces() returns an error. Otherwise,
	// the error is ignored.
	PanicOnAccessErr = 1 << iota
)

func panicf(s string, args ...any) { panic(fmt.Sprintf(s, args...)) }

type Graph struct {
	Nodes   []Node
	Project string
	Options GraphOption

	ids idMap
}

type idMap map[string]*cloud.ResourceID

func (m idMap) selfLink(name string) string {
	r, ok := m[name]
	if !ok {
		panicf("selfLink: %q is not in the map", name)
	}
	return r.SelfLink(meta.VersionGA)
}

// Clone a copy of the Graph.
func (g *Graph) Clone() *Graph {
	return &Graph{
		Nodes:   append([]Node{}, g.Nodes...),
		Project: g.Project,
	}
}

// Set the value of the node, overwritting the existing one.
func (g *Graph) Set(n Node) {
	for i := range g.Nodes {
		if g.Nodes[i].Name == n.Name {
			g.Nodes[i] = n
			return
		}
	}
	g.Nodes = append(g.Nodes, n)
}

// Remove the name name from the Graph.
func (g *Graph) Remove(name string) {
	for i := range g.Nodes {
		if g.Nodes[i].Name == name {
			g.Nodes = append(g.Nodes[0:i], g.Nodes[i+1:]...)
			return
		}
	}
}

// Node in the graph.
type Node struct {
	Name      string
	Refs      []Ref
	Options   NodeOption
	SetupFunc any

	Region  string
	Zone    string
	Project string
}

// Ref to another Node.
type Ref struct {
	// Field for this reference. See the specific type Factory for which fields
	// are available.
	Field string
	// To is the name of the node reference. For regional and zonal scopes, this
	// should be <scope>/<name>.
	To string
}

// NodeOption are flags controlling the state of the nodes. Options
// should be joined via a bitwise OR "|".
type NodeOption int

const (
	// External ownership.
	External NodeOption = 1 << iota
	// Managed ownership.
	Managed
	// Exists state (default).
	Exists
	// DoesNotExist state.
	DoesNotExist
)

func (g *Graph) Builder() *rgraph.Builder {
	g.ids = idMap{}

	for _, n := range g.Nodes {
		nf := getFactory(n.Name)
		var name string
		switch {
		case n.Region != "":
			name = fmt.Sprintf("%s/%s", n.Region, n.Name)
		case n.Zone != "":
			name = fmt.Sprintf("%s/%s", n.Zone, n.Name)
		default:
			name = n.Name
		}
		g.ids[name] = nf.id(g, &n)

	}

	b := rgraph.NewBuilder()
	for _, n := range g.Nodes {
		nf := getFactory(n.Name)
		nb := nf.builder(g, &n)
		b.Add(nb)
	}

	return b
}
