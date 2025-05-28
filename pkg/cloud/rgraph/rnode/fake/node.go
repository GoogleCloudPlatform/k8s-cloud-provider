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

package fake

import (
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
)

type fakeNode struct {
	rnode.NodeBase
	resource Resource
}

var _ rnode.Node = (*fakeNode)(nil)

func (n *fakeNode) Resource() rnode.UntypedResource { return n.resource }

func (n *fakeNode) Diff(gotNode rnode.Node) (*rnode.PlanDetails, error) {
	gotRes, ok := gotNode.Resource().(Resource)
	if !ok {
		return nil, fmt.Errorf("fakeNode %s: invalid type to Diff: %T", n.ID(), gotNode.Resource())
	}

	diff, err := gotRes.Diff(n.resource)
	if err != nil {
		return nil, fmt.Errorf("fakeNode %s: Diff %w", n.ID(), err)
	}
	if diff.HasDiff() {
		return &rnode.PlanDetails{
			Operation: rnode.OpUpdate,
			Why:       "Fake has diff",
			Diff:      diff,
		}, nil
	}

	return &rnode.PlanDetails{
		Operation: rnode.OpNothing,
		Why:       "No diff between got and want",
	}, nil
}

func (n *fakeNode) Actions(got rnode.Node) ([]exec.Action, error) {
	op := n.Plan().Op()

	switch op {
	case rnode.OpCreate:
		return []exec.Action{exec.NewExistsAction(n.ID())}, nil
	case rnode.OpDelete:
		return []exec.Action{exec.NewDoesNotExistAction(n.ID())}, nil
	case rnode.OpNothing:
		return []exec.Action{exec.NewExistsAction(n.ID())}, nil
	case rnode.OpRecreate:
		return []exec.Action{exec.NewExistsAction(n.ID())}, nil
	case rnode.OpUpdate:
		return []exec.Action{exec.NewExistsAction(n.ID())}, nil
	}

	return nil, fmt.Errorf("fakeNode %s: invalid plan op %s", n.ID(), op)
}

func (n *fakeNode) Builder() rnode.Builder {
	b := &Builder{}
	b.Init(n.ID(), n.State(), n.Ownership(), nil)
	return b
}
