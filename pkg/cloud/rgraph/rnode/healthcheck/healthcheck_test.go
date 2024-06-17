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

package healthcheck

import (
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/fake"
	"github.com/google/go-cmp/cmp"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

const projectID = "proj-1"

func TestHealthCheckSchema(t *testing.T) {
	key := meta.GlobalKey("key-1")
	x := NewMutableHealthCheck(projectID, key)
	if err := x.CheckSchema(); err != nil {
		t.Fatalf("CheckSchema() = %v, want nil", err)
	}
}

func newDefaultHC() compute.HealthCheck {
	return compute.HealthCheck{
		Name:               "hc-1",
		HealthyThreshold:   10,
		CheckIntervalSec:   7,
		TimeoutSec:         5,
		Type:               "SSL",
		UnhealthyThreshold: 4,
	}
}
func newDefaultAlphaHC() alpha.HealthCheck {
	return alpha.HealthCheck{
		Name:               "hc-1",
		HealthyThreshold:   10,
		CheckIntervalSec:   7,
		TimeoutSec:         5,
		Type:               "UDP",
		UnhealthyThreshold: 4,
		UdpHealthCheck:     &alpha.UDPHealthCheck{Port: 60},
	}
}
func newDefaultBetaHC() beta.HealthCheck {
	return beta.HealthCheck{
		Name:               "hc-1",
		HealthyThreshold:   10,
		CheckIntervalSec:   7,
		TimeoutSec:         5,
		Type:               "TCP",
		UnhealthyThreshold: 4,
		TcpHealthCheck:     &beta.TCPHealthCheck{Port: 60},
	}
}
func TestHealthCheckBuilder(t *testing.T) {
	id := ID(projectID, meta.GlobalKey("hc-1"))
	hcMutRes := NewMutableHealthCheck(projectID, id.Key)
	hc := newDefaultHC()
	err := hcMutRes.Access(func(x *compute.HealthCheck) {
		*x = hc
	})
	if err != nil {
		t.Fatalf("hcMutRes.Access(_) = %v, want nil", err)
	}
	hcRes, err := hcMutRes.Freeze()
	if err != nil {
		t.Fatalf("hcMutRes.Freeze(_) = %v, want nil", err)
	}
	b := NewBuilderWithResource(hcRes)
	node, err := b.Build()
	if err != nil {
		t.Fatalf("b.Build() = %v, want nil", err)
	}
	res := node.Resource().(HealthCheck)
	ga, err := res.ToGA()
	if err != nil {
		t.Fatalf("hcNode.resource.ToGA() = %v, want nil", err)
	}
	if diff := cmp.Diff(ga, &hc); diff != "" {
		t.Fatalf("cmp.Diff(_, _) = %v, want nil", diff)
	}

	hcNode := node.(*healthCheckNode)
	if outs := hcNode.OutRefs(); len(outs) != 0 {
		t.Fatalf("Health check Out Refs length mismatch: got %d want 0", len(outs))
	}

}

func TestHealthCheckSetAllRequiredFields(t *testing.T) {
	id := ID(projectID, meta.GlobalKey("hc-1"))
	hcMutRes := NewMutableHealthCheck(projectID, id.Key)
	err := hcMutRes.Access(func(x *compute.HealthCheck) {
		x.Name = "hc-1"
	})
	// Check that Access will return error when required fields are not set
	if err == nil {
		t.Fatal("hcMutRes.Access(_) = nil, want error")
	}
	err = hcMutRes.Access(func(x *compute.HealthCheck) {
		x.Name = "hc-1"
		x.HealthyThreshold = 10
		x.CheckIntervalSec = 7
		x.TimeoutSec = 5
		x.Type = "SSL"
		x.UnhealthyThreshold = 4
	})
	if err != nil {
		t.Fatalf("hcMutRes.Access(_) = %v, want nil", err)
	}
	hcRes, err := hcMutRes.Freeze()
	if err != nil {
		t.Fatalf("hcMutRes.Freeze(_) = %v, want nil", err)
	}
	b := NewBuilder(id)
	b.SetOwnership(rnode.OwnershipManaged)
	b.SetResource(hcRes)
	b.SetState(rnode.NodeExists)
	node, err := b.Build()
	if err != nil {
		t.Fatalf("b.Build() = %v, want nil", err)
	}
	hcNode := node.(*healthCheckNode)
	ga, err := hcNode.resource.ToGA()
	if err != nil {
		t.Fatalf("hcNode.resource.ToGA() = %v, want nil", err)
	}
	hc := newDefaultHC()
	if diff := cmp.Diff(ga, &hc); diff != "" {
		t.Fatalf("cmp.Diff(_, _) = %v, want nil", diff)
	}
}

func TestHealthCheckAlphaFields(t *testing.T) {
	id := ID(projectID, meta.GlobalKey("hc-1"))
	hcMutRes := NewMutableHealthCheck(projectID, id.Key)
	err := hcMutRes.Access(func(x *compute.HealthCheck) {
		x.Name = "hc-1"
		x.HealthyThreshold = 10
		x.CheckIntervalSec = 7
		x.TimeoutSec = 5
		x.Type = "UDP"
		x.UnhealthyThreshold = 4
	})
	if err != nil {
		t.Fatalf("hcMutRes.Access(_) = %v, want nil", err)
	}
	// Check that Access will return error when OutputOnly fields are set
	err = hcMutRes.AccessAlpha(func(x *alpha.HealthCheck) {
		x.SelfLinkWithId = "hc-1"
	})
	if err == nil {
		t.Fatalf("hcMutRes.Access(_) = %v, want err", err)
	}
	// Set Alpha specific fields
	err = hcMutRes.AccessAlpha(func(x *alpha.HealthCheck) {
		x.SelfLinkWithId = ""
		x.UdpHealthCheck = &alpha.UDPHealthCheck{Port: 60}
	})
	if err != nil {
		t.Fatalf("hcMutRes.Access(_) = %v, want nil", err)
	}

	hcRes, err := hcMutRes.Freeze()
	if err != nil {
		t.Fatalf("hcMutRes.Freeze(_) = %v, want nil", err)
	}
	b := NewBuilder(id)
	b.SetOwnership(rnode.OwnershipManaged)
	b.SetResource(hcRes)
	b.SetState(rnode.NodeExists)
	node, err := b.Build()
	if err != nil {
		t.Fatalf("b.Build() = %v, want nil", err)
	}
	hcNode := node.(*healthCheckNode)
	_, err = hcNode.resource.ToGA()
	if err == nil {
		t.Fatalf("hcNode.resource.ToGA() = %v, want error", err)
	}
	alpha, err := hcNode.resource.ToAlpha()
	if err != nil {
		t.Fatalf("hcNode.resource.ToAlpha() = %v, want nil", err)
	}
	hc := newDefaultAlphaHC()
	if diff := cmp.Diff(alpha, &hc); diff != "" {
		t.Fatalf("cmp.Diff(_, _) = %v, want nil", diff)
	}
}
func TestHealthCheckBeta(t *testing.T) {
	id := ID(projectID, meta.GlobalKey("hc-1"))
	hcMutRes := NewMutableHealthCheck(projectID, id.Key)
	err := hcMutRes.AccessBeta(func(x *beta.HealthCheck) {
		x.Name = "hc-1"
		x.HealthyThreshold = 10
		x.CheckIntervalSec = 7
		x.TimeoutSec = 5
		x.Type = "TCP"
		x.UnhealthyThreshold = 4
		x.TcpHealthCheck = &beta.TCPHealthCheck{Port: 80}
	})
	if err != nil {
		t.Fatalf("hcMutRes.AccessBeta(_) = %v, want nil", err)
	}

	hcRes, err := hcMutRes.Freeze()
	if err != nil {
		t.Fatalf("hcMutRes.Freeze(_) = %v, want nil", err)
	}
	b := NewBuilder(id)
	b.SetOwnership(rnode.OwnershipManaged)
	b.SetResource(hcRes)
	b.SetState(rnode.NodeExists)
	node, err := b.Build()
	if err != nil {
		t.Fatalf("b.Build() = %v, want nil", err)
	}
	hcNode := node.(*healthCheckNode)
	beta, err := hcNode.resource.ToBeta()
	if err != nil {
		t.Fatalf("hcNode.resource.ToBeta() = %v, want nil", err)
	}
	hc := newDefaultBetaHC()
	if diff := cmp.Diff(beta, &hc); diff == "" {
		t.Fatal("cmp.Diff(_, _) = nil, want diff")
	}
}

func buildHCNode(t *testing.T, name string, hc compute.HealthCheck) rnode.Node {
	id := ID(projectID, meta.GlobalKey(name))
	hcMutRes := NewMutableHealthCheck(projectID, id.Key)
	err := hcMutRes.Access(func(x *compute.HealthCheck) {
		*x = hc
	})
	if err != nil {
		t.Fatalf("hcMutRes.Access(_) = %v, want nil", err)
	}
	hcRes, err := hcMutRes.Freeze()
	if err != nil {
		t.Fatalf("hcMutRes.Freeze(_) = %v, want nil", err)
	}
	b := NewBuilderWithResource(hcRes)
	b.SetState(rnode.NodeExists)
	node, err := b.Build()
	if err != nil {
		t.Fatalf("b.Build() = %v, want nil", err)
	}
	return node
}

func TestHealthCheckDiff(t *testing.T) {
	hc := newDefaultHC()
	wantNode := buildHCNode(t, "hc-1", hc)

	// compare identical nodes, no diff
	plan, err := wantNode.Diff(wantNode)
	if err != nil || plan.Diff != nil {
		t.Fatalf("wantNode.Diff(wantNode) = (%v, %v), want (diff, nil)", plan.Diff, err)
	}
	if plan.Operation != rnode.OpNothing {
		t.Fatalf("plan.Operation mismatch, got: %s, want %s", plan.Operation, rnode.OpNothing)
	}

	// modify health check
	hc.CheckIntervalSec = 100
	gotNode := buildHCNode(t, "hc-1", hc)

	plan, err = wantNode.Diff(gotNode)
	if err != nil || plan.Diff == nil {
		t.Fatalf("wantNode.Diff(gotNode) = (%v, %v), want (diff, nil)", plan.Diff, err)
	}
	if plan.Operation != rnode.OpUpdate {
		t.Fatalf("plan.Operation mismatch, got: %s, want %s", plan.Operation, rnode.OpUpdate)
	}

	//compare alpha and ga node
	id := ID(projectID, meta.GlobalKey("hc-1"))
	hcMutRes := NewMutableHealthCheck(projectID, id.Key)
	err = hcMutRes.Access(func(x *compute.HealthCheck) {
		*x = hc
	})
	if err != nil {
		t.Fatalf("hcMutRes.Access(_) = %v, want nil", err)
	}
	err = hcMutRes.AccessAlpha(func(x *alpha.HealthCheck) {
		x.UdpHealthCheck = &alpha.UDPHealthCheck{Port: 60}
	})
	if err != nil {
		t.Fatalf("hcMutRes.AccessAlpha(_) = %v, want nil", err)
	}
	hcRes, err := hcMutRes.Freeze()
	if err != nil {
		t.Fatalf("hcMutRes.Freeze(_) = %v, want nil", err)
	}
	b := NewBuilderWithResource(hcRes)
	alphaNode, err := b.Build()
	if err != nil {
		t.Fatalf("b.Build() = %v, want nil", err)
	}
	plan, err = wantNode.Diff(alphaNode)
	if err != nil || plan.Diff == nil {
		t.Fatalf("wantNode.Diff(wantNode) = (%v, %v), want (plan, nil)", plan, err)
	}
	if plan.Operation != rnode.OpUpdate {
		t.Fatalf("plan.Operation mismatch, got: %s, want %s", plan.Operation, rnode.OpUpdate)
	}

	// compare with fake Node
	fakeId := ID(projectID, meta.GlobalKey("fake-resource"))
	fakeBuilder := fake.NewBuilder(fakeId)
	fakeRes := fake.NewMutableFake(projectID, fakeId.Key)
	res, err := fakeRes.Freeze()
	fakeBuilder.SetResource(res)
	fakeNode, err := fakeBuilder.Build()

	_, err = wantNode.Diff(fakeNode)
	if err == nil {
		t.Fatal("wantNode.Diff(fakeNode) = nil, want error")
	}

}

func TestAction(t *testing.T) {
	hc := newDefaultHC()
	n1 := buildHCNode(t, "hc-1", hc)
	n2 := buildHCNode(t, "hc-1", hc)

	for _, tc := range []struct {
		desc    string
		op      rnode.Operation
		wantErr bool
		want    []exec.ActionType
	}{
		{
			desc: "create action",
			op:   rnode.OpCreate,
			want: []exec.ActionType{exec.ActionTypeCreate},
		},
		{
			desc: "delete action",
			op:   rnode.OpDelete,
			want: []exec.ActionType{exec.ActionTypeDelete},
		},
		{
			desc: "recreate action",
			op:   rnode.OpRecreate,
			want: []exec.ActionType{exec.ActionTypeDelete, exec.ActionTypeCreate},
		},
		{
			desc: "no action",
			op:   rnode.OpNothing,
			want: []exec.ActionType{exec.ActionTypeMeta},
		},
		{
			desc: "update action",
			op:   rnode.OpUpdate,
			want: []exec.ActionType{exec.ActionTypeUpdate},
		},
		{
			desc:    "default",
			op:      rnode.OpUnknown,
			wantErr: true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {

			n1.Plan().Set(rnode.PlanDetails{
				Operation: tc.op,
				Why:       "test plan",
			})
			actions, err := n1.Actions(n2)
			isError := (err != nil)
			if tc.wantErr != isError {
				t.Fatalf("n.Actions(_) =%v got error %v, want %v", err, tc.wantErr, isError)
			}
			if tc.wantErr {
				return
			}
			if err != nil {
				t.Fatalf("n.Actions(_) = %v, want nil", err)
			}
			if len(actions) != len(tc.want) {
				t.Fatalf("n.Actions(%q) returned list with elements %d want %d", tc.op, len(actions), len(tc.want))
			}
			for i, a := range actions {
				if a.Metadata().Type != tc.want[i] {
					t.Errorf("Actions mismatch: got: %s, want: %s", a.Metadata().Name, tc.want[i])
				}
			}
		})
	}
}
