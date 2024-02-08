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
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/backendservice"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/healthcheck"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/workflow/plan"
	"google.golang.org/api/compute/v1"
)

func buildBackendService(graphBuilder *rgraph.Builder, name string, hcID *cloud.ResourceID, port int64) (*cloud.ResourceID, error) {
	bsID := backendservice.ID(testFlags.project, meta.GlobalKey(resourceName(name)))

	bsMutResource := backendservice.NewMutableBackendService(testFlags.project, bsID.Key)
	bsMutResource.Access(func(x *compute.BackendService) {
		x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
		x.Protocol = "TCP"
		x.Port = port
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

func buildHealthCheck(graphBuilder *rgraph.Builder, name string, checkIntervalSec int64) (*cloud.ResourceID, error) {
	hcID := healthcheck.ID(testFlags.project, meta.GlobalKey(resourceName(name)))
	hcMutRes := healthcheck.NewMutableHealthCheck(testFlags.project, hcID.Key)
	hcMutRes.Access(func(x *compute.HealthCheck) {
		x.CheckIntervalSec = checkIntervalSec
		x.HealthyThreshold = 5
		x.TimeoutSec = 6
		x.Type = "HTTP"
		x.HttpHealthCheck = &compute.HTTPHealthCheck{
			RequestPath: "/",
			Port:        int64(9376),
		}
	})
	hcRes, err := hcMutRes.Freeze()
	if err != nil {
		return nil, err
	}
	hcBuilder := healthcheck.NewBuilder(hcID)
	hcBuilder.SetOwnership(rnode.OwnershipManaged)
	hcBuilder.SetState(rnode.NodeExists)
	hcBuilder.SetResource(hcRes)
	graphBuilder.Add(hcBuilder)
	return hcID, nil
}

func TestHcUpdateWithBackendService(t *testing.T) {
	ctx := context.Background()
	graphBuilder := rgraph.NewBuilder()

	hcID, err := buildHealthCheck(graphBuilder, "hc-test", 15)
	if err != nil {
		t.Fatalf("buildHealthCheck(_, hc-test, _) = %v, want nil", err)
	}

	bsID, err := buildBackendService(graphBuilder, "bs-e2e", hcID, 80)
	if err != nil {
		t.Fatalf("buildBackendService(_, bs-e2e, _, 80) = %v, want nil", err)
	}

	graph, err := graphBuilder.Build()
	if err != nil {
		t.Fatalf("graphBuilder.Build() = %v, want nil", err)
	}
	result, err := plan.Do(ctx, theCloud, graph)
	if err != nil {
		t.Fatalf("plan.Do(_, _, _) = %v, want nil", err)
	}

	ex, err := exec.NewSerialExecutor(result.Actions)
	if err != nil {
		t.Logf("exec.NewSerialExecutor(_, _) err: %v", err)
		return
	}
	res, err := ex.Run(ctx, theCloud)
	if err != nil || res == nil {
		t.Errorf("ex.Run(_,_) = %v, want nil", err)
	}

	t.Cleanup(func() {
		err = theCloud.BackendServices().Delete(ctx, bsID.Key)
		if err != nil {
			t.Logf("delete backend service: %v", err)
		}
		err := theCloud.HealthChecks().Delete(ctx, hcID.Key)
		if err != nil {
			t.Logf("delete health check: %v", err)
		}
	})
	checkGCEResources(t, ctx, hcID, bsID, 15)

	// update health check
	hcID, err = buildHealthCheck(graphBuilder, "hc-test", 60)
	if err != nil {
		t.Fatalf("buildHealthCheck(_, hc-test, 60) = (_, %v), want (_, nil)", err)
	}
	graph, err = graphBuilder.Build()
	if err != nil {
		t.Fatalf("graphBuilder.Build() = %v, want nil", err)
	}
	result, err = plan.Do(ctx, theCloud, graph)
	if err != nil {
		t.Fatalf("plan.Do(_, _, _) = %v, want nil", err)
	}
	expectedActions := []exec.ActionType{
		exec.ActionTypeUpdate,
	}

	t.Logf("\nPlan.Actions: %v", result.Actions)
	t.Logf("\nPlan.Got: %v", result.Got)
	t.Logf("\nPlan.Want: %v", result.Want)

	err = expectActions(result.Actions, expectedActions)
	if err != nil {
		t.Fatalf("expectActions(_, _) = %v, want nil", err)
	}
	t.Log("\nstart NewSerialExecutor for update")
	ex, err = exec.NewSerialExecutor(result.Actions)
	if err != nil {
		t.Logf("exec.NewSerialExecutor err: %v", err)
		return
	}
	res, err = ex.Run(ctx, theCloud)
	if err != nil || res == nil {
		t.Errorf("ex.Run(_,_) = ( %v, %v), want (*result, nil)", res, err)
	}
	t.Logf("exec.NewSerialExecutor finished, res: %v", res)
	if len(res.Pending) > 0 {
		t.Errorf("Executor has pending actions: %v", res.Pending)
	}
	checkGCEResources(t, ctx, hcID, bsID, 60)
	// update health check and check that Exist event was propagated to parents
	hcID, err = buildHealthCheck(graphBuilder, "hc-test", 120)
	if err != nil {
		t.Fatalf("buildHealthCheck(_, hc-test, 120) = (_, %v), want (_, nil)", err)
	}
	// trigger BackendService recreation
	bsID, err = buildBackendService(graphBuilder, "bs-e2e", hcID, 100)
	if err != nil {
		t.Fatalf("buildBackendService(_, bs-e2e, _, 100) = (_, %v), want (_, nil)", err)
	}
	graph, err = graphBuilder.Build()
	if err != nil {
		t.Fatalf("graphBuilder.Build() = %v, want nil", err)
	}
	result, err = plan.Do(ctx, theCloud, graph)
	if err != nil {
		t.Fatalf("plan.Do(_, _, _) = %v, want nil", err)
	}
	// HealthCheck updated expect ActionUpdate
	// BackendService recreated expect Action Delete and Action Add
	expectedActions = []exec.ActionType{
		exec.ActionTypeUpdate,
		exec.ActionTypeCreate,
		exec.ActionTypeDelete,
	}

	t.Logf("\nPlan.Actions: %v", result.Actions)
	t.Logf("\nPlan.Got: %v", result.Got)
	t.Logf("\nPlan.Want: %v", result.Want)

	err = expectActions(result.Actions, expectedActions)
	if err != nil {
		t.Fatalf("expectActions(_, _) = %v, want nil", err)
	}
	t.Log("\nstart NewSerialExecutor for update")
	ex, err = exec.NewSerialExecutor(result.Actions)
	if err != nil {
		t.Logf("exec.NewSerialExecutor err: %v", err)
		return
	}
	res, err = ex.Run(ctx, theCloud)
	if err != nil || res == nil {
		t.Errorf("ex.Run(_,_) = ( %v, %v), want (*result, nil)", res, err)
	}
	t.Logf("exec.NewSerialExecutor finished, res: %v", res)
	if len(res.Pending) > 0 {
		t.Errorf("Executor has pending actions: %v", res.Pending)
	}
	checkGCEResources(t, ctx, hcID, bsID, 120)
}

func checkGCEResources(t *testing.T, ctx context.Context, hcID, bsID *cloud.ResourceID, hcInterval int) {
	t.Helper()
	t.Log("---- Check Health Check ---- ")
	gotHC, err := theCloud.HealthChecks().Get(ctx, hcID.Key)
	if err != nil {
		t.Fatalf("theCloud.HealthChecks().Get(_, %s) = %v, want nil", hcID.Key, err)
	}
	if gotHC.CheckIntervalSec != int64(hcInterval) {
		t.Fatalf("gotHC.CheckIntervalSec mismatch got: %v want: %d", gotHC.CheckIntervalSec, hcInterval)
	}
	t.Log("---- Check BackendService ----")
	gotBS, err := theCloud.BackendServices().Get(ctx, bsID.Key)
	if err != nil {
		t.Fatalf("theCloud.HealthChecks().Get(_, %s) = %v, want nil", bsID.Key, err)
	}

	if len(gotBS.HealthChecks) == 0 || gotBS.HealthChecks[0] != hcID.SelfLink(meta.VersionGA) {
		t.Fatalf("BackendService %s does not have health check %s set", bsID.Key, hcID.SelfLink(meta.VersionGA))
	}
}

func expectActions(got []exec.Action, want []exec.ActionType) error {
	var errs []error
WantLoop:
	for _, wantT := range want {
		for _, gotA := range got {
			if gotA.Metadata().Type == wantT {
				continue WantLoop
			}
		}
		errs = append(errs, fmt.Errorf("Not Found expected action %v", wantT))
	}
	return errors.Join(errs...)
}
