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

package all

import (
	"fmt"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/address"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/backendservice"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/fake"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/forwardingrule"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/healthcheck"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/networkendpointgroup"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/targethttpproxy"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/urlmap"
)

func NewBuilderByID(id *cloud.ResourceID) (rnode.Builder, error) {
	switch id.Resource {
	case "fakes":
		return fake.NewBuilder(id), nil
	case "addresses":
		return address.NewBuilder(id), nil
	case "backendServices":
		return backendservice.NewBuilder(id), nil
	case "fakes":
		return fake.NewBuilder(id), nil
	case "forwardingRules":
		return forwardingrule.NewBuilder(id), nil
	case "healthChecks":
		return healthcheck.NewBuilder(id), nil
	case "networkEndpointGroups":
		return networkendpointgroup.NewBuilder(id), nil
	case "targetHttpProxies":
		return targethttpproxy.NewBuilder(id), nil
	case "urlMaps":
		return urlmap.NewBuilder(id), nil
	}
	return nil, fmt.Errorf("NewBuilderByID: invalid Resource %q", id.Resource)
}
