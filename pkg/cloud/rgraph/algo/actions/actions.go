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

package actions

import (
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
)

// Do accumulates all of the Actions for executing a plan to transform
// got to want.
func Do(got, want *rgraph.Graph) ([]exec.Action, error) {
	var actions []exec.Action
	for _, n := range want.All() {
		gotNode := got.Get(n.ID())
		if gotNode == nil {
			return nil, fmt.Errorf("actions: `got` is missing node %s that is in `want`", n.ID())
		}
		act, err := n.Actions(gotNode)
		if err != nil {
			return nil, err
		}
		actions = append(actions, act...)
	}
	return actions, nil
}
