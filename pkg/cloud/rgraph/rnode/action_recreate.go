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
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
)

func RecreateActions[GA any, Alpha any, Beta any](
	ops GenericOps[GA, Alpha, Beta],
	got, want Node,
	resource api.Resource[GA, Alpha, Beta],
) ([]exec.Action, error) {
	deleteAction := NewGenericDeleteAction(DeletePreconditions(got, want), ops, got)

	createEvents, err := CreatePreconditions(want)
	if err != nil {
		return nil, err
	}
	// Condition: resource must have been deleted.
	createEvents = append(createEvents, exec.NewNotExistsEvent(want.ID()))
	createAction := newGenericCreateAction(createEvents, ops, want.ID(), resource)

	return []exec.Action{deleteAction, createAction}, nil
}
