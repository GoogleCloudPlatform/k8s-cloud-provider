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
	"google.golang.org/api/compute/v1"
)

func init() {
	base := ez.Graph{
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

	start := func() *rgraph.Graph { return base.Builder().MustBuild() }

	updateTarget := func() *rgraph.Graph {
		ezg := base.Clone()
		ezg.Set(ez.Node{Name: "fr", Refs: []ez.Ref{{Field: "IPAddress", To: "addr"}, {Field: "Target", To: "thp-other"}}})
		ezg.Set(ez.Node{Name: "thp-other", Refs: []ez.Ref{{Field: "UrlMap", To: "um"}}})
		ezg.Remove("thp")
		return ezg.Builder().MustBuild()
	}
	testlib.Register(&testlib.TestCase{
		Name:        "lb/forwardingrule/update-target",
		Description: "Update a ForwardingRule Target.",
		Steps: []testlib.Step{
			{Description: "Create LB", Graph: start()},
			{Description: "Update Target", Graph: updateTarget()},
		},
	})

	updateLabels := func() *rgraph.Graph {
		ezg := base.Clone()
		ezg.Set(ez.Node{
			Name:      "fr",
			Refs:      []ez.Ref{{Field: "IPAddress", To: "addr"}, {Field: "Target", To: "thp"}},
			SetupFunc: func(x *compute.ForwardingRule) { x.Labels = map[string]string{"foo": "bar"} },
		})
		return ezg.Builder().MustBuild()
	}
	testlib.Register(&testlib.TestCase{
		Name:        "lb/forwardingrule/update-label",
		Description: "Update a ForwardingRule Label.",
		Steps: []testlib.Step{
			{Description: "Create LB", Graph: start()},
			{Description: "Update Labels", Graph: updateLabels()},
		},
	})

	updateDescription := func() *rgraph.Graph {
		ezg := base.Clone()
		ezg.Set(ez.Node{
			Name:      "fr",
			Refs:      []ez.Ref{{Field: "IPAddress", To: "addr"}, {Field: "Target", To: "thp"}},
			SetupFunc: func(x *compute.ForwardingRule) { x.Description = "changed" },
		})
		return ezg.Builder().MustBuild()
	}
	testlib.Register(&testlib.TestCase{
		Name:        "lb/forwardingrule/update-description",
		Description: "Update a ForwardingRule Description. Will recreate the resource.",
		Steps: []testlib.Step{
			{Description: "Create LB", Graph: start()},
			{Description: "Update Labels", Graph: updateDescription()},
		},
	})
}
