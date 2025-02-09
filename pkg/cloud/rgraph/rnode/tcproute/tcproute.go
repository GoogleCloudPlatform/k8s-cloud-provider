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

package tcproute

import (
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"

	"google.golang.org/api/networkservices/v1"
	beta "google.golang.org/api/networkservices/v1beta1"
)

const (
	resourcePlural = "tcpRoutes"
)

func ID(project string, key *meta.Key) *cloud.ResourceID {
	return &cloud.ResourceID{
		Resource:  resourcePlural,
		APIGroup:  meta.APIGroupNetworkServices,
		ProjectID: project,
		Key:       key,
	}
}

type Resource = api.Resource[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute]
type Mutable = api.MutableResource[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute]

func NewWithTraits(project string, key *meta.Key, tr api.TypeTrait[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute]) Mutable {
	id := ID(project, key)
	return api.NewResource[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute](id, tr)
}

func New(project string, key *meta.Key) Mutable { return NewWithTraits(project, key, &TypeTrait{}) }
