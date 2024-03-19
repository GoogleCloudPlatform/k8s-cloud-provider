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

package backendservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

const (
	proj       = "proj-1"
	hcSelfLink = "https://www.googleapis.com/compute/v1/projects/proj-1/global/healthChecks/hcName"
)

func TestBackendServiceSchema(t *testing.T) {
	key := meta.GlobalKey("key-1")
	x := NewMutableBackendService(proj, key)
	if err := x.CheckSchema(); err != nil {
		t.Fatalf("CheckSchema() = %v, want nil", err)
	}
}

func createBackendServiceNode(name string, mut func(x *compute.BackendService)) (*backendServiceNode, error) {
	bsID := ID(proj, meta.GlobalKey(name))
	bsMutResource := NewMutableBackendService(proj, bsID.Key)
	bsMutResource.Access(mut)
	bsResource, err := bsMutResource.Freeze()
	if err != nil {
		return nil, fmt.Errorf("bsMutResource.Freeze() = %v, want nil", err)
	}

	bsBuilder := NewBuilder(bsID)
	bsBuilder.SetOwnership(rnode.OwnershipManaged)
	bsBuilder.SetState(rnode.NodeExists)
	bsBuilder.SetResource(bsResource)
	bsNode, err := bsBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("bsBuilder.Build() = %v, want nil", err)
	}
	gotNode := bsNode.(*backendServiceNode)
	return gotNode, nil
}
func TestActionUpdate(t *testing.T) {
	modify := func(x *compute.BackendService) {
		x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
		x.Protocol = "TCP"
		x.Port = 80
		x.HealthChecks = []string{hcSelfLink}
	}

	gotNode, err := createBackendServiceNode("bs-name", modify)
	if err != nil {
		t.Fatalf("createBackendServiceNode(bs-name, _) = %v, want nil", err)
	}
	gotBs := gotNode.resource

	_, err = gotBs.ToGA()
	if err != nil {
		t.Errorf("gotBs.ToGA() = %v, want nil", err)
	}

	fingerprint := "AAAAA"
	actions, err := rnode.UpdateActions[compute.BackendService, alpha.BackendService, beta.BackendService](&ops{}, gotNode, gotNode, gotNode.resource, fingerprint)
	if err != nil {
		t.Fatalf("rnode.UpdateActions[]() = %v, want nil", err)
	}
	if len(actions) == 0 {
		t.Fatalf("no actions to update")
	}
	a := actions[0]
	mockCloud := cloud.NewMockGCE(&cloud.SingleProjectRouter{ID: proj})
	updateHook := func(ctx context.Context, key *meta.Key, bs *compute.BackendService, m *cloud.MockBackendServices, o ...cloud.Option) error {
		t.Logf("Update BS fingerprint %s", bs.Fingerprint)
		if bs.Fingerprint != fingerprint {
			t.Fatalf("Update BackendService: fingerprint mismatch got: %s, want %s", bs.Fingerprint, fingerprint)
		}
		return nil
	}
	mockCloud.MockBackendServices.UpdateHook = updateHook
	_, err = a.Run(context.Background(), mockCloud)
	if err != nil {
		t.Fatalf("a.Run(context.Background(), mockCloud) = %v, want nil", err)
	}

}

func TestBackendServiceDiff(t *testing.T) {
	bsName := "bs-name"
	for _, tc := range []struct {
		desc         string
		updateFn     func(x *compute.BackendService)
		expectedOp   rnode.Operation
		expectedDiff bool
	}{
		{
			desc: "No changes",
			updateFn: func(x *compute.BackendService) {
				x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
				x.Protocol = "TCP"
				x.Port = 80
				x.HealthChecks = []string{hcSelfLink}
			},
			expectedOp:   rnode.OpNothing,
			expectedDiff: false,
		},
		{
			desc: "expected recreation on internal schema change",
			updateFn: func(x *compute.BackendService) {
				x.LoadBalancingScheme = "EXTERNAL"
				x.Protocol = "TCP"
				x.Port = 90
				x.HealthChecks = []string{hcSelfLink}
			},
			expectedOp:   rnode.OpRecreate,
			expectedDiff: true,
		},
		{
			desc: "expected recreation on network change",
			updateFn: func(x *compute.BackendService) {
				x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
				x.Protocol = "TCP"
				x.Port = 90
				x.HealthChecks = []string{hcSelfLink}
				x.Network = "new-network"
			},
			expectedOp:   rnode.OpRecreate,
			expectedDiff: true,
		},
		{
			desc: "expected update on port change",
			updateFn: func(x *compute.BackendService) {
				x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
				x.Protocol = "TCP"
				x.Port = 123
				x.HealthChecks = []string{hcSelfLink}
			},
			expectedOp:   rnode.OpUpdate,
			expectedDiff: true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {

			create := func(x *compute.BackendService) {
				x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
				x.Protocol = "TCP"
				x.Port = 80
				x.HealthChecks = []string{hcSelfLink}
			}

			gotNode, err := createBackendServiceNode(bsName, create)
			if err != nil {
				t.Fatalf("createBackendServiceNode(%s, _) = %v, want nil", bsName, err)
			}
			wantBS, err := createBackendServiceNode(bsName, tc.updateFn)
			if err != nil {
				t.Fatalf("createBackendServiceNode(%s, _) = %v, want nil", bsName, err)
			}
			plan, err := gotNode.Diff(wantBS)
			if err != nil || plan == nil {
				t.Fatalf("gotNode.Diff(_) = (%v, %v), want plan,  nil", plan, err)
			}
			if plan.Operation != tc.expectedOp {
				t.Errorf("%v != %v", plan.Operation, tc.expectedOp)
			}

			if tc.expectedDiff && (plan.Diff == nil || len(plan.Diff.Items) == 0) {
				t.Errorf("Result did not returned diff")
			}
			t.Logf("Diff results %+v", plan)
		})
	}
}
