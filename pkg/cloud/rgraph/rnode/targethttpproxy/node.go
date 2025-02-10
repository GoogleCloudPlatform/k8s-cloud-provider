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

package targethttpproxy

import (
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

type targetHttpProxyNode struct {
	rnode.NodeBase
	resource Resource
}

var _ rnode.Node = (*targetHttpProxyNode)(nil)

func (n *targetHttpProxyNode) Resource() rnode.UntypedResource { return n.resource }

func (n *targetHttpProxyNode) Diff(gotNode rnode.Node) (*rnode.PlanDetails, error) {
	got, ok := gotNode.(*targetHttpProxyNode)
	if !ok {
		return nil, fmt.Errorf("TargetHttpProxyNode: invalid type to Diff: %T", gotNode)
	}

	diff, err := got.resource.Diff(n.resource)
	if err != nil {
		return nil, fmt.Errorf("TargetHttpProxyNode: Diff %w", err)
	}

	if diff.HasDiff() {
		// TODO: handle set labels with an update operation.
		return &rnode.PlanDetails{
			Operation: rnode.OpRecreate,
			Why:       "TargetHttpProxy needs to be recreated (no update method exists)",
			Diff:      diff,
		}, nil
	}

	return &rnode.PlanDetails{
		Operation: rnode.OpNothing,
		Why:       "No diff between got and want",
	}, nil
}

func (n *targetHttpProxyNode) Actions(got rnode.Node) ([]exec.Action, error) {
	op := n.Plan().Op()

	switch op {
	case rnode.OpCreate:
		return rnode.CreateActions[compute.TargetHttpProxy, alpha.TargetHttpProxy, beta.TargetHttpProxy](&targetHttpProxyOps{}, n, n.resource)

	case rnode.OpDelete:
		return rnode.DeleteActions[compute.TargetHttpProxy, alpha.TargetHttpProxy, beta.TargetHttpProxy](&targetHttpProxyOps{}, got, n)

	case rnode.OpNothing:
		return []exec.Action{exec.NewExistsAction(n.ID())}, nil

	case rnode.OpRecreate:
		return rnode.RecreateActions[compute.TargetHttpProxy, alpha.TargetHttpProxy, beta.TargetHttpProxy](&targetHttpProxyOps{}, got, n, n.resource)

	case rnode.OpUpdate:
		// TODO
	}

	return nil, fmt.Errorf("TargetHttpProxyNode: invalid plan op %s", op)
}

func (n *targetHttpProxyNode) Builder() rnode.Builder {
	b := &builder{}
	b.Init(n.ID(), n.State(), n.Ownership(), n.resource)
	return b
}
