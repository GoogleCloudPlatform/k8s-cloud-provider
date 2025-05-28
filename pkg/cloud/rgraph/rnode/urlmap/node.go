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

package urlmap

import (
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

type urlMapNode struct {
	rnode.NodeBase
	resource Resource
}

var _ rnode.Node = (*urlMapNode)(nil)

func (n *urlMapNode) Resource() rnode.UntypedResource { return n.resource }

func (n *urlMapNode) Diff(gotNode rnode.Node) (*rnode.PlanDetails, error) {
	got, ok := gotNode.(*urlMapNode)
	if !ok {
		return nil, fmt.Errorf("UrlMapNode: invalid type to Diff: %T", gotNode)
	}

	diff, err := got.resource.Diff(n.resource)
	if err != nil {
		return nil, fmt.Errorf("UrlMapNode: Diff %w", err)
	}

	if diff.HasDiff() {
		// TODO: handle set labels with an update operation.
		return &rnode.PlanDetails{
			Operation: rnode.OpRecreate,
			Why:       "UrlMap needs to be recreated (no update method exists)",
			Diff:      diff,
		}, nil
	}

	return &rnode.PlanDetails{
		Operation: rnode.OpNothing,
		Why:       "No diff between got and want",
	}, nil
}

func (n *urlMapNode) Actions(got rnode.Node) ([]exec.Action, error) {
	op := n.Plan().Op()

	switch op {
	case rnode.OpCreate:
		return rnode.CreateActions[compute.UrlMap, alpha.UrlMap, beta.UrlMap](&urlMapOps{}, n, n.resource)

	case rnode.OpDelete:
		return rnode.DeleteActions[compute.UrlMap, alpha.UrlMap, beta.UrlMap](&urlMapOps{}, got, n)

	case rnode.OpNothing:
		return []exec.Action{exec.NewExistsAction(n.ID())}, nil

	case rnode.OpRecreate:
		return rnode.RecreateActions[compute.UrlMap, alpha.UrlMap, beta.UrlMap](&urlMapOps{}, got, n, n.resource)

	case rnode.OpUpdate:
		// TODO
	}

	return nil, fmt.Errorf("UrlMapNode: invalid plan op %s", op)
}

func (n *urlMapNode) Builder() rnode.Builder {
	b := &builder{}
	b.Init(n.ID(), n.State(), n.Ownership(), n.resource)
	return b
}
