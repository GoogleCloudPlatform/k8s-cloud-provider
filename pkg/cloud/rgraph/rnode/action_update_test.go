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

package rnode

import (
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
)

const project = "proj-id"

func globalID(name string) *cloud.ResourceID {
	key := meta.GlobalKey(name)
	return &cloud.ResourceID{
		Resource:  "node",
		APIGroup:  "",
		ProjectID: project,
		Key:       key,
	}
}

func createFakeNode(toRefs []string) Node {
	fn := fakeNode{}
	fn.ownership = OwnershipManaged
	fn.state = NodeExists
	fn.id = globalID("fn")
	for _, to := range toRefs {
		outRef := ResourceRef{
			From: fn.id,
			To:   globalID(to),
		}
		fn.outRefs = append(fn.outRefs, outRef)
	}
	return &fn
}

func createNotExistingFakeNode() Node {
	fn := fakeNode{}
	fn.ownership = OwnershipManaged
	fn.state = NodeDoesNotExist
	fn.id = globalID("fn")
	return &fn
}

func updateEventList(toRefs ...string) exec.EventList {
	var events exec.EventList
	from := globalID("fn")
	for _, to := range toRefs {
		events = append(events, exec.NewDropRefEvent(from, globalID(to)))
	}
	events = append(events, exec.NewExistsEvent(from))
	return events
}

func newExistsEventList(toRefs ...string) exec.EventList {
	var events exec.EventList
	for _, to := range toRefs {
		events = append(events, exec.NewExistsEvent(globalID(to)))
	}
	return events
}

func TestPostUpdateActions(t *testing.T) {
	for _, tc := range []struct {
		desc       string
		oldNode    Node
		newNode    Node
		wantEvents exec.EventList
		wantErr    bool
	}{
		{
			desc:    "node's without outefs",
			oldNode: createFakeNode(nil),
			newNode: createFakeNode(nil),
		},
		{
			desc:       "node's with added outefs",
			oldNode:    createFakeNode(nil),
			newNode:    createFakeNode([]string{"a", "b"}),
			wantEvents: newExistsEventList("a", "b"),
		},
		{
			desc:    "node's with deleted outefs",
			oldNode: createFakeNode([]string{"a", "b"}),
			newNode: createFakeNode(nil),
		},
		{
			desc:       "node's with replaced outef",
			oldNode:    createFakeNode([]string{"a", "b"}),
			newNode:    createFakeNode([]string{"a", "c"}),
			wantEvents: newExistsEventList("a", "c"),
		},
		{
			desc:    "nodes don't exist",
			oldNode: createNotExistingFakeNode(),
			newNode: createNotExistingFakeNode(),
			wantErr: true,
		},
		{
			desc:    "new node doesn't exist",
			oldNode: createFakeNode([]string{"a", "b"}),
			newNode: createNotExistingFakeNode(),
			wantErr: true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {

			gotEvents, err := updatePreconditions(tc.oldNode, tc.newNode)
			gotErr := err != nil
			if gotErr != tc.wantErr {
				t.Errorf("updatePreconditions(_, _) = %v, want %v", gotErr, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if len(gotEvents) != len(tc.wantEvents) {
				t.Fatalf("event's length mismatch, got: %d, want: %d", len(gotEvents), len(tc.wantEvents))
			}
			for i, gotEvent := range gotEvents {
				wantEvent := tc.wantEvents[i]
				if !gotEvent.Equal(wantEvent) {
					t.Errorf("%v != %v", gotEvent.String(), wantEvent.String())
				}
			}
		})
	}
}

func TestUpdatePreconditions(t *testing.T) {
	for _, tc := range []struct {
		desc       string
		oldNode    Node
		newNode    Node
		wantEvents exec.EventList
	}{
		{
			desc:       "node's without outefs",
			oldNode:    createFakeNode(nil),
			newNode:    createFakeNode(nil),
			wantEvents: updateEventList(),
		},
		{
			desc:       "node's with added outefs",
			oldNode:    createFakeNode(nil),
			newNode:    createFakeNode([]string{"a", "b"}),
			wantEvents: updateEventList(),
		},
		{
			desc:       "node's with deleted outefs",
			oldNode:    createFakeNode([]string{"a", "b"}),
			newNode:    createFakeNode(nil),
			wantEvents: updateEventList("a", "b"),
		},
		{
			desc:       "node's with deleted first outefs",
			oldNode:    createFakeNode([]string{"a", "b"}),
			newNode:    createFakeNode([]string{"b"}),
			wantEvents: updateEventList("a"),
		},
		{
			desc:       "node's with deleted outefs, random order",
			oldNode:    createFakeNode([]string{"a", "b", "c"}),
			newNode:    createFakeNode([]string{"b", "a"}),
			wantEvents: updateEventList("c"),
		},
		{
			desc:       "node's with replaced outefs",
			oldNode:    createFakeNode([]string{"a", "b", "c"}),
			newNode:    createFakeNode([]string{"a", "b", "e"}),
			wantEvents: updateEventList("c"),
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {

			gotEvents := postUpdateActionEvents(tc.oldNode, tc.newNode)
			if len(gotEvents) != len(tc.wantEvents) {
				t.Fatalf("postUpdateActionEvents(got, want) = %d, want %d", len(gotEvents), len(tc.wantEvents))
			}
			for i, gotEvent := range gotEvents {
				wantEvent := tc.wantEvents[i]
				if !gotEvent.Equal(wantEvent) {
					t.Errorf("%v != %v", gotEvent.String(), wantEvent.String())
				}
			}
		})
	}
}
