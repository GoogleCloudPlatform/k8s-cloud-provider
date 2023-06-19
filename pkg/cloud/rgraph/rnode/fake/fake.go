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

package fake

import (
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
)

// fakeResource is a resource used only for testing.
type fakeResource struct {
	Name            string
	Dependencies    []string
	NullFields      []string
	ForceSendFields []string
}

// ID for the resource.
func ID(project string, key *meta.Key) *cloud.ResourceID {
	return &cloud.ResourceID{
		Resource:  "fakes",
		ProjectID: project,
		Key:       key,
	}
}

type mutableFake = api.MutableResource[fakeResource, fakeResource, fakeResource]

func newMutableFake(project string, key *meta.Key) mutableFake {
	res := &cloud.ResourceID{
		Resource:  "fakes",
		ProjectID: project,
		Key:       key,
	}
	return api.NewResource[fakeResource, fakeResource, fakeResource](res, &fakeTypeTrait{})
}

// Fake resource for testing.
type Fake = api.Resource[fakeResource, fakeResource, fakeResource]

type fakeTypeTrait struct {
	api.BaseTypeTrait[fakeResource, fakeResource, fakeResource]
}
