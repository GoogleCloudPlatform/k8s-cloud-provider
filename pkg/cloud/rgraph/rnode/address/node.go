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

package address

import (
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

type addressNode struct {
	rnode.NodeBase
	resource Resource
}

var _ rnode.Node = (*addressNode)(nil)

func (n *addressNode) Resource() rnode.UntypedResource { return n.resource }

func (n *addressNode) Diff(gotNode rnode.Node) (*rnode.PlanDetails, error) {
	gotRes, ok := gotNode.Resource().(Resource)
	if !ok {
		return nil, fmt.Errorf("AddressNode: invalid type to Diff: %T", gotNode.Resource())
	}

	diff, err := gotRes.Diff(n.resource)
	if err != nil {
		return nil, fmt.Errorf("AddressNode: Diff %w", err)
	}

	if diff.HasDiff() {
		// TODO: setLabels() when the field goes GA.
		return &rnode.PlanDetails{
			Operation: rnode.OpRecreate,
			Why:       "Address needs to be recreated (no update method exists)",
			Diff:      diff,
		}, nil
	}

	return &rnode.PlanDetails{
		Operation: rnode.OpNothing,
		Why:       "No diff between got and want",
	}, nil
}

func (n *addressNode) Actions(got rnode.Node) ([]exec.Action, error) {
	op := n.Plan().Op()

	switch op {
	case rnode.OpCreate:
		// TODO: .Labels can only be updated via the setLabels method. This is
		// currently in Beta and we don't support it.
		return rnode.CreateActions[compute.Address, alpha.Address, beta.Address](&ops{}, n, n.resource)

	case rnode.OpDelete:
		return rnode.DeleteActions[compute.Address, alpha.Address, beta.Address](&ops{}, got, n)

	case rnode.OpNothing:
		return []exec.Action{exec.NewExistsAction(n.ID())}, nil

	case rnode.OpRecreate:
		return rnode.RecreateActions[compute.Address, alpha.Address, beta.Address](&ops{}, got, n, n.resource)

	case rnode.OpUpdate:
		return nil, fmt.Errorf("%s is not supported for Address", op)
	}

	return nil, fmt.Errorf("AddressNode: invalid plan op %s", op)
}

func (n *addressNode) Builder() rnode.Builder {
	b := &builder{}
	b.Init(n.ID(), n.State(), n.Ownership(), n.resource)
	return b
}
