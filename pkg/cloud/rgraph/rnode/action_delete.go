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
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
)

func NewGenericDeleteAction[GA any, Alpha any, Beta any](
	want exec.EventList,
	ops GenericOps[GA, Alpha, Beta],
	got Node,
) *genericDeleteAction[GA, Alpha, Beta] {
	return &genericDeleteAction[GA, Alpha, Beta]{
		ActionBase: exec.ActionBase{Want: want},
		ops:        ops,
		id:         got.ID(),
		outRefs:    got.OutRefs(),
	}
}

func DeletePreconditions(got, want Node) exec.EventList {
	var ret exec.EventList
	// Condition: no inRefs to the resource still exist.
	for _, ref := range got.InRefs() {
		ret = append(ret, exec.NewDropRefEvent(ref.From, ref.To))
	}
	return ret
}

func DeleteActions[GA any, Alpha any, Beta any](
	ops GenericOps[GA, Alpha, Beta],
	got, want Node,
) ([]exec.Action, error) {
	return []exec.Action{
		NewGenericDeleteAction(DeletePreconditions(got, want), ops, got),
	}, nil
}

type genericDeleteAction[GA any, Alpha any, Beta any] struct {
	exec.ActionBase
	ops     GenericOps[GA, Alpha, Beta]
	id      *cloud.ResourceID
	outRefs []ResourceRef

	start, end time.Time
}

func (a *genericDeleteAction[GA, Alpha, Beta]) Run(
	ctx context.Context,
	c cloud.Cloud,
) (exec.EventList, error) {
	a.start = time.Now()
	err := a.ops.DeleteFuncs(c).Do(ctx, a.id)

	var events exec.EventList
	// Event: Node no longer exists.
	events = append(events, exec.NewNotExistsEvent(a.id))
	for _, ref := range a.outRefs {
		events = append(events, exec.NewDropRefEvent(ref.From, ref.To))
	}

	a.end = time.Now()

	return events, err
}

func (a *genericDeleteAction[GA, Alpha, Beta]) DryRun() exec.EventList {
	a.start = time.Now()
	a.end = a.start
	return exec.EventList{exec.NewNotExistsEvent(a.id)}
}

func (a *genericDeleteAction[GA, Alpha, Beta]) String() string {
	return fmt.Sprintf("GenericDeleteAction(%v)", a.id)
}

func (a *genericDeleteAction[GA, Alpha, Beta]) Metadata() *exec.ActionMetadata {
	return &exec.ActionMetadata{
		Name:    fmt.Sprintf("GenericDeleteAction(%s)", a.id),
		Type:    exec.ActionTypeDelete,
		Summary: fmt.Sprintf("Delete %s", a.id),
	}
}
