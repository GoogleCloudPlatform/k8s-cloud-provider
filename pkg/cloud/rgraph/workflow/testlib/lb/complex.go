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
				{Name: "addr1"},
				{Name: "addr2"},
				{Name: "fr1", Refs: []ez.Ref{{Field: "IPAddress", To: "addr1"}, {Field: "Target", To: "thp"}}},
				{Name: "fr2", Refs: []ez.Ref{{Field: "IPAddress", To: "addr2"}, {Field: "Target", To: "thp"}}},
				{Name: "fr3", Refs: []ez.Ref{{Field: "IPAddress", To: "addr2"}, {Field: "Target", To: "thp2"}}},
				{Name: "thp", Refs: []ez.Ref{{Field: "UrlMap", To: "um"}}},
				{Name: "thp2", Refs: []ez.Ref{{Field: "UrlMap", To: "um2"}}},
				{Name: "um", Refs: []ez.Ref{{Field: "DefaultService", To: "bs"}}},
				{Name: "um2", Refs: []ez.Ref{{Field: "DefaultService", To: "bs2"}}},
				{
					Name: "bs",
					Refs: []ez.Ref{
						{Field: "Backends.Group", To: "us-central1-a/neg1"},
						{Field: "Backends.Group", To: "us-central1-b/neg1"},
						{Field: "Backends.Group", To: "us-central1-c/neg1"},
						{Field: "Backends.Group", To: "us-central1-b/neg2"},
						{Field: "Healthchecks", To: "hc"},
					},
				},
				{
					Name: "bs2",
					Refs: []ez.Ref{
						{Field: "Backends.Group", To: "us-central1-b/neg1"},
						{Field: "Backends.Group", To: "us-central1-b/neg2"},
						{Field: "Healthchecks", To: "hc2"},
					},
				},
				{Name: "hc"},
				{Name: "hc2"},
				{Name: "neg1", Zone: "us-central1-a"},
				{Name: "neg1", Zone: "us-central1-b"},
				{Name: "neg1", Zone: "us-central1-c"},
				{Name: "neg2", Zone: "us-central1-b"},
			},
		}
		return ezg.Builder().MustBuild()
	}
	simple := func() *rgraph.Graph {
		ezg := ez.Graph{
			Nodes: []ez.Node{
				{Name: "addr1"},
				{Name: "fr1", Refs: []ez.Ref{{Field: "IPAddress", To: "addr1"}, {Field: "Target", To: "thp"}}},
				{Name: "thp", Refs: []ez.Ref{{Field: "UrlMap", To: "um"}}},
				{Name: "um", Refs: []ez.Ref{{Field: "DefaultService", To: "bs"}}},
				{Name: "bs", Refs: []ez.Ref{{Field: "Backends.Group", To: "us-central1-b/neg1"}, {Field: "Healthchecks", To: "hc"}}},
				{Name: "hc"},
				{Name: "neg1", Zone: "us-central1-b"},

				{Name: "fr2", Options: ez.DoesNotExist},
				{Name: "fr3", Options: ez.DoesNotExist},
			},
		}
		return ezg.Builder().MustBuild()
	}

	testlib.Register(&testlib.TestCase{
		Name:        "lb/complex",
		Description: "Complex LB graph.",
		Steps: []testlib.Step{
			{Description: "Create LB", Graph: start()},
			{Graph: simple()},
			{Graph: simple()},
		},
	})
}
