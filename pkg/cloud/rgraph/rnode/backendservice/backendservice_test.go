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
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

const proj = "proj-1"

func TestBackendServiceSchema(t *testing.T) {
	key := meta.GlobalKey("key-1")
	x := NewMutableBackendService(proj, key)
	if err := x.CheckSchema(); err != nil {
		t.Fatalf("CheckSchema() = %v, want nil", err)
	}
}

func TestActionUpdate(t *testing.T) {
	bsID := ID(proj, meta.GlobalKey("bs-name"))
	// hcID := healthcheck.ID(testFlags.project, meta.GlobalKey("hc-name"))
	bsMutResource := NewMutableBackendService(proj, bsID.Key)
	bsMutResource.Access(func(x *compute.BackendService) {
		x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
		x.Protocol = "TCP"
		x.Port = 80
		x.HealthChecks = []string{"https://www.googleapis.com/compute/v1/projects/proj-1/global/healthChecks/hcName"}
	})
	bsResource, err := bsMutResource.Freeze()
	if err != nil {
		t.Fatalf("bsMutResource.Freeze() = %v, want nil", err)
	}

	bsBuilder := NewBuilder(bsID)
	bsBuilder.SetOwnership(rnode.OwnershipManaged)
	bsBuilder.SetState(rnode.NodeExists)
	bsBuilder.SetResource(bsResource)
	bsNode, err := bsBuilder.Build()
	if err != nil {
		t.Fatalf("bsBuilder.Build() = %v, want nil", err)
	}
	gotNode := bsNode.(*backendServiceNode)
	fingerprint := "AAAAA"
	actions, err := rnode.UpdateActions[compute.BackendService, alpha.BackendService, beta.BackendService](&ops{}, bsNode, gotNode, gotNode.resource, fingerprint)
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
