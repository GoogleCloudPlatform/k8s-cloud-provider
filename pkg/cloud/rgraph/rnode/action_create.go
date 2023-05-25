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
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
)

func CreatePreconditions(want Node) (exec.EventList, error) {
	outRefs := want.OutRefs()
	var events exec.EventList
	// Condition: references must exist before creation.
	for _, ref := range outRefs {
		events = append(events, exec.NewExistsEvent(ref.To))
	}
	return events, nil
}

func CreateActions[GA any, Alpha any, Beta any](
	ops GenericOps[GA, Alpha, Beta],
	node Node,
	resource api.Resource[GA, Alpha, Beta],
) ([]exec.Action, error) {
	events, err := CreatePreconditions(node)
	if err != nil {
		return nil, err
	}
	return []exec.Action{
		newGenericCreateAction(events, ops, node.ID(), resource),
	}, nil
}

func newGenericCreateAction[GA any, Alpha any, Beta any](
	want exec.EventList,
	ops GenericOps[GA, Alpha, Beta],
	id *cloud.ResourceID,
	resource api.Resource[GA, Alpha, Beta],
) *genericCreateAction[GA, Alpha, Beta] {
	return &genericCreateAction[GA, Alpha, Beta]{
		ActionBase: exec.ActionBase{Want: want},
		ops:        ops,
		id:         id,
		resource:   resource,
	}
}

type genericCreateAction[GA any, Alpha any, Beta any] struct {
	exec.ActionBase
	ops      GenericOps[GA, Alpha, Beta]
	id       *cloud.ResourceID
	resource api.Resource[GA, Alpha, Beta]

	start, end time.Time
}

func (a *genericCreateAction[GA, Alpha, Beta]) Run(
	ctx context.Context,
	c cloud.Cloud,
) (exec.EventList, error) {
	a.start = time.Now()
	err := a.ops.CreateFuncs(c).Do(ctx, a.id, a.resource)
	a.end = a.start

	return exec.EventList{exec.NewExistsEvent(a.id)}, err
}

func (a *genericCreateAction[GA, Alpha, Beta]) DryRun() exec.EventList {
	a.start = time.Now()
	a.end = a.start
	return exec.EventList{exec.NewExistsEvent(a.id)}
}

func (a *genericCreateAction[GA, Alpha, Beta]) String() string {
	return fmt.Sprintf("GenericCreateAction(%v)", a.id)
}

func (a *genericCreateAction[GA, Alpha, Beta]) Metadata() *exec.ActionMetadata {
	return &exec.ActionMetadata{
		Name:    fmt.Sprintf("GenericCreateAction(%s)", a.id),
		Type:    exec.ActionTypeCreate,
		Summary: fmt.Sprintf("Create %s", a.id),
	}
}
