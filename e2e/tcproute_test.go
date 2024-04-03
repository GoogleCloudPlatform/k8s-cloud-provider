/*
Copyright 2024 Google LLC

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
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/cerrors"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/backendservice"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/networkendpointgroup"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/tcproute"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/workflow/plan"
	"github.com/kr/pretty"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/networkservices/v1"
)

const (
	meshName   = "test-mesh"
	region     = "us-central1"
	zone       = region + "-c"
	routeCIDR  = "10.240.3.83/32"
	routeCIRD2 = "10.240.4.83/32"
)

func TestTcpRoute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	bs := &compute.BackendService{
		Name:                resourceName("bs1"),
		Backends:            []*compute.Backend{},
		LoadBalancingScheme: "INTERNAL_SELF_MANAGED",
	}
	bsKey := meta.GlobalKey(bs.Name)

	t.Cleanup(func() {
		err := theCloud.BackendServices().Delete(ctx, bsKey)
		t.Logf("bs delete: %v", err)
	})

	// TcpRoute needs a BackendService to point to.
	err := theCloud.BackendServices().Insert(ctx, bsKey, bs)
	t.Logf("bs insert: %v", err)
	if err != nil {
		t.Fatal(err)
	}

	// Current API does not support the new URL scheme.
	serviceName := fmt.Sprintf("https://compute.googleapis.com/v1/projects/%s/global/backendServices/%s", testFlags.project, bs.Name)
	tcpr := &networkservices.TcpRoute{
		Name: resourceName("route1"),
		Rules: []*networkservices.TcpRouteRouteRule{
			{
				Action: &networkservices.TcpRouteRouteAction{
					Destinations: []*networkservices.TcpRouteRouteDestination{
						{ServiceName: serviceName},
					},
				},
			},
		},
	}
	t.Logf("tcpr = %s", pretty.Sprint(tcpr))
	tcprKey := meta.GlobalKey(tcpr.Name)

	// Insert
	t.Cleanup(func() {
		err := theCloud.TcpRoutes().Delete(ctx, tcprKey)
		t.Logf("tcpRoute delete: %v", err)
	})

	err = theCloud.TcpRoutes().Insert(ctx, tcprKey, tcpr)
	t.Logf("tcproutes insert: %v", err)
	if err != nil {
		t.Fatalf("Insert() = %v", err)
	}

	// Get
	tcpRoute, err := theCloud.TcpRoutes().Get(ctx, tcprKey)
	t.Logf("tcpRoute = %s", pretty.Sprint(tcpRoute))
	if err != nil {
		t.Fatalf("Get(%s) = %v", tcprKey, err)
	}

	if len(tcpRoute.Rules) < 1 || len(tcpRoute.Rules[0].Action.Destinations) < 1 {
		t.Fatalf("gotTcpRoute = %s, need at least one destination", pretty.Sprint(tcpRoute))
	}
	gotServiceName := tcpRoute.Rules[0].Action.Destinations[0].ServiceName
	if gotServiceName != serviceName {
		t.Fatalf("gotTcpRoute = %s, gotServiceName = %q, want %q", pretty.Sprint(tcpRoute), gotServiceName, serviceName)
	}
}

func buildNEG(graphBuilder *rgraph.Builder, name, zone string) (*cloud.ResourceID, error) {
	negID := networkendpointgroup.ID(testFlags.project, meta.ZonalKey(resourceName(name), zone))
	negMut := networkendpointgroup.NewMutableNetworkEndpointGroup(testFlags.project, negID.Key)
	negMut.Access(func(x *compute.NetworkEndpointGroup) {
		x.Zone = zone
		x.NetworkEndpointType = "GCE_VM_IP_PORT"
		x.Name = negID.Key.Name
		x.Network = defaultNetworkURL()
		x.Subnetwork = defaultSubnetworkURL()
		x.Description = "neg for rGraph test"
	})

	negRes, err := negMut.Freeze()
	if err != nil {
		return nil, err
	}
	negBuilder := networkendpointgroup.NewBuilder(negID)
	negBuilder.SetOwnership(rnode.OwnershipManaged)
	negBuilder.SetState(rnode.NodeExists)
	negBuilder.SetResource(negRes)
	graphBuilder.Add(negBuilder)
	return negID, nil
}

func buildBackendServiceWithNEG(graphBuilder *rgraph.Builder, name string, hcID, negID *cloud.ResourceID) (*cloud.ResourceID, error) {
	bsID := backendservice.ID(testFlags.project, meta.GlobalKey(resourceName(name)))

	bsMutResource := backendservice.NewMutableBackendService(testFlags.project, bsID.Key)
	bsMutResource.Access(func(x *compute.BackendService) {
		x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
		x.Protocol = "TCP"
		x.PortName = "http"
		x.Port = 80
		x.SessionAffinity = "NONE"
		x.TimeoutSec = 30
		x.Backends = []*compute.Backend{
			{
				Group:          negID.SelfLink(meta.VersionGA),
				BalancingMode:  "CONNECTION",
				MaxConnections: 10,
				CapacityScaler: 1,
			},
		}
		x.ConnectionDraining = &compute.ConnectionDraining{}
		x.HealthChecks = []string{hcID.SelfLink(meta.VersionGA)}
	})
	bsResource, err := bsMutResource.Freeze()
	if err != nil {
		return nil, err
	}

	bsBuilder := backendservice.NewBuilder(bsID)
	bsBuilder.SetOwnership(rnode.OwnershipManaged)
	bsBuilder.SetState(rnode.NodeExists)
	bsBuilder.SetResource(bsResource)

	graphBuilder.Add(bsBuilder)
	return bsID, nil
}

func buildTCPRoute(graphBuilder *rgraph.Builder, name, address, meshURL string, bsID *cloud.ResourceID) (*cloud.ResourceID, error) {
	tcpID := tcproute.ID(testFlags.project, meta.GlobalKey(resourceName(name)))
	tcpMutRes := tcproute.NewMutableTcpRoute(testFlags.project, tcpID.Key)

	tcpMutRes.Access(func(x *networkservices.TcpRoute) {
		x.Description = "tcp route for rGraph test"
		x.Name = tcpID.Key.Name
		x.Meshes = []string{meshURL}
		x.Rules = []*networkservices.TcpRouteRouteRule{
			{
				Action: &networkservices.TcpRouteRouteAction{
					Destinations: []*networkservices.TcpRouteRouteDestination{
						{
							ServiceName: resourceSelfLink(bsID),
							Weight:      10,
						},
					},
				},
				Matches: []*networkservices.TcpRouteRouteMatch{
					{
						Address: address,
						Port:    "80",
					},
				},
			},
		}
	})

	tcpRes, err := tcpMutRes.Freeze()
	if err != nil {
		return nil, err
	}

	tcpRouteBuilder := tcproute.NewBuilder(tcpID)
	tcpRouteBuilder.SetOwnership(rnode.OwnershipManaged)
	tcpRouteBuilder.SetState(rnode.NodeExists)
	tcpRouteBuilder.SetResource(tcpRes)

	graphBuilder.Add(tcpRouteBuilder)
	return tcpID, nil
}

type routesServices struct {
	bsID    *cloud.ResourceID
	address string
}

func buildTCPRouteWithBackends(graphBuilder *rgraph.Builder, name, meshURL string, services []routesServices) (*cloud.ResourceID, error) {
	tcpID := tcproute.ID(testFlags.project, meta.GlobalKey(resourceName(name)))
	tcpMutRes := tcproute.NewMutableTcpRoute(testFlags.project, tcpID.Key)

	tcpMutRes.Access(func(x *networkservices.TcpRoute) {
		x.Description = "tcp route for rGraph test"
		x.Name = tcpID.Key.Name
		x.Meshes = []string{meshURL}
		for _, route := range services {
			tcpRoute := networkservices.TcpRouteRouteRule{
				Action: &networkservices.TcpRouteRouteAction{
					Destinations: []*networkservices.TcpRouteRouteDestination{
						{
							ServiceName: resourceSelfLink(route.bsID),
							Weight:      10,
						},
					},
				},
				Matches: []*networkservices.TcpRouteRouteMatch{
					{
						Address: route.address,
						Port:    "80",
					},
				},
			}

			x.Rules = append(x.Rules, &tcpRoute)
		}
	})

	tcpRes, err := tcpMutRes.Freeze()
	if err != nil {
		return nil, err
	}

	tcpRouteBuilder := tcproute.NewBuilder(tcpID)
	tcpRouteBuilder.SetOwnership(rnode.OwnershipManaged)
	tcpRouteBuilder.SetState(rnode.NodeExists)
	tcpRouteBuilder.SetResource(tcpRes)

	graphBuilder.Add(tcpRouteBuilder)
	return tcpID, nil
}

func ensureMesh(ctx context.Context, t *testing.T) (string, *meta.Key) {
	meshKey := meta.GlobalKey(resourceName(meshName))
	mesh, err := theCloud.Meshes().Get(ctx, meshKey)
	if err != nil {
		if cerrors.IsGoogleAPINotFound(err) {
			// Mesh not found create one
			meshLocal := networkservices.Mesh{
				Name: resourceName(meshName),
			}
			t.Logf("Insert mesh %v", meshLocal)
			err = theCloud.Meshes().Insert(ctx, meshKey, &meshLocal)
			if err != nil {
				t.Fatalf("theCloud.Meshes().Insert(_, %v, %+v) = %v, want nil", meshKey, meshLocal, err)
			}
			mesh, err = theCloud.Meshes().Get(ctx, meshKey)
			if err != nil {
				t.Fatalf("theCloud.Meshes().Get(_, %v) = %v, want nil", meshKey, err)
			}
		} else {
			t.Fatalf("theCloud.Meshes().Get(_, %s) = %v, want nil", meshKey, err)
		}
	}
	return mesh.SelfLink, meshKey
}

func TestRgraphTCPRouteAddBackends(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	meshURL, meshKey := ensureMesh(ctx, t)
	t.Cleanup(func() {
		err := theCloud.Meshes().Delete(ctx, meshKey)
		t.Logf("theCloud.Meshes().Delete(ctx, %s): %v", meshKey, err)
	})
	graphBuilder := rgraph.NewBuilder()
	negID, err := buildNEG(graphBuilder, "neg-test", zone)
	if err != nil {
		t.Fatalf("buildNEG(_, neg-test, %s) = (_, %v), want (_, nil)", zone, err)
	}
	t.Cleanup(func() {
		err := theCloud.NetworkEndpointGroups().Delete(ctx, negID.Key)
		t.Logf("theCloud.NetworkEndpointGroups().Delete(ctx, %s): %v", negID.Key, err)
	})

	hcID, err := buildHealthCheck(graphBuilder, "hc-test", 15)
	if err != nil {
		t.Fatalf("buildHealthCheck(_, hc-test, 15) = (_, %v), want (_, nil)", err)
	}
	t.Cleanup(func() {
		err := theCloud.HealthChecks().Delete(ctx, hcID.Key)
		t.Logf("theCloud.HealthChecks().Delete(ctx, %s): %v", hcID.Key, err)
	})
	bsID, err := buildBackendServiceWithNEG(graphBuilder, "bs-test", hcID, negID)
	t.Logf("BackendServices created: %v", bsID)
	if err != nil {
		t.Fatalf("buildBackendServiceWithNEG(_, bs-test, _, _) = (_, %v), want (_, nil)", err)
	}
	t.Cleanup(func() {
		err = theCloud.BackendServices().Delete(ctx, bsID.Key)
		t.Logf("theCloud.BackendServices().Delete(_, %s): %v", bsID.Key, err)
	})
	tcprID, err := buildTCPRoute(graphBuilder, "tcproute-test", routeCIDR, meshURL, bsID)
	if err != nil {
		t.Fatalf("buildTCPRoute(_, tcproute-test, _, _, _) = (_, %v), want (_, nil)", err)
	}
	t.Logf("TCPRoute created: %v", tcprID)
	t.Cleanup(func() {
		err := theCloud.TcpRoutes().Delete(ctx, tcprID.Key)
		t.Logf("theCloud.TcpRoutes().Delete(_, %s): %v", tcprID.Key, err)
	})

	expectedActions := []exec.ActionMetadata{
		{Type: exec.ActionTypeCreate, Name: actionName(exec.ActionTypeCreate, tcprID)},
		{Type: exec.ActionTypeCreate, Name: actionName(exec.ActionTypeCreate, bsID)},
		{Type: exec.ActionTypeCreate, Name: actionName(exec.ActionTypeCreate, hcID)},
		{Type: exec.ActionTypeCreate, Name: actionName(exec.ActionTypeCreate, negID)},
	}
	processGraphAndExpectActions(t, graphBuilder, expectedActions)

	checkGCEBackendService(t, ctx, theCloud, hcID, bsID, 80)
	checkAppNetTCPRoute(t, ctx, theCloud, tcprID.Key.Name, meshURL, bsID)
	negID2, err := buildNEG(graphBuilder, "neg-test-2", zone)
	if err != nil {
		t.Fatalf("buildNEG(_, neg-test-2, %s) = (_, %v), want (_, nil)", zone, err)
	}
	t.Cleanup(func() {
		err := theCloud.NetworkEndpointGroups().Delete(ctx, negID2.Key)
		t.Logf("theCloud.NetworkEndpointGroups().Delete(ctx, %s): %v", negID2.Key, err)
	})

	hcID2, err := buildHealthCheck(graphBuilder, "hc-test-2", 15)
	if err != nil {
		t.Fatalf("buildHealthCheck(_, hc-test-2, _) = (_, %v), want (_, nil)", err)
	}
	t.Cleanup(func() {
		err := theCloud.HealthChecks().Delete(ctx, hcID2.Key)
		t.Logf("theCloud.HealthChecks().Delete(ctx, %s): %v", hcID2.Key, err)
	})
	bsID2, err := buildBackendServiceWithNEG(graphBuilder, "bs-test-2", hcID2, negID2)
	if err != nil {
		t.Fatalf("buildBackendServiceWithNEG(_, bs-test-2, _, _) = (_, %v), want (_, nil)", err)
	}
	t.Cleanup(func() {
		err = theCloud.BackendServices().Delete(ctx, bsID2.Key)
		t.Logf("theCloud.BackendServices().Delete(ctx, %s): %v", bsID2.Key, err)
	})
	routes := []routesServices{
		{bsID, routeCIDR},
		{bsID2, routeCIRD2},
	}
	tcprID, err = buildTCPRouteWithBackends(graphBuilder, "tcproute-test", meshURL, routes)
	t.Cleanup(func() {
		err := theCloud.TcpRoutes().Delete(ctx, tcprID.Key)
		t.Logf("theCloud.TcpRoutes().Delete(ctx, %s): %v", tcprID.Key, err)
	})
	expectedActions = []exec.ActionMetadata{
		{Type: exec.ActionTypeUpdate, Name: actionName(exec.ActionTypeUpdate, tcprID)},
		{Type: exec.ActionTypeCreate, Name: actionName(exec.ActionTypeCreate, bsID2)},
		{Type: exec.ActionTypeCreate, Name: actionName(exec.ActionTypeCreate, hcID2)},
		{Type: exec.ActionTypeCreate, Name: actionName(exec.ActionTypeCreate, negID2)},
		{Type: exec.ActionTypeMeta, Name: eventName(negID)},
		{Type: exec.ActionTypeMeta, Name: eventName(bsID)},
		{Type: exec.ActionTypeMeta, Name: eventName(hcID)},
	}
	processGraphAndExpectActions(t, graphBuilder, expectedActions)
	checkGCEBackendService(t, ctx, theCloud, hcID2, bsID2, 80)
	checkAppNetTCPRoute(t, ctx, theCloud, tcprID.Key.Name, meshURL, bsID, bsID2)
}

func actionName(actionType exec.ActionType, id *cloud.ResourceID) string {
	return fmt.Sprintf("Generic%sAction(%s)", actionType, id)
}

func processGraphAndExpectActions(t *testing.T, graphBuilder *rgraph.Builder, expectedActions []exec.ActionMetadata) {
	t.Helper()
	ctx := context.Background()
	graph, err := graphBuilder.Build()
	if err != nil {
		t.Fatalf("graphBuilder.Build() = %v, want nil", err)
	}

	result, err := plan.Do(ctx, theCloud, graph)
	if err != nil {
		t.Fatalf("plan.Do(_, _, _) = %v, want nil", err)
	}

	t.Logf("\nPlan.Actions: %v", result.Actions)
	t.Logf("\nPlan.Got: %v", result.Got)
	t.Logf("\nPlan.Want: %v", result.Want)

	err = expectActions(result.Actions, expectedActions)
	if err != nil {
		t.Fatalf("expectActions(_, _) = %v, want nil", err)
	}

	ex, err := exec.NewSerialExecutor(result.Actions)
	if err != nil {
		t.Logf("exec.NewSerialExecutor err: %v", err)
		return
	}
	res, err := ex.Run(context.Background(), theCloud)
	if err != nil || res == nil {
		t.Errorf("ex.Run(_,_) = ( %v, %v), want (*result, nil)", res, err)
	}
	t.Logf("exec got Result.Completed len(%d) =\n%v", len(res.Completed), res.Completed)
	t.Logf("exec got Result.Errors len(%d) =\n%v", len(res.Errors), res.Errors)
	t.Logf("exec got Result.Pending len(%d) =\n%v", len(res.Pending), res.Pending)
}
