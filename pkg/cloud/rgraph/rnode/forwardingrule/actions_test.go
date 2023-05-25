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
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/targethttpproxy"
)

func TestCreateAction(t *testing.T) {
	// TODO
}

func TestUpdateAction(t *testing.T) {
	id := ID("proj", meta.GlobalKey("fr"))
	targetID := targethttpproxy.ID("proj", meta.GlobalKey("tp"))
	oldTargetID := targethttpproxy.ID("proj", meta.GlobalKey("tp2"))

	for _, tc := range []struct {
		name       string
		action     *forwardingRuleUpdateAction
		wantEvents exec.EventList
	}{
		{
			name: "update target",
			action: &forwardingRuleUpdateAction{
				id:        id,
				target:    targetID,
				oldTarget: oldTargetID,
			},
			wantEvents: exec.EventList{
				exec.NewDropRefEvent(id, oldTargetID),
			},
		},
		{
			name: "update label",
			action: &forwardingRuleUpdateAction{
				id:     id,
				labels: map[string]string{"foo": "bar"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mock := cloud.NewMockGCE(&cloud.SingleProjectRouter{ID: "proj"})

			events := tc.action.DryRun()
			if !exec.EventList(events).Equal(tc.wantEvents) {
				t.Errorf("DryRun() = %v, want %v", events, tc.wantEvents)
			}
			events, err := tc.action.Run(context.Background(), mock)
			if err != nil {
				t.Fatalf("Run() = %v, want nil", err)
			}
			if !exec.EventList(events).Equal(tc.wantEvents) {
				t.Errorf("DryRun() = %v, want %v", events, tc.wantEvents)
			}
		})
	}
}
