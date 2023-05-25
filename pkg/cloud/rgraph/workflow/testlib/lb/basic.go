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
				{Name: "addr"},
				{Name: "fr", Refs: []ez.Ref{{Field: "IPAddress", To: "addr"}, {Field: "Target", To: "thp"}}},
				{Name: "thp", Refs: []ez.Ref{{Field: "UrlMap", To: "um"}}},
				{Name: "um", Refs: []ez.Ref{{Field: "DefaultService", To: "bs"}}},
				{Name: "bs", Refs: []ez.Ref{{Field: "Backends.Group", To: "us-central1-b/neg"}, {Field: "Healthchecks", To: "hc"}}},
				{Name: "hc"},
				{Name: "neg", Zone: "us-central1-b"},
			},
		}
		return ezg.Builder().MustBuild()
	}

	testlib.Register(&testlib.TestCase{
		Name:        "lb/basic",
		Description: "Create a basic lb with no pre-existing resources.",
		Steps: []testlib.Step{
			{
				Description: "Create LB",
				Graph:       start(),
			},
		},
	})
	testlib.Register(&testlib.TestCase{
		Name:        "lb/basic-stable",
		Description: "Create a basic lb with no pre-existing resources. NOP on subsequent sync.",
		Steps: []testlib.Step{
			{
				Description: "Create LB",
				Graph:       start(),
			},
			{
				Description: "Same graph should be NOP",
				Graph:       start(),
			},
		},
	})
}
