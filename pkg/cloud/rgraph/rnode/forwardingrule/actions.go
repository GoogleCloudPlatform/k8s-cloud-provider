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
	"context"
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"google.golang.org/api/compute/v1"
)

func forwardingRuleSetLabels(
	ctx context.Context,
	cl cloud.Cloud,
	key *meta.Key,
	labelFingerprint string,
	labels map[string]string,
) error {
	switch key.Type() {
	case meta.Global:
		return cl.GlobalForwardingRules().SetLabels(ctx, key, &compute.GlobalSetLabelsRequest{
			LabelFingerprint: labelFingerprint,
			Labels:           labels,
		})
	case meta.Regional:
		return cl.ForwardingRules().SetLabels(ctx, key, &compute.RegionSetLabelsRequest{
			LabelFingerprint: labelFingerprint,
			Labels:           labels,
		})
	}
	return fmt.Errorf("forwardingRuleMethodsByScope: invalid scope %v", key.Type())
}

func newForwardingRuleCreateAction(id *cloud.ResourceID, res Resource, want exec.EventList) exec.Action {
	return &forwardingRuleCreateAction{
		ActionBase: exec.ActionBase{Want: want},
		id:         id,
		res:        res,
	}
}

type forwardingRuleCreateAction struct {
	exec.ActionBase
	id  *cloud.ResourceID
	res Resource
}

func (act *forwardingRuleCreateAction) Run(ctx context.Context, cl cloud.Cloud) (exec.EventList, error) {
	// XXX: project routing
	ops := &ops{}
	err := ops.CreateFuncs(cl).Do(ctx, act.id, act.res)
	if err != nil {
		return nil, err
	}

	ga, _ := act.res.ToGA()
	labels := ga.Labels
	if labels != nil && len(labels) > 0 {
		res, err := ops.GetFuncs(cl).Do(ctx, meta.VersionGA, act.id, &TypeTrait{})
		if err != nil {
			return nil, err

		}
		ga, _ = res.ToGA()
		if err := forwardingRuleSetLabels(ctx, cl, act.id.Key, ga.LabelFingerprint, labels); err != nil {
			return nil, err
		}
	}

	return exec.EventList{exec.NewExistsEvent(act.id)}, nil
}

func (act *forwardingRuleCreateAction) DryRun() exec.EventList {
	return exec.EventList{exec.NewExistsEvent(act.id)}
}

func (act *forwardingRuleCreateAction) String() string {
	return fmt.Sprintf("ForwardingRuleCreateAction(%s)", act.id)
}

func (act *forwardingRuleCreateAction) Metadata() *exec.ActionMetadata {
	return &exec.ActionMetadata{
		Name:    fmt.Sprintf("ForwardingRuleCreateAction(%s)", act.id),
		Type:    exec.ActionTypeCreate,
		Summary: fmt.Sprintf("Create %s", act.id),
	}
}

type forwardingRuleUpdateAction struct {
	exec.ActionBase

	id *cloud.ResourceID
	// target if non-empty will call setTarget(),
	target *cloud.ResourceID
	// oldTarget is the previous target before the update.
	oldTarget *cloud.ResourceID

	// labelFingerprint for the update operation.
	labelFingerprint string
	// labels if non-nil will call setLabels().
	labels map[string]string
}

func (act *forwardingRuleUpdateAction) Run(ctx context.Context, cl cloud.Cloud) (exec.EventList, error) {
	// TODO: project routing.
	if act.labels != nil {
		switch act.id.Key.Type() {
		case meta.Global:
			err := cl.GlobalForwardingRules().SetLabels(ctx, act.id.Key, &compute.GlobalSetLabelsRequest{
				LabelFingerprint: act.labelFingerprint,
				Labels:           act.labels,
			})
			if err != nil {
				return nil, fmt.Errorf("forwardingRuleUpdateAction Run(%s): SetLabels: %w", act.id, err)
			}
		case meta.Regional:
			err := cl.ForwardingRules().SetLabels(ctx, act.id.Key, &compute.RegionSetLabelsRequest{
				LabelFingerprint: act.labelFingerprint,
				Labels:           act.labels,
			})
			if err != nil {
				return nil, fmt.Errorf("forwardingRuleUpdateAction Run(%s): SetLabels: %w", act.id, err)
			}
		default:
			return nil, fmt.Errorf("forwardingRuleUpdateAction Run(%s): invalid key type", act.id)
		}
	}

	if act.target != nil {
		switch act.id.Key.Type() {
		case meta.Global:
			err := cl.GlobalForwardingRules().SetTarget(ctx, act.id.Key, &compute.TargetReference{
				Target: act.target.SelfLink(meta.VersionGA),
			})
			if err != nil {
				return nil, fmt.Errorf("forwardingRuleUpdateAction Run(%s): SetTarget: %w", act.id, err)
			}
		case meta.Regional:
			err := cl.GlobalForwardingRules().SetTarget(ctx, act.id.Key, &compute.TargetReference{
				Target: act.target.SelfLink(meta.VersionGA),
			})
			if err != nil {
				return nil, fmt.Errorf("forwardingRuleUpdateAction Run(%s): SetTarget: %w", act.id, err)
			}
		default:
			return nil, fmt.Errorf("forwardingRuleUpdateAction Run(%s): invalid key type", act.id)
		}
	}

	var events exec.EventList
	if act.oldTarget != nil && !act.target.Equal(act.oldTarget) {
		events = append(events, exec.NewDropRefEvent(act.id, act.oldTarget))
	}

	return events, nil
}

func (act *forwardingRuleUpdateAction) DryRun() exec.EventList {
	var events exec.EventList
	if act.oldTarget != nil && !act.target.Equal(act.oldTarget) {
		events = append(events, exec.NewDropRefEvent(act.id, act.oldTarget))
	}
	return events
}

func (act *forwardingRuleUpdateAction) String() string {
	return fmt.Sprintf("ForwardingRuleUpdateAction(%s)", act.id)
}

func (act *forwardingRuleUpdateAction) Metadata() *exec.ActionMetadata {
	return &exec.ActionMetadata{
		Name:    fmt.Sprintf("ForwardingRuleUpdateAction(%s)", act.id),
		Type:    exec.ActionTypeUpdate,
		Summary: fmt.Sprintf("Update %s", act.id),
	}
}
