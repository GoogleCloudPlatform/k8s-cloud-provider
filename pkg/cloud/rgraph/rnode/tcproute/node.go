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

package tcproute

import (
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"google.golang.org/api/networkservices/v1"
	beta "google.golang.org/api/networkservices/v1beta1"
)

type tcpRouteNode struct {
	rnode.NodeBase
	resource Resource
}

var _ rnode.Node = (*tcpRouteNode)(nil)

func (n *tcpRouteNode) Resource() rnode.UntypedResource { return n.resource }

func (n *tcpRouteNode) Diff(gotNode rnode.Node) (*rnode.PlanDetails, error) {
	got, ok := gotNode.(*tcpRouteNode)
	if !ok {
		return nil, fmt.Errorf("TcpRouteNode: invalid type to Diff: %T", gotNode)
	}

	diff, err := got.resource.Diff(n.resource)
	if err != nil {
		return nil, fmt.Errorf("TcpRouteNode: Diff %w", err)
	}

	for i, item := range diff.Items {
		if item.Path.Equal(api.Path{"*", ".Name"}) {
			diff.Items = append(diff.Items[:i], diff.Items[i+1:]...)
			break
		}
	}
	if diff.HasDiff() {
		return &rnode.PlanDetails{
			Operation: rnode.OpUpdate,
			Why:       "TcpRoute needs to be recreated",
			Diff:      diff,
		}, nil
	}

	return &rnode.PlanDetails{
		Operation: rnode.OpNothing,
		Why:       "No diff between got and want",
	}, nil
}

func (n *tcpRouteNode) runOp(got rnode.Node, op rnode.Operation) ([]exec.Action, error) {
	switch op {
	case rnode.OpCreate:
		return rnode.CreateActions[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute](&tcpRouteOps{}, n, n.resource)

	case rnode.OpDelete:
		return rnode.DeleteActions[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute](&tcpRouteOps{}, got, n)

	case rnode.OpNothing:
		return []exec.Action{exec.NewExistsAction(n.ID())}, nil

	case rnode.OpRecreate:
		return rnode.RecreateActions[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute](&tcpRouteOps{}, got, n, n.resource)

	case rnode.OpUpdate:
		// TCP route does not support fingerprint
		return rnode.UpdateActions[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute](&tcpRouteOps{}, got, n, n.resource, "")
	}

	return nil, fmt.Errorf("TcpRouteNode: invalid plan op %s", op)
}

func (n *tcpRouteNode) Actions(got rnode.Node) ([]exec.Action, error) {
	op := n.Plan().Op()
	ret, err := n.runOp(got, op)
	if err != nil {
		return nil, fmt.Errorf("TCP Route err: %w", err)
	}
	return ret, nil
}

func (n *tcpRouteNode) Builder() rnode.Builder {
	b := &builder{}
	b.Init(n.ID(), n.State(), n.Ownership(), n.resource)
	return b
}
