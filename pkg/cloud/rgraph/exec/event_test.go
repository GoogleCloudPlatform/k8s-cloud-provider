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

func TestEventListEqual(t *testing.T) {
	for _, tc := range []struct {
		name string
		a, b EventList
		want bool
	}{
		{
			name: "empty",
			want: true,
		},
		{
			name: "empty with non-empty",
			a:    EventList{StringEvent("a")},
		},
		{
			name: "one element ==",
			a:    EventList{StringEvent("a")},
			b:    EventList{StringEvent("a")},
			want: true,
		},
		{
			name: "one element !=",
			a:    EventList{StringEvent("a")},
			b:    EventList{StringEvent("b")},
		},
		{
			name: "multiple elements, ==",
			a:    EventList{StringEvent("a"), StringEvent("b")},
			b:    EventList{StringEvent("a"), StringEvent("b")},
			want: true,
		},
		{
			name: "multiple elements, !=",
			a:    EventList{StringEvent("a"), StringEvent("b")},
			b:    EventList{StringEvent("a"), StringEvent("c")},
		},
		{
			name: "multiple elements, reordering ==",
			a:    EventList{StringEvent("a"), StringEvent("b")},
			b:    EventList{StringEvent("b"), StringEvent("a")},
			want: true,
		},
		{
			name: "multiple elements, reordering !=",
			a:    EventList{StringEvent("a"), StringEvent("b")},
			b:    EventList{StringEvent("b"), StringEvent("c")},
		},
		{
			name: "multiple different count elements, !=",
			a:    EventList{StringEvent("a"), StringEvent("b")},
			b:    EventList{StringEvent("a"), StringEvent("b"), StringEvent("c")},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotAB := tc.a.Equal(tc.b)
			if gotAB != tc.want {
				t.Errorf("a.Equal(b) = %t, want %t (a = %v, b = %v)", gotAB, tc.want, tc.a, tc.b)
			}
			gotBA := tc.b.Equal(tc.a)
			if gotAB != tc.want {
				t.Errorf("b.Equal(a) = %t, want %t (a = %v, b = %v)", gotBA, tc.want, tc.a, tc.b)
			}
		})
	}
}

func diffEvents(a, b EventList) string {
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

	events := EventList{
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
