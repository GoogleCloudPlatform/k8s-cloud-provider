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
package lb

import (
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/testing/ez"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/workflow/testlib"
)

func init() {
	start := func() *rgraph.Graph {
		ezg := ez.Graph{
			Nodes: []ez.Node{
				{Name: "tcp-route", Refs: []ez.Ref{{Field: "Rules.Action.Destinations.ServiceName", To: "bs"}}},
				{Name: "bs", Refs: []ez.Ref{{Field: "Backends.Group", To: "us-central1-b/neg"}, {Field: "Healthchecks", To: "hc"}}},

				{Name: "hc"},
				{Name: "neg", Zone: "us-central1-b"},
			},
		}
		return ezg.Builder().MustBuild()
	}

	testlib.Register(&testlib.TestCase{
		Name:        "lb/tcp-route",
		Description: "Create a lb with tcp route.",
		Steps: []testlib.Step{
			{
				Description: "Create TcpRoute LB",
				Graph:       start(),
			},
		},
	})

	update := func() *rgraph.Graph {
		ezg := ez.Graph{
			Nodes: []ez.Node{
				{Name: "tcp-route", Refs: []ez.Ref{
					{Field: "Rules.Action.Destinations.ServiceName", To: "bs"},
					{Field: "Rules.Action.Destinations.ServiceName", To: "bs1"},
				}},
				{Name: "bs", Refs: []ez.Ref{{Field: "Backends.Group", To: "us-central1-b/neg"}, {Field: "Healthchecks", To: "hc"}}},
				{Name: "hc"},

				{Name: "bs1", Refs: []ez.Ref{{Field: "Backends.Group", To: "us-central1-c/neg"}, {Field: "Healthchecks", To: "hc1"}}},
				{Name: "hc1"},

				{Name: "neg", Zone: "us-central1-b"},
				{Name: "neg", Zone: "us-central1-c"},
			},
		}
		return ezg.Builder().MustBuild()
	}

	testlib.Register(&testlib.TestCase{
		Name:        "lb/tcp-route-multi-bs",
		Description: "Create a lb with multiple backends for tcp route.",
		Steps: []testlib.Step{
			{
				Description: "Create TcpRoute LB",
				Graph:       start(),
			},
			{
				Description: "Add backend service",
				Graph:       update(),
			},
			{
				Description: "Remove backend service",
				Graph:       start(),
			},
		},
	})
}
