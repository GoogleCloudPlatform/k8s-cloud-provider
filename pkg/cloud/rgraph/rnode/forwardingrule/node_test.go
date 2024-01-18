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

package forwardingrule

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/targethttpproxy"
	"github.com/google/go-cmp/cmp"
	"github.com/kr/pretty"
	"google.golang.org/api/compute/v1"
)

func TestNodeBuilder(t *testing.T) {
	id := ID("proj", meta.GlobalKey("fr"))
	b := NewBuilder(id)
	b.SetOwnership(rnode.OwnershipExternal)
	b.SetState(rnode.NodeDoesNotExist)
	n, err := b.Build()
	if err != nil {
		t.Fatalf("Build() = %v", err)
	}
	b2 := n.Builder()
	type result struct {
		O rnode.OwnershipStatus
		S rnode.NodeState
	}
	if diff := cmp.Diff(
		result{O: b2.Ownership(), S: b2.State()},
		result{O: rnode.OwnershipExternal, S: rnode.NodeDoesNotExist},
	); diff != "" {
		t.Fatalf("Diff() -got,+want: %s", diff)
	}
}

func TestCreateActions(t *testing.T) {
	id := ID("proj", meta.GlobalKey("fr"))
	addrID := targethttpproxy.ID("proj", meta.GlobalKey("addr"))
	targetID := targethttpproxy.ID("proj", meta.GlobalKey("tp"))
	mr := NewMutableForwardingRule(id.ProjectID, id.Key)
	mr.Access(func(x *compute.ForwardingRule) {
		x.Name = "fr"
		x.IPAddress = addrID.SelfLink(meta.VersionGA)
		x.Target = targetID.SelfLink(meta.VersionGA)
	})
	r, _ := mr.Freeze()
	b := NewBuilderWithResource(r)
	b.SetState(rnode.NodeExists)
	b.SetOwnership(rnode.OwnershipManaged)
	n, _ := b.Build()
	n.Plan().Set(rnode.PlanDetails{Operation: rnode.OpCreate})

	b = NewBuilder(id)
	b.SetState(rnode.NodeDoesNotExist)
	b.SetOwnership(rnode.OwnershipManaged)
	g, _ := b.Build()

	actions, err := n.Actions(g)
	if err != nil {
		t.Fatal(err)
	}
	var strActions []string
	for _, act := range actions {
		strActions = append(strActions, fmt.Sprint(act))
	}
	if diff := cmp.Diff(strActions, []string{
		"ForwardingRuleCreateAction(compute/forwardingRules:proj/fr)",
	}); diff != "" {
		t.Errorf("Diff(actions) -got,+want: %s", diff)
	}
}

func TestDiffAndActions(t *testing.T) {
	id := ID("proj", meta.GlobalKey("fr"))
	targetID := targethttpproxy.ID("proj", meta.GlobalKey("tp"))
	targetID2 := targethttpproxy.ID("proj", meta.GlobalKey("tp2"))

	const (
		ignoreAccessErr = 1 << iota
	)

	makeFR := func(f func(x *compute.ForwardingRule), flags int) ForwardingRule {
		t.Helper()

		fr := NewMutableForwardingRule(id.ProjectID, id.Key)
		fr.Access(func(x *compute.ForwardingRule) {
			x.Name = "fr"
		})
		if f != nil {
			err := fr.Access(f)
			if err != nil && flags&ignoreAccessErr == 0 {
				t.Fatalf("Access() = %v, want nil", err)
			}
		}
		r, err := fr.Freeze()
		if err != nil {
			t.Fatalf("fr.Freeze() = %v, want nil", err)
		}
		return r
	}

	baseFields := func(x *compute.ForwardingRule) {
		x.IPAddress = "1.2.3.4"
		x.IPProtocol = "TCP"
		x.LoadBalancingScheme = "INTERNAL_MANAGED"
		x.Ports = []string{"80"}
		x.Target = targetID.SelfLink(meta.VersionGA)
		x.ForceSendFields = []string{
			"AllPorts",
			"AllowGlobalAccess",
			"AllowPscGlobalAccess",
			"AllPorts",
			"BackendService",
			"Description",
			"IpVersion",
			"IsMirroringCollector",
			"MetadataFilters",
			"Network",
			"NetworkTier",
			"NoAutomateDnsZone",
			"PortRange",
			"ServiceDirectoryRegistrations",
			"ServiceLabel",
			"SourceIpRanges",
			"Subnetwork",
		}
	}

	for _, tc := range []struct {
		name string
		frw  ForwardingRule
		frg  ForwardingRule

		wantDiff       bool
		wantOp         rnode.Operation
		wantErr        bool
		wantActionsErr bool
		wantActions    []string
	}{
		{
			name: "no diff",
			frw: makeFR(func(x *compute.ForwardingRule) {
				baseFields(x)
				x.NullFields = []string{"Labels"}
			}, 0),
			frg: makeFR(func(x *compute.ForwardingRule) {
				baseFields(x)
			}, ignoreAccessErr),
			wantOp: rnode.OpNothing,
			wantActions: []string{
				"EventAction([Exists(compute/forwardingRules:proj/fr)])",
			},
		},
		{
			name: "update .Target",
			frw: makeFR(func(x *compute.ForwardingRule) {
				baseFields(x)
				x.NullFields = []string{"Labels"}
			}, 0),
			frg: makeFR(func(x *compute.ForwardingRule) {
				baseFields(x)
				x.Target = targetID2.SelfLink(meta.VersionGA)
			}, ignoreAccessErr),
			wantDiff: true,
			wantOp:   rnode.OpUpdate,
			wantActions: []string{
				"EventAction([Exists(compute/forwardingRules:proj/fr)])",
				"ForwardingRuleUpdateAction(compute/forwardingRules:proj/fr)",
			},
		},
		{
			name: "update .Labels",
			frw: makeFR(func(x *compute.ForwardingRule) {
				baseFields(x)
				x.Labels = map[string]string{"foo": "bar"}
			}, 0),
			frg: makeFR(func(x *compute.ForwardingRule) {
				baseFields(x)
				x.Labels = map[string]string{"foo": "bar2"}
			}, ignoreAccessErr),
			wantDiff: true,
			wantOp:   rnode.OpUpdate,
			wantActions: []string{
				"EventAction([Exists(compute/forwardingRules:proj/fr)])",
				"ForwardingRuleUpdateAction(compute/forwardingRules:proj/fr)",
			},
		},
		{
			name: "other changes override target, labels changes",
			frw: makeFR(func(x *compute.ForwardingRule) {
				baseFields(x)
				x.Labels = map[string]string{"foo": "bar"}
			}, 0),
			frg: makeFR(func(x *compute.ForwardingRule) {
				baseFields(x)
				x.Labels = map[string]string{"foo": "bar2"}
				x.Ports = []string{"443"} // Forces recreate.
			}, ignoreAccessErr),
			wantDiff: true,
			wantOp:   rnode.OpRecreate,
			wantActions: []string{
				"GenericDeleteAction(compute/forwardingRules:proj/fr)",
				"GenericCreateAction(compute/forwardingRules:proj/fr)",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bg := NewBuilderWithResource(tc.frg)
			bw := NewBuilderWithResource(tc.frw)

			ng, err := bg.Build()
			if err != nil {
				t.Fatalf("bg.Build() = %v, want nil", err)
			}
			nw, err := bw.Build()
			if err != nil {
				t.Fatalf("bw.Build() = %v, want nil", err)
			}

			pd, err := ng.Diff(nw)
			t.Logf("Diff() = %v; %s", err, pretty.Sprint(pd))
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("")
			}
			if gotDiff := pd.Diff != nil && pd.Diff.HasDiff(); gotDiff != tc.wantDiff {
				t.Errorf("gotDiff = %t, want %t", gotDiff, tc.wantDiff)
			}
			if gotOp := pd.Operation; gotOp != tc.wantOp {
				t.Errorf("gotOp = %s, want %s", gotOp, tc.wantOp)
			}
			// Set the plan to be the same as given by the diff.
			nw.Plan().Set(rnode.PlanDetails{
				Operation: pd.Operation,
				Diff:      pd.Diff,
			})
			actions, err := nw.Actions(ng)
			if gotActionsErr := err != nil; gotActionsErr != tc.wantActionsErr {
				t.Fatalf("Actions() = %v, want nil", err)
			}
			var strActions []string
			for _, act := range actions {
				strActions = append(strActions, fmt.Sprint(act))
			}
			if diff := cmp.Diff(strActions, tc.wantActions); diff != "" {
				t.Errorf("Diff(actions) -got,+want: %s", diff)
			}
		})
	}
}
