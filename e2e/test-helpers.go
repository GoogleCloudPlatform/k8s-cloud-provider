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
	"net/http"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/api/googleapi"
)

// expectActions checks if got contains actions in want.
// Actions are compared using ActionMetadata where Summary is ignored.
func expectActions(got []exec.Action, want []exec.ActionMetadata) error {
	gotMap := make(map[string]exec.ActionType)

	for _, gotA := range got {
		gotMap[gotA.Metadata().Name] = gotA.Metadata().Type
	}

	wantMap := make(map[string]exec.ActionType)
	for _, wantMetadata := range want {
		wantMap[wantMetadata.Name] = wantMetadata.Type
	}

	if diff := cmp.Diff(gotMap, wantMap); diff != "" {
		return errors.New(diff)
	}
	return nil
}

func resourceSelfLink(id *cloud.ResourceID) string {
	apiGroup := meta.APIGroupCompute
	relName := cloud.RelativeResourceName(id.ProjectID, id.Resource, id.Key)
	prefix := fmt.Sprintf("https://%s.googleapis.com/v1", apiGroup)
	return prefix + "/" + relName
}

func checkGCEHealthCheck(t *testing.T, ctx context.Context, cloud cloud.Cloud, hcID *cloud.ResourceID, hcInterval int) {
	t.Helper()
	t.Log("---- Check Health Check ---- ")
	gotHC, err := cloud.HealthChecks().Get(ctx, hcID.Key)
	if err != nil {
		t.Fatalf("cloud.HealthChecks().Get(_, %s) = %v, want nil", hcID.Key, err)
	}
	if gotHC.CheckIntervalSec != int64(hcInterval) {
		t.Fatalf("gotHC.CheckIntervalSec mismatch got: %v want: %d", gotHC.CheckIntervalSec, hcInterval)
	}
}

func checkTCPGCEHealthCheck(t *testing.T, ctx context.Context, cloud cloud.Cloud, hcID *cloud.ResourceID, hcInterval int) {
	t.Helper()
	t.Log("---- Check TCP Health Check ---- ")
	gotHC, err := cloud.HealthChecks().Get(ctx, hcID.Key)
	if err != nil {
		t.Fatalf("cloud.HealthChecks().Get(_, %s) = %v, want nil", hcID.Key, err)
	}
	if gotHC.Type != "TCP" {
		t.Fatalf("gotHC.Type mismatch got: %v want: TCP", gotHC.Type)
	}
	if gotHC.CheckIntervalSec != int64(hcInterval) {
		t.Fatalf("gotHC.CheckIntervalSec mismatch got: %v want: %d", gotHC.CheckIntervalSec, hcInterval)
	}
}
func checkGCEBackendService(t *testing.T, ctx context.Context, cloud cloud.Cloud, hcID, bsID *cloud.ResourceID, bsPort int) {
	t.Helper()
	t.Log("---- Check BackendService ----")
	gotBS, err := cloud.BackendServices().Get(ctx, bsID.Key)
	if err != nil {
		t.Fatalf("cloud.HealthChecks().Get(_, %s) = %v, want nil", bsID.Key, err)
	}

	if gotBS.Port != int64(bsPort) {
		t.Errorf("BackendService port mismatch, got: %d want %d", gotBS.Port, bsPort)
	}
	if len(gotBS.HealthChecks) == 0 || gotBS.HealthChecks[0] != hcID.SelfLink(meta.VersionGA) {
		t.Fatalf("BackendService %s does not have health check %s set", bsID.Key, hcID.SelfLink(meta.VersionGA))
	}
}

func checkAppNetTCPRoute(t *testing.T, ctx context.Context, cloud cloud.Cloud, tcprName, meshURL string, svcIds ...*cloud.ResourceID) {
	t.Helper()
	t.Log("---- Check TCPRoute ----")
	tcprKey := meta.GlobalKey(tcprName)
	tcpRoute, err := cloud.TcpRoutes().Get(ctx, tcprKey)
	if err != nil {
		t.Fatalf("cloud.TcpRoutes().Get(%s) = %v", tcprKey, err)
	}
	if len(tcpRoute.Meshes) != 1 {
		t.Fatalf("len(tcpRoute.Meshes) mismatch, got %d want 1", len(tcpRoute.Meshes))
	}
	if tcpRoute.Meshes[0] != meshURL {
		t.Fatalf("len(tcpRoute.Meshes[0]) mismatch, got %s want %s", tcpRoute.Meshes[0], meshURL)
	}

	if len(tcpRoute.Rules) != len(svcIds) {
		t.Fatalf("len(tcpRoute.Rules) mismatch, got %d want %d", len(tcpRoute.Rules), len(svcIds))
	}
Outer:
	for _, rule := range tcpRoute.Rules {
		if len(rule.Action.Destinations) != 1 {
			t.Fatalf("len(rule.Action.Destinations) mismatch, got %d want 1", len(rule.Action.Destinations))
		}
		for _, svcId := range svcIds {
			if rule.Action.Destinations[0].ServiceName == resourceSelfLink(svcId) {
				continue Outer
			}
		}
		t.Fatalf("Rule ServiceName %s, not found in expected: %v", rule.Action.Destinations[0].ServiceName, svcIds)
	}
}

func IsNotFoundError(err error) bool {
	var gerr *googleapi.Error
	if errors.As(err, &gerr) {
		return gerr.Code == http.StatusNotFound
	}
	return false
}
