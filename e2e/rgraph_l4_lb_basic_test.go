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
package e2e

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/algo/graphviz"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/backendservice"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/forwardingrule"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/workflow/plan"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func TestRgraphL4LBBasic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	graphBuilder := rgraph.NewBuilder()

	bsID := backendservice.ID(testFlags.project, meta.RegionalKey("rgraph1", "us-central1"))
	frID := forwardingrule.ID(testFlags.project, meta.RegionalKey("rgraph1", "us-central1"))

	frMutResource := forwardingrule.NewMutableForwardingRule(testFlags.project, frID.Key)
	err := frMutResource.Access(func(x *compute.ForwardingRule) {
		y := compute.ForwardingRule{
			IPProtocol:          "TCP",
			AllPorts:            false,
			AllowGlobalAccess:   false,
			BackendService:      bsID.SelfLink(meta.VersionGA),
			Description:         "rgraph created",
			IpVersion:           "IPV4",
			LoadBalancingScheme: "EXTERNAL",
			Name:                frID.Key.Name,
			NetworkTier:         "PREMIUM",
			NoAutomateDnsZone:   false,
			PortRange:           "80",
			Ports:               []string{},
			SourceIpRanges:      []string{},
			ServerResponse:      googleapi.ServerResponse{}, /*
				ForceSendFields: []string{
					"AllowGlobalAccess",
					"AllPorts",
					"IsMirroringCollector",
					"Network",
					"NoAutomateDnsZone",
					"Ports",
					"Subnetwork",
					"Target",
				},
				NullFields: []string{
					"IPAddress",
					"ServiceLabel",
					"Labels",
					"MetadataFilters",
					"ServiceDirectoryRegistrations",
				},*/
		}
		*x = y
	})
	if err != nil {
		//t.Fatal(err)
	}

	frResource, err := frMutResource.Freeze()
	if err != nil {
		t.Fatal(err)
	}

	frBuilder := forwardingrule.NewBuilder(frID)
	frBuilder.SetOwnership(rnode.OwnershipManaged)
	frBuilder.SetState(rnode.NodeExists)
	frBuilder.SetResource(frResource)

	graphBuilder.Add(frBuilder)

	bsMutResource := backendservice.NewMutableBackendService(testFlags.project, bsID.Key)
	bsMutResource.Access(func(x *compute.BackendService) {
		x.LoadBalancingScheme = "EXTERNAL"
		x.Protocol = "TCP"
	})
	bsResource, err := bsMutResource.Freeze()
	if err != nil {
		t.Fatal(err)
	}

	bsBuilder := backendservice.NewBuilder(bsID)
	bsBuilder.SetOwnership(rnode.OwnershipManaged)
	bsBuilder.SetState(rnode.NodeExists)
	bsBuilder.SetResource(bsResource)

	graphBuilder.Add(bsBuilder)

	graph, err := graphBuilder.Build()

	if err != nil {
		t.Fatal(err)
	}

	result, err := plan.Do(ctx, theCloud, graph)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("want=\n%s", graphviz.Do(result.Want))
	t.Logf("got=\n%s", graphviz.Do(result.Got))

	func() {
		var viz exec.GraphvizTracer
		ex, err := exec.NewSerialExecutor(result.Actions, exec.DryRunOption(true), exec.TracerOption(&viz))
		if err != nil {
			return
		}
		ex.Run(context.Background(), theCloud)
		t.Logf("plan=\n%s", viz.String())
	}()

	//t.Fatal("blah")

	// t.Logf("%s", pretty.Sprint(result.Actions))

	result, err = plan.Do(ctx, theCloud, graph)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("plan:\n%s", graph.ExplainPlan())

	ex, err := exec.NewSerialExecutor(result.Actions)

	if err != nil {
		t.Fatal(err)
	}

	exResult, err := ex.Run(ctx, theCloud)

	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%v", exResult)
}
