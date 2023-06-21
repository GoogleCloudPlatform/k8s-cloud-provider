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

// FakeResource is a resource used only for testing.
type FakeResource struct {
	Name string
	// Value is compared in Diff.
	Value string
	// Dependencies are URLs to other resources i.e. OutRefs.
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

type MutableFake = api.MutableResource[FakeResource, FakeResource, FakeResource]

func NewMutableFake(project string, key *meta.Key) MutableFake {
	res := &cloud.ResourceID{
		Resource:  "fakes",
		ProjectID: project,
		Key:       key,
	}
	return api.NewResource[FakeResource, FakeResource, FakeResource](res, &fakeTypeTrait{})
}

// Fake resource for testing.
type Fake = api.Resource[FakeResource, FakeResource, FakeResource]

type fakeTypeTrait struct {
	api.BaseTypeTrait[FakeResource, FakeResource, FakeResource]
}
