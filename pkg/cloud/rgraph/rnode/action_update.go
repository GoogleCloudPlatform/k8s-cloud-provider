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
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
)

func UpdateActions[GA any, Alpha any, Beta any](
	ops GenericOps[GA, Alpha, Beta],
	got, want Node,
	resource api.Resource[GA, Alpha, Beta],
) ([]exec.Action, error) {
	preEvents, err := updatePreconditions(got, want)
	if err != nil {
		return nil, err
	}
	postEvents := postUpdateActionEvents(got, want)
	return []exec.Action{
		newGenericUpdateAction(preEvents, ops, want.ID(), resource, postEvents),
	}, nil
}

func newGenericUpdateAction[GA any, Alpha any, Beta any](
	want exec.EventList,
	ops GenericOps[GA, Alpha, Beta],
	id *cloud.ResourceID,
	resource api.Resource[GA, Alpha, Beta],
	postEvents exec.EventList,
) *genericUpdateAction[GA, Alpha, Beta] {
	return &genericUpdateAction[GA, Alpha, Beta]{
		ActionBase: exec.ActionBase{Want: want},
		ops:        ops,
		id:         id,
		resource:   resource,
		postEvents: postEvents,
	}
}

type genericUpdateAction[GA any, Alpha any, Beta any] struct {
	exec.ActionBase
	ops        GenericOps[GA, Alpha, Beta]
	id         *cloud.ResourceID
	resource   api.Resource[GA, Alpha, Beta]
	postEvents exec.EventList

	start, end time.Time
}

func (a *genericUpdateAction[GA, Alpha, Beta]) Run(
	ctx context.Context,
	c cloud.Cloud,
) (exec.EventList, error) {
	a.start = time.Now()
	err := a.ops.UpdateFuncs(c).Do(ctx, "", a.id, a.resource)
	a.end = time.Now()

	// Emit DropReference events for removed references.
	return a.postEvents, err
}

func (a *genericUpdateAction[GA, Alpha, Beta]) DryRun() exec.EventList {
	// Emit DropReference events for removed references.
	return a.postEvents
}

func (a *genericUpdateAction[GA, Alpha, Beta]) String() string {
	return fmt.Sprintf("GenericUpdateAction(%v)", a.id)
}

func (a *genericUpdateAction[GA, Alpha, Beta]) Metadata() *exec.ActionMetadata {
	return &exec.ActionMetadata{
		Name:    fmt.Sprintf("GenericUpdateAction(%s)", a.id),
		Type:    exec.ActionTypeUpdate,
		Summary: fmt.Sprintf("Update %s", a.id),
	}
}

func updatePreconditions(got, want Node) (exec.EventList, error) {
	// Update can only occur if the resource Exists TODO: is there a case where
	// the ambient signal for existance from Update op collides with a
	// reference to it?
	if got.State() != NodeExists || want.State() != NodeExists {
		return nil, fmt.Errorf("node for update does not exist")
	}

	outRefs := want.OutRefs()
	var events exec.EventList
	// Condition: references must exist before update.
	for _, ref := range outRefs {
		events = append(events, exec.NewExistsEvent(ref.To))
	}
	return events, nil
}

func postUpdateActionEvents(got, want Node) exec.EventList {
	wantOutRefs := want.OutRefs()
	gotOutRefs := got.OutRefs()

	wantRefs := make(map[meta.Key]struct{})
	for _, r := range wantOutRefs {
		var empty struct{}
		wantRefs[*r.To.Key] = empty
	}

	// Drop reference for resources that does not exists in want Node.
	var events exec.EventList
	for _, wantRef := range gotOutRefs {
		_, ok := wantRefs[*wantRef.To.Key]
		if !ok {
			events = append(events, exec.NewDropRefEvent(wantRef.From, wantRef.To))
		}
	}

	return events
}
