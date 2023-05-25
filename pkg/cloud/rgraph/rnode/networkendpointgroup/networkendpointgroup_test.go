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

package networkendpointgroup

import (
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
)

func TestNetworkEndpointGroupSchema(t *testing.T) {
	const proj = "proj-1"
	key := meta.GlobalKey("key-1")
	x := NewMutableNetworkEndpointGroup(proj, key)
	if err := x.CheckSchema(); err != nil {
		t.Fatalf("CheckSchema() = %v, want nil", err)
	}
}
