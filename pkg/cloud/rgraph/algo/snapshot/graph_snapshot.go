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

package snapshot

// TODO: fixme

import (
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	compute "google.golang.org/api/compute/v0.beta"
)

func NewGraphSnapshot() *GraphSnapshot {
	ret := &GraphSnapshot{
		Address:              map[string]NodeSnapshot[compute.Address, alpha.Address, beta.Address]{},
		BackendService:       map[string]NodeSnapshot[compute.BackendService, alpha.BackendService, beta.BackendService]{},
		ForwardingRule:       map[string]NodeSnapshot[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule]{},
		HealthCheck:          map[string]NodeSnapshot[compute.HealthCheck, alpha.HealthCheck, beta.HealthCheck]{},
		NetworkEndpointGroup: map[string]NodeSnapshot[compute.NetworkEndpointGroup, alpha.NetworkEndpointGroup, beta.NetworkEndpointGroup]{},
		TargetHttpProxy:      map[string]NodeSnapshot[compute.TargetHttpProxy, alpha.TargetHttpProxy, beta.TargetHttpProxy]{},
		UrlMap:               map[string]NodeSnapshot[compute.UrlMap, alpha.UrlMap, beta.UrlMap]{},
	}
	return ret
}

// GraphSnapshot is a JSON-serializable snapshot of a resource graph. This data
// structure is used to save and load resource graphs for testing and debugging.
type GraphSnapshot struct {
	Address              map[string]NodeSnapshot[compute.Address, alpha.Address, beta.Address]
	BackendService       map[string]NodeSnapshot[compute.BackendService, alpha.BackendService, beta.BackendService]
	ForwardingRule       map[string]NodeSnapshot[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule]
	HealthCheck          map[string]NodeSnapshot[compute.HealthCheck, alpha.HealthCheck, beta.HealthCheck]
	NetworkEndpointGroup map[string]NodeSnapshot[compute.NetworkEndpointGroup, alpha.NetworkEndpointGroup, beta.NetworkEndpointGroup]
	TargetHttpProxy      map[string]NodeSnapshot[compute.TargetHttpProxy, alpha.TargetHttpProxy, beta.TargetHttpProxy]
	UrlMap               map[string]NodeSnapshot[compute.UrlMap, alpha.UrlMap, beta.UrlMap]
}

type ResourceSnapshot[GA any, Alpha any, Beta any] struct {
	GA    *GA
	Alpha *Alpha
	Beta  *Beta
}

type NodeSnapshot[GA any, Alpha any, Beta any] struct {
	Desired *ResourceSnapshot[GA, Alpha, Beta]
	Current *ResourceSnapshot[GA, Alpha, Beta]
	SyncOp  rnode.Operation
}
