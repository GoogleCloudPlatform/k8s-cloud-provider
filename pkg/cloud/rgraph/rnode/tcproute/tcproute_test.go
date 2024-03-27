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

package tcproute

import (
	"context"
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"google.golang.org/api/networkservices/v1"
)

const projectID = "proj-1"

func TestTcpRouteSchema(t *testing.T) {
	key := meta.GlobalKey("key-1")
	x := NewMutableTcpRoute(projectID, key)
	if err := x.CheckSchema(); err != nil {
		t.Fatalf("CheckSchema() = %v, want nil", err)
	}
}

func TestTCPRouteBuilder(t *testing.T) {
	id := ID(projectID, meta.GlobalKey("tcproute-1"))
	b := NewBuilder(id)
	tcpMutResource := defaultTCPRouteResource(t, id)
	match := &networkservices.TcpRouteRouteMatch{
		Address: "adr",
		Port:    "80",
	}
	err := tcpMutResource.Access(func(x *networkservices.TcpRoute) {
		x.Rules[0].Matches = []*networkservices.TcpRouteRouteMatch{match}
	})

	if err != nil {
		t.Fatalf("tcpMutResource.Access(_) = %v, want nil", err)
	}

	tcpResource, err := tcpMutResource.Freeze()
	if err != nil {
		t.Fatalf(" tcpMutResource.Freeze() = %v, want nil", err)
	}
	if err := b.SetResource(tcpResource); err != nil {
		t.Fatalf("SetResource(_) = %v, want nil", err)
	}
	n, err := b.Build()
	if err != nil || n.ID() == nil {
		t.Fatalf("b.Build() = ( %v, %v), want (node, nil)", n.ID(), err)
	}

	if *n.ID() != *id {
		t.Fatalf("node resourceID mismatch, got: %v, want: %v", *n.ID(), *id)
	}
	validateOutRefs(t, b)
}

func TestBuildTcpRouteWithResource(t *testing.T) {
	id := ID(projectID, meta.GlobalKey("tcproute-1"))
	tcpMutResource := defaultTCPRouteResource(t, id)
	res, err := tcpMutResource.Freeze()
	if err != nil {
		t.Fatalf("tcpMutResource.Freeze() = %v, want nil", err)
	}
	b := NewBuilderWithResource(res)
	validateOutRefs(t, b)
}

func TestNodeDiffResource(t *testing.T) {
	id := ID(projectID, meta.GlobalKey("tcproute-1"))

	n1 := createTcpNode(t, id, rnode.NodeExists)
	mutRes := defaultTCPRouteResource(t, id)
	err := mutRes.Access(func(x *networkservices.TcpRoute) {
		x.Rules[0].Action.Destinations[0].Weight = 50
	})
	if err != nil {
		t.Fatalf("tcp mutable resource update failed, err %v, want nil", err)
	}

	r, err := mutRes.Freeze()
	if err != nil {
		t.Fatalf("mutRes.Freeze() = %v, want nil", err)
	}
	b := n1.Builder()
	b.SetResource(r)
	n2, err := b.Build()
	if err != nil {
		t.Fatalf("rnode.Build() = %v, want nil", err)
	}

	p, err := n1.Diff(n2)
	if err != nil || p == nil {
		t.Fatalf("rnode.Diff(_) = %v, want nil", err)
	}
	if p.Diff == nil {
		t.Fatalf("Diff should not be empty")
	}
	if p.Operation != rnode.OpUpdate {
		t.Fatalf("plan Operation mismatch got: %q, want: %q", p.Operation, rnode.OpRecreate)
	}
}

func TestNodeDiffTheSameResource(t *testing.T) {
	id := ID(projectID, meta.GlobalKey("tcproute-1"))
	n1 := createTcpNode(t, id, rnode.NodeExists)
	n2 := createTcpNode(t, id, rnode.NodeExists)

	// compare the same nodes
	p, err := n2.Diff(n1)
	if err != nil || p == nil {
		t.Fatalf("rnode.Diff(_) = %v, want nil", err)
	}
	if p.Diff != nil {
		t.Fatalf("same node should not have Diff")
	}
	if p.Operation != rnode.OpNothing {
		t.Fatalf("plan Operation mismatch got: %q, want: %q", p.Operation, rnode.OpNothing)
	}
}

func TestAction(t *testing.T) {
	id := ID(projectID, meta.GlobalKey("tcp-n1"))
	n1 := createTcpNode(t, id, rnode.NodeExists)
	n2 := createTcpNode(t, id, rnode.NodeDoesNotExist)

	for _, tc := range []struct {
		desc    string
		op      rnode.Operation
		wantErr bool
		want    int
	}{
		{
			desc: "create action",
			op:   rnode.OpCreate,
			want: 1,
		},
		{
			desc: "delete action",
			op:   rnode.OpCreate,
			want: 1,
		},
		{
			desc: "recreate action",
			op:   rnode.OpRecreate,
			want: 2,
		},
		{
			desc: "no action",
			op:   rnode.OpNothing,
			want: 1,
		},
		{
			desc:    "update action - not implemented",
			op:      rnode.OpUpdate,
			wantErr: true,
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
			a, err := n1.Actions(n2)
			isError := (err != nil)
			if tc.wantErr != isError {
				t.Fatalf("n.Actions(_) got error %v, want %v", tc.wantErr, isError)
			}
			if tc.wantErr {
				return
			}
			if err != nil {
				t.Fatalf("n.Actions(_) = %v, want nil", err)
			}
			if len(a) != tc.want {
				t.Fatalf("n.Actions(%q) returned list with elements %d want %d", tc.op, len(a), tc.want)
			}
		})
	}
}

func TestSyncFromCloud(t *testing.T) {
	ctx := context.Background()
	cl := cloud.NewMockGCE(&cloud.SingleProjectRouter{ID: projectID})

	key := meta.GlobalKey("tcproute-2")
	id := ID(projectID, key)

	b := NewBuilder(id)

	if err := b.SyncFromCloud(ctx, cl); err != nil {
		t.Fatalf("b.SyncFromCloud(_, _) = %v, want nil", err)
	}
	if b.State() != rnode.NodeDoesNotExist {
		t.Fatalf("node state mismatch, got: %v, want %v", b.State(), rnode.NodeDoesNotExist)
	}

	// Add tcproute to the cloud and sync again
	obj := defaultTCPRoute()

	if err := cl.MockTcpRoutes.Insert(ctx, key, obj); err != nil {
		t.Fatalf("Error initializing facke cloud, got: %v, want nil", err)
	}

	if err := b.SyncFromCloud(ctx, cl); err != nil {
		t.Fatalf("b.SyncFromCloud(_, _) = %v, want nil", err)
	}
	if b.State() != rnode.NodeExists {
		t.Fatalf("node state mismatch, got: %v, want %v", b.State(), rnode.NodeExists)
	}
	r := b.Resource()
	got, ok := r.(TcpRoute)
	if !ok {
		t.Fatalf("node's resource has uncastable type: %T", got)
	}
	gaRes, err := got.ToGA()
	if err != nil {
		t.Fatalf("got.ToGA() = %v, want nil", err)
	}
	if !reflect.DeepEqual(*gaRes, *obj) {
		t.Fatalf("Objects are not equal: got: %+v, want: %+v", *gaRes, *obj)
	}
}

func validateOutRefs(t *testing.T, b rnode.Builder) {
	outRefs, err := b.OutRefs()
	if err != nil {
		t.Fatalf("b.OutRefs() = %v, want nil", err)
	}
	if len(outRefs) != 2 {
		t.Errorf("Expected 2 out refs")
	}
	for _, o := range outRefs {
		if o.From == nil {
			t.Errorf("OutRefReference From is nil")
			continue
		}
		if *o.From != *b.ID() {
			t.Errorf("o.From != id got : %v, want: %v", o.From, *b.ID())
		}

		if o.To == nil {
			t.Errorf("OutRefReference To is nil")
			continue
		}
		if o.To.Resource != "backendServices" {
			t.Errorf("o.To.Resource != BackendService: got: %v", o.To.Resource)
		}
	}
}

func defaultTCPRouteResource(t *testing.T, id *cloud.ResourceID) MutableTcpRoute {
	d := &networkservices.TcpRouteRouteDestination{
		ServiceName: "https://networkservices.googleapis.com/v1/projects/proj-1/global/backendServices/bs",
		Weight:      10,
	}
	trrr := &networkservices.TcpRouteRouteRule{
		Action: &networkservices.TcpRouteRouteAction{
			Destinations: []*networkservices.TcpRouteRouteDestination{d},
			IdleTimeout:  "5",
		},
		Matches: []*networkservices.TcpRouteRouteMatch{},
	}
	tcpMutResource := NewMutableTcpRoute(projectID, id.Key)
	err := tcpMutResource.Access(func(x *networkservices.TcpRoute) {
		x.Description = "desc"
		x.Name = id.Key.Name
		x.Meshes = []string{"mesh-1"}
		x.Rules = []*networkservices.TcpRouteRouteRule{trrr, trrr}
	})
	if err != nil {
		t.Errorf("Access(_) = %v, want nil", err)
	}
	return tcpMutResource
}

func defaultTCPRoute() *networkservices.TcpRoute {
	d := &networkservices.TcpRouteRouteDestination{
		ServiceName: "https://networkservices.googleapis.com/v1/projects/proj-1/global/backendServices/bs",
		Weight:      50,
	}
	trrr := &networkservices.TcpRouteRouteRule{
		Action: &networkservices.TcpRouteRouteAction{
			Destinations: []*networkservices.TcpRouteRouteDestination{d},
		},
		Matches: []*networkservices.TcpRouteRouteMatch{},
	}
	return &networkservices.TcpRoute{
		Name:   "tcproute-2",
		Meshes: []string{"mesh-2"},
		Rules:  []*networkservices.TcpRouteRouteRule{trrr},
	}
}

func createTcpNode(t *testing.T, id *cloud.ResourceID, state rnode.NodeState) rnode.Node {
	b := NewBuilder(id)

	tcpResource, err := defaultTCPRouteResource(t, id).Freeze()
	if err != nil {
		t.Fatalf(" tcpMutResource.Freeze() = %v, want nil", err)
	}
	if err := b.SetResource(tcpResource); err != nil {
		t.Fatalf("SetResource(_) = %v, want nil", err)
	}
	b.SetState(state)
	b.SetOwnership(rnode.OwnershipManaged)
	n, err := b.Build()
	if err != nil || n.ID() == nil {
		t.Fatalf("b.Build() = ( %v, %v), want (node, nil)", n.ID(), err)
	}
	return n
}
