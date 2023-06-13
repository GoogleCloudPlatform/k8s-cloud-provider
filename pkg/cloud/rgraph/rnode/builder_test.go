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
	"github.com/google/go-cmp/cmp"
)

func TestBuilderBase(t *testing.T) {
	var nb BuilderBase
	id := &cloud.ResourceID{
		Resource: "fake",
		Key:      meta.GlobalKey("res1"),
	}
	nb.Defaults(id)
	type tuple struct {
		ID *cloud.ResourceID
		NS NodeState
		O  OwnershipStatus
		V  meta.Version
	}
	got := tuple{nb.id, nb.state, nb.ownership, nb.version}
	if diff := cmp.Diff(got, tuple{id, NodeUnknown, OwnershipUnknown, meta.VersionGA}); diff != "" {
		t.Errorf("nb; -got,+want: %s", diff)
	}
}
