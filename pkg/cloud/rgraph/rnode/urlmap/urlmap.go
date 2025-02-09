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

package urlmap

import (
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"

	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

const (
	resourcePlural = "urlMaps"
)

func ID(project string, key *meta.Key) *cloud.ResourceID {
	return &cloud.ResourceID{
		Resource:  resourcePlural,
		APIGroup:  meta.APIGroupCompute,
		ProjectID: project,
		Key:       key,
	}
}

type MutableUrlMap = api.MutableResource[compute.UrlMap, alpha.UrlMap, beta.UrlMap]

func NewMutableUrlMapWithTraits(project string, key *meta.Key, tr api.TypeTrait[compute.UrlMap, alpha.UrlMap, beta.UrlMap]) MutableUrlMap {
	id := ID(project, key)
	return api.NewResource[compute.UrlMap, alpha.UrlMap, beta.UrlMap](id, tr)
}

func NewMutableUrlMap(project string, key *meta.Key) MutableUrlMap {
	return NewMutableUrlMapWithTraits(project, key, &TypeTrait{})
}

type UrlMap = api.Resource[compute.UrlMap, alpha.UrlMap, beta.UrlMap]
