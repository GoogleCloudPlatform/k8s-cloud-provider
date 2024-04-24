package e2e

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"google.golang.org/api/networkservices/v1"
)

func TestHealthcheckUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	meshURL, meshKey := ensureMesh(ctx, t, "hc-update-test-mesh")
	t.Cleanup(func() {
		err := theCloud.Meshes().Delete(ctx, meshKey)
		t.Logf("theCloud.Meshes().Delete(ctx, %s): %v", meshKey, err)
	})

	graphBuilder := rgraph.NewBuilder()
	negID, err := buildNEG(graphBuilder, "neg-hc-update-test", zone)
	if err != nil {
		t.Fatalf("buildNEG(_, neg-test, %s) = (_, %v), want (_, nil)", zone, err)
	}
	t.Cleanup(func() {
		err := theCloud.NetworkEndpointGroups().Delete(ctx, negID.Key)
		t.Logf("theCloud.NetworkEndpointGroups().Delete(ctx, %s): %v", negID.Key, err)
	})

	hcID, err := buildHealthCheck(graphBuilder, "hc-update-test", 15)
	if err != nil {
		t.Fatalf("buildHealthCheck(_, hc-update-test, 15) = (_, %v), want (_, nil)", err)
	}
	t.Cleanup(func() {
		err := theCloud.HealthChecks().Delete(ctx, hcID.Key)
		t.Logf("theCloud.HealthChecks().Delete(ctx, %s): %v", hcID.Key, err)
	})
	bsID, err := buildBackendServiceWithNEG(graphBuilder, "hc-update-test-bs", hcID, negID)
	t.Logf("BackendServices created: %v", bsID)
	if err != nil {
		t.Fatalf("buildBackendServiceWithNEG(_, bs-test, _, _) = (_, %v), want (_, nil)", err)
	}
	t.Cleanup(func() {
		err = theCloud.BackendServices().Delete(ctx, bsID.Key)
		t.Logf("theCloud.BackendServices().Delete(_, %s): %v", bsID.Key, err)
	})
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
					Address: "10.240.15.83/32",
					Port:    "80",
				},
			},
		},
	}

	tcprID, err := buildTCPRoute(graphBuilder, "hc-update-test", meshURL, rules, bsID)
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

	checkGCEHealthCheck(t, ctx, theCloud, hcID, 15)
	hcID, err = buildHealthCheck(graphBuilder, "hc-update-test", 25)

	expectedActions = []exec.ActionMetadata{
		{Type: exec.ActionTypeUpdate, Name: actionName(exec.ActionTypeUpdate, hcID)},
		{Type: exec.ActionTypeUpdate, Name: actionName(exec.ActionTypeUpdate, tcprID)},
		{Type: exec.ActionTypeMeta, Name: eventName(negID)},
		{Type: exec.ActionTypeMeta, Name: eventName(bsID)},
	}
	processGraphAndExpectActions(t, graphBuilder, expectedActions)
	checkGCEHealthCheck(t, ctx, theCloud, hcID, 25)
}
