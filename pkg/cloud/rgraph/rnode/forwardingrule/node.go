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

package forwardingrule

import (
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

func nodeErr(s string, args ...any) error { return fmt.Errorf("forwardingRule: "+s, args...) }

type forwardingRuleNode struct {
	rnode.NodeBase
	resource Resource
}

var _ rnode.Node = (*forwardingRuleNode)(nil)

func (n *forwardingRuleNode) Resource() rnode.UntypedResource { return n.resource }

// changedFields is a helper that interprets the set of fields that have been changed in a Diff.
type changedFields struct {
	target bool
	labels bool
	other  bool

	// messages are human-readable descriptions of the changed fields.
	messages []string
}

// process an item from the diff. returns true if the item can be handled
// without recreating the resource.
func (c *changedFields) process(item api.DiffItem) bool {
	var messages []string

	switch {
	case api.Path{}.Pointer().Field("Target").Equal(item.Path):
		c.messages = append(messages, fmt.Sprintf("Target (%q -> %q)", item.A, item.B))
		c.target = true
		return true
	case item.Path.HasPrefix(api.Path{}.Pointer().Field("Labels")):
		c.messages = append(messages, fmt.Sprintf("Labels (%v -> %v)", item.A, item.B))
		c.labels = true
		return true
	default:
		c.messages = append(messages, fmt.Sprintf("%s (%v -> %v)", item.Path, item.A, item.B))
		c.other = true
	}

	return false
}

func (n *forwardingRuleNode) Diff(gotNode rnode.Node) (*rnode.PlanDetails, error) {
	got, ok := gotNode.(*forwardingRuleNode)
	if !ok {
		return nil, nodeErr("invalid type to Diff: %T", gotNode)
	}

	diff, err := got.resource.Diff(n.resource)
	if err != nil {
		return nil, nodeErr("Diff: %w", err)
	}

	if diff.HasDiff() {
		var changed changedFields
		for _, item := range diff.Items {
			changed.process(item)
		}

		if !changed.other {
			return &rnode.PlanDetails{
				Operation: rnode.OpUpdate,
				Why:       fmt.Sprintf("update in place (changed=%+v)", changed),
				Diff:      diff,
			}, nil
		}

		return &rnode.PlanDetails{
			Operation: rnode.OpRecreate,
			Why:       "needs to be recreated",
			Diff:      diff,
		}, nil
	}

	return &rnode.PlanDetails{
		Operation: rnode.OpNothing,
		Why:       "No diff between got and want",
	}, nil
}

func (n *forwardingRuleNode) Actions(got rnode.Node) ([]exec.Action, error) {
	op := n.Plan().Op()

	switch op {
	case rnode.OpCreate:
		return n.createActions()

	case rnode.OpDelete:
		return rnode.DeleteActions[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule](&ops{}, got, n)

	case rnode.OpNothing:
		return []exec.Action{exec.NewExistsAction(n.ID())}, nil

	case rnode.OpRecreate:
		return rnode.RecreateActions[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule](&ops{}, got, n, n.resource)

	case rnode.OpUpdate:
		return n.updateActions(got)
	}
	return nil, nodeErr("invalid plan op %s", op)
}

func (n *forwardingRuleNode) Builder() rnode.Builder {
	b := &builder{}
	b.Init(n.ID(), n.State(), n.Ownership(), n.resource)
	return b
}

func (n *forwardingRuleNode) createActions() ([]exec.Action, error) {
	want, err := rnode.CreatePreconditions(n)
	if err != nil {
		return nil, err
	}
	return []exec.Action{
		newForwardingRuleCreateAction(n.ID(), n.resource, want),
	}, nil
}

func (n *forwardingRuleNode) updateActions(ngot rnode.Node) ([]exec.Action, error) {
	details := n.Plan().Details()
	if details == nil {
		return nil, nodeErr("updateActions: node %s has not been planned", n.ID())
	}
	got, ok := ngot.(*forwardingRuleNode)
	if !ok {
		return nil, nodeErr("updateActions: node %s has invalid type %T", n.ID(), ngot)
	}

	act := &forwardingRuleUpdateAction{id: n.ID()}

	var changed changedFields
	for _, item := range details.Diff.Items {
		if !changed.process(item) {
			return nil, nodeErr("updateActions %s: field %s cannot be updated in place", n.ID(), item.Path)
		}
	}
	if changed.target {
		oldTarget, err := parseTarget(fmt.Sprintf("updateActions %s", n.ID()), got)
		if err != nil {
			return nil, err
		}
		target, err := parseTarget(fmt.Sprintf("updateActions %s", n.ID()), n)
		if err != nil {
			return nil, err
		}
		act.Want = append(act.Want, exec.NewExistsEvent(target))
		act.oldTarget = oldTarget
		act.target = target
	}

	if changed.labels {
		gotRes, _ := got.resource.ToGA()
		wantRes, _ := n.resource.ToGA()
		act.labelFingerprint = gotRes.LabelFingerprint
		act.labels = wantRes.Labels
	}

	return []exec.Action{
		// Action: Signal resource exists.
		exec.NewExistsAction(n.ID()),
		// Action: Do the updates.
		act,
	}, nil
}

func parseTarget(errPrefix string, n *forwardingRuleNode) (*cloud.ResourceID, error) {
	res, _ := n.resource.ToGA()
	ret, err := cloud.ParseResourceURL(res.Target)
	if err != nil {
		return nil, nodeErr("%s: invalid .Target %q: %w", errPrefix, res.Target, err)
	}
	return ret, nil
}
