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

package exec

import (
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/google/go-cmp/cmp"
)

func diffEvents(a, b []Event) string {
	am := map[string]struct{}{}
	bm := map[string]struct{}{}

	for _, e := range a {
		am[e.String()] = struct{}{}
	}
	for _, e := range b {
		bm[e.String()] = struct{}{}
	}
	return cmp.Diff(am, bm)
}

func TestEventEqual(t *testing.T) {
	id1 := &cloud.ResourceID{
		ProjectID: "proj1",
		Resource:  "res1",
		Key:       meta.GlobalKey("x"),
	}
	id2 := &cloud.ResourceID{
		ProjectID: "proj1",
		Resource:  "res1",
		Key:       meta.GlobalKey("y"),
	}

	events := []Event{
		NewDropRefEvent(id1, id2),
		NewDropRefEvent(id2, id1),
		NewExistsEvent(id1),
		NewExistsEvent(id2),
		NewNotExistsEvent(id1),
		NewNotExistsEvent(id2),
		StringEvent("a"),
		StringEvent("b"),
	}
	for i := 0; i < len(events); i++ {
		for j := 0; j < len(events); j++ {
			if i == j {
				if !events[i].Equal(events[j]) {
					t.Errorf("%v != %v, want ==", events[i], events[j])
				}
			} else {
				if events[i].Equal(events[j]) {
					t.Errorf("%v == %v, want !=", events[i], events[j])
				}
			}
		}
	}
}
