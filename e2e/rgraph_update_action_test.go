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
		x.PortName = "http"
		x.SessionAffinity = "NONE"
		x.Port = port
		x.TimeoutSec = 30
		x.HealthChecks = []string{hcID.SelfLink(meta.VersionGA)}
		x.ConnectionDraining = &compute.ConnectionDraining{}
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
		x.UnhealthyThreshold = 2
		x.HttpHealthCheck = &compute.HTTPHealthCheck{
			RequestPath: "/",
			Port:        int64(9376),
			ProxyHeader: "NONE",
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
func buildTCPHealthCheck(graphBuilder *rgraph.Builder, name string, checkIntervalSec int64) (*cloud.ResourceID, error) {
	hcID := healthcheck.ID(testFlags.project, meta.GlobalKey(resourceName(name)))
	hcMutRes := healthcheck.NewMutableHealthCheck(testFlags.project, hcID.Key)
	hcMutRes.Access(func(x *compute.HealthCheck) {
		x.CheckIntervalSec = checkIntervalSec
		x.HealthyThreshold = 5
		x.TimeoutSec = 6
		x.Type = "TCP"
		x.TcpHealthCheck = &compute.TCPHealthCheck{
			Port: 80,
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
	hcName := "hc-test"
	hcID, err := buildHealthCheck(graphBuilder, hcName, 15)
	if err != nil {
		t.Fatalf("buildHealthCheck(_, %s, _) = %v, want nil", hcName, err)
	}
	bsName := "bs-e2e"
	bsID, err := buildBackendService(graphBuilder, bsName, hcID, 80)
	if err != nil {
		t.Fatalf("buildBackendService(_, %s, _, 80) = %v, want nil", bsName, err)
	}

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
		t.Logf("exec.NewSerialExecutor(_, _) err: %v", err)
		return
	}
	res, err := ex.Run(ctx)
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
	checkGCEHealthCheck(t, ctx, theCloud, hcID, 15)
	checkGCEBackendService(t, ctx, theCloud, hcID, bsID, 80)

	// update health check
	hcID, err = buildHealthCheck(graphBuilder, hcName, 60)
	if err != nil {
		t.Fatalf("buildHealthCheck(_, %s, 60) = (_, %v), want (_, nil)", hcName, err)
	}
	graph, err = graphBuilder.Build()
	if err != nil {
		t.Fatalf("graphBuilder.Build() = %v, want nil", err)
	}
	result, err = plan.Do(ctx, theCloud, graph)
	if err != nil {
		t.Fatalf("plan.Do(_, _, _) = %v, want nil", err)
	}

	expectedActions := []exec.ActionMetadata{
		{Type: exec.ActionTypeUpdate, Name: actionName(exec.ActionTypeUpdate, hcID)},
		{Type: exec.ActionTypeMeta, Name: eventName(bsID)},
	}
	t.Logf("\nPlan.Actions: %v", result.Actions)
	t.Logf("\nPlan.Got: %v", result.Got)
	t.Logf("\nPlan.Want: %v", result.Want)

	err = expectActions(result.Actions, expectedActions)
	if err != nil {
		t.Fatalf("expectActions(_, _) = %v, want nil", err)
	}
	t.Log("\nstart NewSerialExecutor for update")
	ex, err = exec.NewSerialExecutor(theCloud, result.Actions)
	if err != nil {
		t.Logf("exec.NewSerialExecutor err: %v", err)
		return
	}
	res, err = ex.Run(ctx)
	if err != nil || res == nil {
		t.Errorf("ex.Run(_,_) = ( %v, %v), want (*result, nil)", res, err)
	}
	t.Logf("exec.NewSerialExecutor finished, res: %v", res)
	if len(res.Pending) > 0 {
		t.Errorf("Executor has pending actions: %v", res.Pending)
	}

	checkGCEHealthCheck(t, ctx, theCloud, hcID, 60)
	checkGCEBackendService(t, ctx, theCloud, hcID, bsID, 80)
	// update health check and check that Exist event was propagated to parents
	hcID, err = buildHealthCheck(graphBuilder, hcName, 120)
	if err != nil {
		t.Fatalf("buildHealthCheck(_, %s, 120) = (_, %v), want (_, nil)", hcName, err)
	}
	// update BackendService
	bsID, err = buildBackendService(graphBuilder, bsName, hcID, 100)
	if err != nil {
		t.Fatalf("buildBackendService(_, %s, _, 100) = (_, %v), want (_, nil)", bsName, err)
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
	expectedActions = []exec.ActionMetadata{
		{Type: exec.ActionTypeUpdate, Name: actionName(exec.ActionTypeUpdate, bsID)},
		{Type: exec.ActionTypeUpdate, Name: actionName(exec.ActionTypeUpdate, hcID)},
	}

	t.Logf("\nPlan.Actions: %v", result.Actions)
	t.Logf("\nPlan.Got: %v", result.Got)
	t.Logf("\nPlan.Want: %v", result.Want)

	err = expectActions(result.Actions, expectedActions)
	if err != nil {
		t.Fatalf("expectActions(_, _) = %v, want nil", err)
	}
	t.Log("\nstart NewSerialExecutor for update")
	ex, err = exec.NewSerialExecutor(theCloud, result.Actions)
	if err != nil {
		t.Logf("exec.NewSerialExecutor err: %v", err)
		return
	}
	res, err = ex.Run(ctx)
	if err != nil || res == nil {
		t.Errorf("ex.Run(_,_) = ( %v, %v), want (*result, nil)", res, err)
	}
	t.Logf("exec.NewSerialExecutor finished, res: %v", res)
	if len(res.Pending) > 0 {
		t.Errorf("Executor has pending actions: %v", res.Pending)
	}
	checkGCEHealthCheck(t, ctx, theCloud, hcID, 120)
	checkGCEBackendService(t, ctx, theCloud, hcID, bsID, 100)
}
func TestHcUpdateType(t *testing.T) {
	ctx := context.Background()
	graphBuilder := rgraph.NewBuilder()
	hcName := "hc-test"
	hcID, err := buildHealthCheck(graphBuilder, hcName, 15)
	if err != nil {
		t.Fatalf("buildHealthCheck(_, %s, _) = %v, want nil", hcName, err)
	}

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
		t.Logf("exec.NewSerialExecutor(_, _) err: %v", err)
		return
	}
	res, err := ex.Run(ctx)
	if err != nil || res == nil {
		t.Errorf("ex.Run(_,_) = %v, want nil", err)
	}

	t.Cleanup(func() {
		err := theCloud.HealthChecks().Delete(ctx, hcID.Key)
		if err != nil {
			t.Logf("delete health check: %v", err)
		}
	})
	checkGCEHealthCheck(t, ctx, theCloud, hcID, 15)

	// update health check
	hcID, err = buildTCPHealthCheck(graphBuilder, hcName, 60)
	if err != nil {
		t.Fatalf("buildTCPHealthCheck(_, %s, 60) = (_, %v), want (_, nil)", hcName, err)
	}
	graph, err = graphBuilder.Build()
	if err != nil {
		t.Fatalf("graphBuilder.Build() = %v, want nil", err)
	}
	result, err = plan.Do(ctx, theCloud, graph)
	if err != nil {
		t.Fatalf("plan.Do(_, _, _) = %v, want nil", err)
	}
	expectedActions := []exec.ActionMetadata{
		{Type: exec.ActionTypeUpdate, Name: actionName(exec.ActionTypeUpdate, hcID)},
	}

	t.Logf("\nPlan.Actions: %v", result.Actions)
	t.Logf("\nPlan.Got: %v", result.Got)
	t.Logf("\nPlan.Want: %v", result.Want)

	err = expectActions(result.Actions, expectedActions)
	if err != nil {
		t.Fatalf("expectActions(_, _) = %v, want nil", err)
	}
	t.Log("\nstart NewSerialExecutor for update")
	ex, err = exec.NewSerialExecutor(theCloud, result.Actions)
	if err != nil {
		t.Logf("exec.NewSerialExecutor err: %v", err)
		return
	}
	res, err = ex.Run(ctx)
	if err != nil || res == nil {
		t.Errorf("ex.Run(_,_) = ( %v, %v), want (*result, nil)", res, err)
	}
	t.Logf("exec.NewSerialExecutor finished, res: %v", res)
	if len(res.Pending) > 0 {
		t.Errorf("Executor has pending actions: %v", res.Pending)
	}

	checkTCPGCEHealthCheck(t, ctx, theCloud, hcID, 60)
}
func TestUpdateBackendService(t *testing.T) {
	ctx := context.Background()
	graphBuilder := rgraph.NewBuilder()
	hcName := "hc-update"
	hcID, err := buildHealthCheck(graphBuilder, hcName, 15)
	if err != nil {
		t.Fatalf("buildHealthCheck(_, %s, _) = %v, want nil", hcName, err)
	}
	bsName := "bs-update"
	bsID, err := buildBackendService(graphBuilder, bsName, hcID, 80)
	if err != nil {
		t.Fatalf("buildBackendService(_, %s, _, 80) = %v, want nil", bsName, err)
	}

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
		t.Logf("exec.NewSerialExecutor(_, _) err: %v", err)
		return
	}
	res, err := ex.Run(ctx)
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
	checkGCEBackendService(t, ctx, theCloud, hcID, bsID, 80)

	// update BackendService
	bsID, err = buildBackendService(graphBuilder, bsName, hcID, 100)
	if err != nil {
		t.Fatalf("buildBackendService(_, %s, _, 100) = (_, %v), want (_, nil)", bsName, err)
	}
	graph, err = graphBuilder.Build()
	if err != nil {
		t.Fatalf("graphBuilder.Build() = %v, want nil", err)
	}
	result, err = plan.Do(ctx, theCloud, graph)
	if err != nil {
		t.Fatalf("plan.Do(_, _, _) = %v, want nil", err)
	}

	expectedActions := []exec.ActionMetadata{
		{Type: exec.ActionTypeUpdate, Name: actionName(exec.ActionTypeUpdate, bsID)},
		{Type: exec.ActionTypeMeta, Name: eventName(hcID)},
	}

	t.Logf("\nPlan.Actions: %v", result.Actions)
	t.Logf("\nPlan.Got: %v", result.Got)
	t.Logf("\nPlan.Want: %v", result.Want)

	err = expectActions(result.Actions, expectedActions)
	if err != nil {
		t.Fatalf("expectActions(_, _) = %v, want nil", err)
	}
	t.Log("\nstart NewSerialExecutor for update")
	ex, err = exec.NewSerialExecutor(theCloud, result.Actions)
	if err != nil {
		t.Logf("exec.NewSerialExecutor err: %v", err)
		return
	}
	res, err = ex.Run(ctx)
	if err != nil || res == nil {
		t.Errorf("ex.Run(_,_) = ( %v, %v), want (*result, nil)", res, err)
	}
	t.Logf("exec.NewSerialExecutor finished, res: %v", res)
	if len(res.Pending) > 0 {
		t.Logf("Executor has %v, pending actions %v", len(res.Pending), res.Pending)
		res, err = ex.Run(ctx)
		if err != nil || res == nil {
			t.Errorf("ex.Run(_,_) = ( %v, %v), want (*result, nil)", res, err)
		}
	}
	checkGCEBackendService(t, ctx, theCloud, hcID, bsID, 100)
}

func eventName(id *cloud.ResourceID) string {
	return fmt.Sprintf("EventAction([Exists(%s)])", id)
}
