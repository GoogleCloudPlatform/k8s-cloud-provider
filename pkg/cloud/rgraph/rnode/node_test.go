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
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/google/go-cmp/cmp"
)

type fakeBuilder struct{ BuilderBase }

func (*fakeBuilder) Resource() UntypedResource                               { return nil }
func (*fakeBuilder) SetResource(UntypedResource) error                       { return nil }
func (*fakeBuilder) OutRefs() ([]ResourceRef, error)                         { return nil, nil }
func (*fakeBuilder) SyncFromCloud(ctx context.Context, cl cloud.Cloud) error { return nil }

func (b *fakeBuilder) Build() (Node, error) {
	ret := &fakeNode{}
	ret.InitFromBuilder(b)
	return ret, nil
}

type fakeNode struct{ NodeBase }

func (*fakeNode) Resource() UntypedResource           { return nil }
func (*fakeNode) Builder() Builder                    { return &fakeBuilder{} }
func (*fakeNode) Diff(Node) (*PlanDetails, error)     { return nil, nil }
func (*fakeNode) Actions(Node) ([]exec.Action, error) { return nil, nil }

func TestNodeBase(t *testing.T) {
	id := &cloud.ResourceID{Resource: "fake", Key: meta.GlobalKey("res1")}
	nb := fakeBuilder{
		BuilderBase: BuilderBase{
			id:        id,
			state:     NodeExists,
			ownership: OwnershipExternal,
		},
	}
	n, _ := nb.Build()

	got := []any{n.ID(), n.State(), n.Ownership()}
	if diff := cmp.Diff(got, []any{
		id, NodeExists, OwnershipExternal,
	}); diff != "" {
		t.Errorf("Diff() -got+want: = %s", diff)
	}

	t.Log(n)
}
