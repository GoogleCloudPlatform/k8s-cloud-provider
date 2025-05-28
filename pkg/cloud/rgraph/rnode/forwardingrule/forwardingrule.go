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
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"

	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

const (
	resourcePlural = "forwardingRules"
)

func ID(project string, key *meta.Key) *cloud.ResourceID {
	return &cloud.ResourceID{
		Resource:  resourcePlural,
		APIGroup:  meta.APIGroupCompute,
		ProjectID: project,
		Key:       key,
	}
}

type Resource = api.Resource[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule]
type Mutable = api.MutableResource[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule]

func NewWithTraits(project string, key *meta.Key, tr api.TypeTrait[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule]) Mutable {
	id := ID(project, key)
	return api.NewResource[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule](id, tr)
}

func New(project string, key *meta.Key) Mutable { return NewWithTraits(project, key, &TypeTrait{}) }
