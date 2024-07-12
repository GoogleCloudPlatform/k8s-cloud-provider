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
	"errors"
	"fmt"
	"strconv"
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

func buildTCPRoute(graphBuilder *rgraph.Builder, name, meshURL string, rules []*networkservices.TcpRouteRouteRule) (*cloud.ResourceID, error) {
	tcpID := tcproute.ID(testFlags.project, meta.GlobalKey(resourceName(name)))
	tcpMutRes := tcproute.NewMutableTcpRoute(testFlags.project, tcpID.Key)

	tcpMutRes.Access(func(x *networkservices.TcpRoute) {
		x.Description = "tcp route for rGraph test"
		x.Name = tcpID.Key.Name
		x.Meshes = []string{meshURL}
		x.Rules = rules
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

// meshName must be unique per test for tests isolation.
// TODO: fix ensureMesh so it returns a mesh with hash suffix added to the mesh
func ensureMesh(ctx context.Context, t *testing.T, meshName string) (string, *meta.Key) {
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

	meshURL, meshKey := ensureMesh(ctx, t, "test-mesh")
	t.Cleanup(func() {
		err := theCloud.Meshes().Delete(ctx, meshKey)
		t.Logf("theCloud.Meshes().Delete(ctx, %s): %v", meshKey, err)
	})
	graphBuilder := rgraph.NewBuilder()
	negID, err := buildNEG(graphBuilder, "neg-test", zone)
	if err != nil {
		t.Fatalf("buildNEG(_, neg-test, %s) = (_, %v), want (_, nil)", zone, err)
	}

	hcID, err := buildHealthCheck(graphBuilder, "hc-test", 15)
	if err != nil {
		t.Fatalf("buildHealthCheck(_, hc-test, 15) = (_, %v), want (_, nil)", err)
	}

	bsID, err := buildBackendServiceWithNEG(graphBuilder, "bs-test", hcID, negID)
	t.Logf("BackendServices created: %v", bsID)
	if err != nil {
		t.Fatalf("buildBackendServiceWithNEG(_, bs-test, _, _) = (_, %v), want (_, nil)", err)
	}

	rules := []*networkservices.TcpRouteRouteRule{
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
					Address: routeCIDR,
					Port:    "80",
				},
			},
		},
	}

	tcprID, err := buildTCPRoute(graphBuilder, "tcproute-test", meshURL, rules)
	if err != nil {
		t.Fatalf("buildTCPRoute(_, tcproute-test, _, _, _) = (_, %v), want (_, nil)", err)
	}
	t.Logf("TCPRoute created: %v", tcprID)

	expectedActions := []exec.ActionMetadata{
		{Type: exec.ActionTypeCreate, Name: actionName(exec.ActionTypeCreate, tcprID)},
		{Type: exec.ActionTypeCreate, Name: actionName(exec.ActionTypeCreate, bsID)},
		{Type: exec.ActionTypeCreate, Name: actionName(exec.ActionTypeCreate, hcID)},
		{Type: exec.ActionTypeCreate, Name: actionName(exec.ActionTypeCreate, negID)},
	}

	processGraphAndExpectActions(t, graphBuilder, expectedActions)
	t.Cleanup(func() {
		var err error
		err = theCloud.TcpRoutes().Delete(ctx, tcprID.Key)
		t.Logf("theCloud.TcpRoutes().Delete(_, %s): %v", tcprID.Key, err)
		err = theCloud.BackendServices().Delete(ctx, bsID.Key)
		t.Logf("theCloud.BackendServices().Delete(_, %s): %v", bsID.Key, err)
		err = theCloud.NetworkEndpointGroups().Delete(ctx, negID.Key)
		t.Logf("theCloud.NetworkEndpointGroups().Delete(ctx, %s): %v", negID.Key, err)
		err = theCloud.HealthChecks().Delete(ctx, hcID.Key)
		t.Logf("theCloud.HealthChecks().Delete(ctx, %s): %v", hcID.Key, err)
	})

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

	ex, err := exec.NewSerialExecutor(theCloud, result.Actions)
	if err != nil {
		t.Fatalf("exec.NewSerialExecutor err: %v", err)
	}
	res, err := ex.Run(context.Background())
	if err != nil || res == nil {
		t.Errorf("ex.Run(_,_) = ( %v, %v), want (*result, nil)", res, err)
	}
	t.Logf("exec got Result.Completed len(%d) =\n%v", len(res.Completed), res.Completed)
	t.Logf("exec got Result.Errors len(%d) =\n%v", len(res.Errors), res.Errors)
	t.Logf("exec got Result.Pending len(%d) =\n%v", len(res.Pending), res.Pending)
}

func defaultNetworkURL() string {
	return cloud.NewNetworksResourceID(testFlags.project, "default").SelfLink(meta.VersionGA)
}

func defaultSubnetworkURL() string {
	return cloud.NewSubnetworksResourceID(testFlags.project, region, "default").SelfLink(meta.VersionGA)
}

func createManyHealthchecks(graphBuilder *rgraph.Builder, hcNum int, name string) ([]*cloud.ResourceID, error) {
	var hcs []*cloud.ResourceID
	var e error
	for i := 0; i < hcNum; i++ {
		hc, err := buildHealthCheck(graphBuilder, "hc-"+name+"-"+strconv.Itoa(i), 15)
		errors.Join(e, err)
		hcs = append(hcs, hc)
	}
	return hcs, e
}

func createManyBackendServicesWithHC(graphBuilder *rgraph.Builder, bsNum int, name string, hcs []*cloud.ResourceID) ([]*cloud.ResourceID, error) {
	hcNum := len(hcs)
	if len(hcs) < bsNum {
		return nil, fmt.Errorf("createManyBackendServicesWithHC: not enough healthchecks: want %d, got %d", bsNum, hcNum)
	}
	var bss []*cloud.ResourceID
	var e error
	for i := 0; i < bsNum; i++ {
		bs, err := buildBackendServiceWithLBScheme(graphBuilder, name+"-"+strconv.Itoa(i)+"-bs", hcs[i], "INTERNAL_SELF_MANAGED")
		errors.Join(e, err)
		bss = append(bss, bs)
	}
	return bss, e
}

func createTcpRule(bsID *cloud.ResourceID, routeCIDR string) *networkservices.TcpRouteRouteRule {
	return &networkservices.TcpRouteRouteRule{
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
				Address: routeCIDR,
				Port:    "80",
			},
		},
	}
}

func createTCPRoutes(t *testing.T, graphBuilder *rgraph.Builder, numTCPR int, namePrefix string, meshURL string, bss []*cloud.ResourceID) ([]*cloud.ResourceID, error) {

	bsNum := len(bss)
	if len(bss) < 2*numTCPR {
		return nil, fmt.Errorf("ccreateTCPRoutes: not enough BackendServices: want %d, got %d", bsNum, 2*numTCPR)
	}
	var tcprs []*cloud.ResourceID
	var e error
	for i := 0; i < 2*numTCPR; i += 2 {
		cidr1, cidr2 := "10.240."+strconv.Itoa(i)+".83/32", "10.240."+strconv.Itoa(i+1)+".83/32"
		tcprRules := []*networkservices.TcpRouteRouteRule{
			createTcpRule(bss[i], cidr1),
			createTcpRule(bss[i+1], cidr2),
		}
		name := namePrefix + "-" + strconv.Itoa(i/2)
		tcpr, err := buildTCPRoute(graphBuilder, name, meshURL, tcprRules)
		if err != nil {
			errors.Join(e, fmt.Errorf("buildTcpRoute(_, %s, %s, %v) = %v, want nil", name, meshURL, tcprRules, err))
		}
		tcprs = append(tcprs, tcpr)
		t.Logf("%s = %s", name, pretty.Sprint(tcpr))
	}
	return tcprs, e
}
func TestMeshWithMultipleTCPRoutes(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	graphBuilder := rgraph.NewBuilder()
	resUniqueIdPart := "multiple-tcpr"
	meshURL, meshKey := ensureMesh(ctx, t, "multiple-tcpr-mesh")

	t.Cleanup(func() {
		err := theCloud.Meshes().Delete(ctx, meshKey)
		t.Logf("theCloud.Meshes().Delete(ctx, %s): %v", meshKey, err)
	})

	hcs, err := createManyHealthchecks(graphBuilder, 6, resUniqueIdPart)
	if err != nil {
		t.Fatalf("createManyHealthchecks(_, 6, %s) = (_, %v), want (_, nil)", resUniqueIdPart, err)
	}

	bss, err := createManyBackendServicesWithHC(graphBuilder, 6, resUniqueIdPart, hcs)
	if err != nil {
		t.Fatalf("createManyBackendServicesWithHC(_, 6, %s) = (_, %v), want (_, nil)", resUniqueIdPart, err)
	}

	tcprs, err := createTCPRoutes(t, graphBuilder, 3, resUniqueIdPart, meshURL, bss)

	graph, err := graphBuilder.Build()
	if err != nil {
		t.Fatalf("graphBuilder.Build() = %v, want nil", err)
	}

	result, err := plan.Do(ctx, theCloud, graph)
	if err != nil {
		t.Fatalf("plan.Do(_, _, _) = %v, want nil", err)
	}

	ex, err := exec.NewSerialExecutor(theCloud, result.Actions)
	if err != nil {
		t.Fatalf("exec.NewSerialExecutor(_, _) err: %v", err)
		return
	}
	res, err := ex.Run(ctx)
	if err != nil || res == nil {
		t.Errorf("ex.Run(_,_) = %v, want nil", err)
	}

	for i := 0; i < len(hcs); i++ {
		hcKey := hcs[i].Key
		_, err = theCloud.HealthChecks().Get(ctx, hcKey)
		if err != nil {
			t.Fatalf("theCloud.Healthchecks().Get(_, %s) = %v, want nil", hcKey, err)
		}
	}

	for i := 0; i < len(bss); i++ {
		bsKey := bss[i].Key
		_, err = theCloud.BackendServices().Get(ctx, bsKey)
		if err != nil {
			t.Fatalf("theCloud.BackendServices().Get(_, %s) = %v, want nil", bsKey, err)
		}
	}

	for i := 0; i < len(tcprs); i++ {
		tcprKey := tcprs[i].Key
		_, err = theCloud.TcpRoutes().Get(ctx, tcprKey)
		if err != nil {
			t.Fatalf("theCloud.TcpRoutes().Get(_, %s) = %v, want nil", tcprKey, err)
		}
	}

	t.Cleanup(func() {
		for _, r := range tcprs {
			err := theCloud.TcpRoutes().Delete(ctx, r.Key)
			if err != nil {
				t.Logf("delete TCProute: %v", err)
			}
		}

		for _, bs := range bss {
			err = theCloud.BackendServices().Delete(ctx, bs.Key)
			if err != nil {
				t.Logf("delete backend service: %v", err)
			}
		}
		for _, hc := range hcs {
			err = theCloud.HealthChecks().Delete(ctx, hc.Key)
			t.Logf("theCloud.HealthChecks().Delete(ctx, %s): %v", hc.Key, err)
		}
	})

	rules := []*networkservices.TcpRouteRouteRule{
		{
			Action: &networkservices.TcpRouteRouteAction{
				Destinations: []*networkservices.TcpRouteRouteDestination{
					{
						ServiceName: resourceSelfLink(bss[1]),
						Weight:      10,
					},
				},
			},
			Matches: []*networkservices.TcpRouteRouteMatch{
				{
					Address: "10.240.1.83/32",
					Port:    "80",
				},
			},
		},
	}
	// Update TCP rules by removing rule pointing to removed BackendService
	tcprs[0], err = buildTCPRoute(graphBuilder, "multiple-tcpr-0", meshURL, rules)
	if err != nil {
		t.Fatalf(fmt.Sprintf("buildTcpRoute(_, %s, multiple-tcpr-0, %v) = %v, want nil", meshURL, rules, err))
	}

	expectedActions := []exec.ActionMetadata{
		{Type: exec.ActionTypeMeta, Name: eventName(bss[0])},
		{Type: exec.ActionTypeMeta, Name: eventName(bss[1])},
		{Type: exec.ActionTypeMeta, Name: eventName(bss[2])},
		{Type: exec.ActionTypeMeta, Name: eventName(bss[3])},
		{Type: exec.ActionTypeMeta, Name: eventName(bss[4])},
		{Type: exec.ActionTypeMeta, Name: eventName(bss[5])},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[0])},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[1])},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[2])},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[3])},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[4])},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[5])},
		{Type: exec.ActionTypeMeta, Name: eventName(tcprs[1])},
		{Type: exec.ActionTypeMeta, Name: eventName(tcprs[2])},
		{Type: exec.ActionTypeUpdate, Name: actionName(exec.ActionTypeUpdate, tcprs[0])},
	}

	processGraphAndExpectActions(t, graphBuilder, expectedActions)

	updatedTcpr, err := theCloud.TcpRoutes().Get(ctx, tcprs[0].Key)
	if err != nil {
		t.Fatalf("theCloud.TcpRoutes().Get(_, %v) = nil, %v, want _, nil", tcprs[0].Key, err)
	}
	fmt.Printf("Updated tcproute rules: %v, rules len: %d", updatedTcpr.Rules[0], len(updatedTcpr.Rules))

	updatedRules := updatedTcpr.Rules
	if len(updatedRules) != 1 {
		t.Fatalf("theCloud.TcpRoutes().Get(_, %v).Rules = %v, want 1 rule", tcprs[0].Key, updatedRules)
	}
	// Remove one of BackendServices
	graphBuilder.Get(bss[0]).SetState(rnode.NodeDoesNotExist)
	graphBuilder.Get(hcs[0]).SetState(rnode.NodeDoesNotExist)
	removedBS, bss := bss[0], bss[1:]
	removedHC, hcs := hcs[0], hcs[1:]

	expectedActions = []exec.ActionMetadata{
		{Type: exec.ActionTypeMeta, Name: eventName(bss[0])},
		{Type: exec.ActionTypeMeta, Name: eventName(bss[1])},
		{Type: exec.ActionTypeMeta, Name: eventName(bss[2])},
		{Type: exec.ActionTypeMeta, Name: eventName(bss[3])},
		{Type: exec.ActionTypeMeta, Name: eventName(bss[4])},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[0])},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[1])},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[2])},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[3])},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[4])},
		{Type: exec.ActionTypeMeta, Name: eventName(tcprs[0])},
		{Type: exec.ActionTypeMeta, Name: eventName(tcprs[1])},
		{Type: exec.ActionTypeMeta, Name: eventName(tcprs[2])},
		{Type: exec.ActionTypeDelete, Name: actionName(exec.ActionTypeDelete, removedBS)},
		{Type: exec.ActionTypeDelete, Name: actionName(exec.ActionTypeDelete, removedHC)},
	}

	processGraphAndExpectActions(t, graphBuilder, expectedActions)

	bs, err := theCloud.BackendServices().Get(ctx, removedBS.Key)
	if err == nil {
		t.Fatalf("theCloud.BackendServices().Get(_, %v) = %v, nil, want err", removedBS.Key, bs)
	}

	graphBuilder.Get(tcprs[0]).SetState(rnode.NodeDoesNotExist)
	graphBuilder.Get(bss[0]).SetState(rnode.NodeDoesNotExist)
	graphBuilder.Get(hcs[0]).SetState(rnode.NodeDoesNotExist)

	removedBS1, bss := bss[0], bss[1:]
	removedHC1, hcs := hcs[0], hcs[1:]
	removedTcpr, tcprs := tcprs[0], tcprs[1:]

	expectedActions = []exec.ActionMetadata{
		{Type: exec.ActionTypeMeta, Name: eventName(bss[0])},
		{Type: exec.ActionTypeMeta, Name: eventName(removedBS)},
		{Type: exec.ActionTypeMeta, Name: eventName(bss[1])},
		{Type: exec.ActionTypeMeta, Name: eventName(bss[2])},
		{Type: exec.ActionTypeMeta, Name: eventName(bss[3])},
		{Type: exec.ActionTypeMeta, Name: eventName(removedHC)},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[0])},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[1])},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[2])},
		{Type: exec.ActionTypeMeta, Name: eventName(hcs[3])},
		{Type: exec.ActionTypeDelete, Name: actionName(exec.ActionTypeDelete, removedHC1)},
		{Type: exec.ActionTypeDelete, Name: actionName(exec.ActionTypeDelete, removedBS1)},
		{Type: exec.ActionTypeDelete, Name: actionName(exec.ActionTypeDelete, removedTcpr)},
		{Type: exec.ActionTypeMeta, Name: eventName(tcprs[0])},
		{Type: exec.ActionTypeMeta, Name: eventName(tcprs[1])},
	}

	processGraphAndExpectActions(t, graphBuilder, expectedActions)

	bs, err = theCloud.BackendServices().Get(ctx, removedBS1.Key)
	if err == nil {
		t.Fatalf("theCloud.BackendServices().Get(_, %v) = %v, nil, want err", removedBS1.Key, bs)
	}

	hc, err := theCloud.HealthChecks().Get(ctx, removedHC1.Key)
	if err == nil {
		t.Fatalf("theCloud.HealthChecks().Get(_, %v) = %v, nil, want err", removedHC1, hc)
	}

	tcpr, err := theCloud.TcpRoutes().Get(ctx, removedTcpr.Key)
	if err == nil {
		t.Fatalf("theCloud.TcpRoutes().Get(_, %v) = %v, nil, want err", removedTcpr.Key, tcpr)
	}
}
