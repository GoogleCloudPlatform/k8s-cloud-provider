package e2e

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/backendservice"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/workflow/plan"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kr/pretty"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/networkservices/v1"
)

func buildBackendServiceWithLBScheme(graphBuilder *rgraph.Builder, name string, hcID *cloud.ResourceID, lbScheme string) (*cloud.ResourceID, error) {
	bsID := backendservice.ID(testFlags.project, meta.GlobalKey(name))

	bsMutResource := backendservice.NewMutableBackendService(testFlags.project, bsID.Key)
	bsMutResource.Access(func(x *compute.BackendService) {
		x.LoadBalancingScheme = lbScheme
		x.Protocol = "TCP"
		x.PortName = "http"
		x.SessionAffinity = "NONE"
		x.Port = 80
		x.TimeoutSec = 30
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

func checkTCPRoute(t *testing.T, ctx context.Context, cloud cloud.Cloud, tcprID *cloud.ResourceID, rulesToBs [][]string) {
	t.Helper()
	t.Log("---- Check TCP Route ---- ")
	tcpr, err := cloud.TcpRoutes().Get(ctx, tcprID.Key)
	if err != nil {
		t.Fatalf("cloud.TCPRoutes().Get(_, %s) = %v, want nil", tcprID.Key, err)
	}
	less := func(x, y string) bool { return x < y }
	for i, r := range tcpr.Rules {
		dests := r.Action.Destinations
		srvcs := make([]string, len(dests))
		for i, dst := range dests {
			srvcs[i] = dst.ServiceName
		}
		if df := cmp.Diff(srvcs, rulesToBs[i], cmpopts.SortSlices(less)); df != "" {
			t.Fatalf("Rule %d with action %+v  points to incorrect backend services, diff: %s , want nil", i, r.Action, df)
		}
	}
}

func checkBackendService(t *testing.T, ctx context.Context, cloud cloud.Cloud, bsID *cloud.ResourceID, wantBS *compute.BackendService, comparer cmp.Option) {
	t.Helper()
	t.Log("---- Check BackendService ----")
	gotBS, err := cloud.BackendServices().Get(ctx, bsID.Key)
	if err != nil {
		t.Fatalf("cloud.HealthChecks().Get(_, %s) = %v, want nil", bsID.Key, err)
	}

	if df := cmp.Diff(gotBS, wantBS, comparer); df != "" {
		t.Fatalf("Backend Service %+v is different than desired %+v , diff: %s , want nil", gotBS, wantBS, df)
	}
}
func TestBackendServiceUpdate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	graphBuilder := rgraph.NewBuilder()
	meshURL, meshKey := ensureMesh(ctx, t, "test-bs-mesh")
	t.Cleanup(func() {
		err := theCloud.Meshes().Delete(ctx, meshKey)
		t.Logf("theCloud.Meshes().Delete(ctx, %s): %v", meshKey, err)
	})

	hc1ID, err := buildHealthCheck(graphBuilder, "hc1-test", 15)
	if err != nil {
		t.Fatalf("buildHealthCheck(_, hc1-test, 15) = (_, %v), want (_, nil)", err)
	}
	hc2ID, err := buildHealthCheck(graphBuilder, "hc2-test", 15)
	if err != nil {
		t.Fatalf("buildHealthCheck(_, hc2-test, 15) = (_, %v), want (_, nil)", err)
	}

	bs1Name := resourceName("bs1-e2e")
	bs2Name := resourceName("bs2-e2e")

	bs1ID, err := buildBackendServiceWithLBScheme(graphBuilder, bs1Name, hc1ID, "INTERNAL_SELF_MANAGED")
	if err != nil {
		t.Fatalf("buildBackendServiceWithLBScheme(_, %s, _) = %v, want nil", bs1Name, err)
	}

	bs2ID, err := buildBackendServiceWithLBScheme(graphBuilder, bs2Name, hc2ID, "INTERNAL_SELF_MANAGED")
	if err != nil {
		t.Fatalf("buildBackendServiceWithLBScheme(_, %s, _) = %v, want nil", bs2Name, err)
	}

	rules := []*networkservices.TcpRouteRouteRule{
		{
			Action: &networkservices.TcpRouteRouteAction{
				Destinations: []*networkservices.TcpRouteRouteDestination{
					{
						ServiceName: resourceSelfLink(bs1ID),
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
		{
			Action: &networkservices.TcpRouteRouteAction{
				Destinations: []*networkservices.TcpRouteRouteDestination{
					{
						ServiceName: resourceSelfLink(bs2ID),
						Weight:      10,
					},
				},
			},
			Matches: []*networkservices.TcpRouteRouteMatch{
				{
					Address: routeCIRD2,
					Port:    "80",
				},
			},
		},
	}
	tcpr, err := buildTCPRoute(graphBuilder, "test-route", meshURL, rules, bs1ID)
	if err != nil {
		t.Fatalf("buildTcpRoute(_, test-route, %s, %v, %s) = %v, want nil", meshURL, rules, bs1ID, err)
	}

	t.Logf("tcpr = %s", pretty.Sprint(tcpr))

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
	t.Cleanup(func() {
		err := theCloud.TcpRoutes().Delete(ctx, tcpr.Key)
		if err != nil {
			t.Logf("delete TCProute: %v", err)
		}
		err = theCloud.BackendServices().Delete(ctx, bs1ID.Key)
		if err != nil {
			t.Logf("delete backend service: %v", err)
		}
		err = theCloud.BackendServices().Delete(ctx, bs2ID.Key)
		if err != nil {
			t.Logf("delete backend service: %v", err)
		}
		err = theCloud.HealthChecks().Delete(ctx, hc1ID.Key)
		t.Logf("theCloud.HealthChecks().Delete(ctx, %s): %v", hc1ID.Key, err)
		err = theCloud.HealthChecks().Delete(ctx, hc2ID.Key)
		t.Logf("theCloud.HealthChecks().Delete(ctx, %s): %v", hc2ID.Key, err)
	})
	rulesToBs := [][]string{{resourceSelfLink(bs1ID)}, {resourceSelfLink(bs2ID)}}
	checkTCPRoute(t, ctx, theCloud, tcpr, rulesToBs)
	compareLBScheme := cmp.Comparer(func(a, b *compute.BackendService) bool {
		return cmp.Equal(a.LoadBalancingScheme, b.LoadBalancingScheme)
	})
	wantBS := &compute.BackendService{LoadBalancingScheme: "INTERNAL_SELF_MANAGED"}
	checkBackendService(t, ctx, theCloud, bs1ID, wantBS, compareLBScheme)

	// update backend service
	bs1ID, err = buildBackendServiceWithLBScheme(graphBuilder, bs1Name, hc1ID, "INTERNAL_MANAGED")

	graph, err = graphBuilder.Build()
	if err != nil {
		t.Fatalf("graphBuilder.Build() = %v, want nil", err)
	}
	result, err = plan.Do(ctx, theCloud, graph)
	if err != nil {
		t.Fatalf("plan.Do(_, _, _) = %v, want nil", err)
	}

	//TODO(kl52752) Change the expectation when the tcp route won't be recreated

	expectedActions := []exec.ActionMetadata{
		{Type: exec.ActionTypeMeta, Name: eventName(bs2ID)},
		{Type: exec.ActionTypeMeta, Name: eventName(hc1ID)},
		{Type: exec.ActionTypeMeta, Name: eventName(hc2ID)},
		{Type: exec.ActionTypeCreate, Name: actionName(exec.ActionTypeCreate, bs1ID)},
		{Type: exec.ActionTypeCreate, Name: actionName(exec.ActionTypeCreate, tcpr)},
		{Type: exec.ActionTypeDelete, Name: actionName(exec.ActionTypeDelete, bs1ID)},
		{Type: exec.ActionTypeDelete, Name: actionName(exec.ActionTypeDelete, tcpr)},
	}

	processGraphAndExpectActions(t, graphBuilder, expectedActions)

	wantBS = &compute.BackendService{LoadBalancingScheme: "INTERNAL_MANAGED"}
	checkBackendService(t, ctx, theCloud, bs1ID, wantBS, compareLBScheme)
}
