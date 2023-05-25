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
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

type ops struct{}

func (*ops) GetFuncs(gcp cloud.Cloud) *rnode.GetFuncs[compute.NetworkEndpointGroup, alpha.NetworkEndpointGroup, beta.NetworkEndpointGroup] {
	return &rnode.GetFuncs[compute.NetworkEndpointGroup, alpha.NetworkEndpointGroup, beta.NetworkEndpointGroup]{
		GA: rnode.GetFuncsByScope[compute.NetworkEndpointGroup]{
			Zonal: gcp.NetworkEndpointGroups().Get,
		},
		Alpha: rnode.GetFuncsByScope[alpha.NetworkEndpointGroup]{
			Zonal: gcp.AlphaNetworkEndpointGroups().Get,
		},
		Beta: rnode.GetFuncsByScope[beta.NetworkEndpointGroup]{
			Zonal: gcp.BetaNetworkEndpointGroups().Get,
		},
	}
}

func (*ops) CreateFuncs(gcp cloud.Cloud) *rnode.CreateFuncs[compute.NetworkEndpointGroup, alpha.NetworkEndpointGroup, beta.NetworkEndpointGroup] {
	return &rnode.CreateFuncs[compute.NetworkEndpointGroup, alpha.NetworkEndpointGroup, beta.NetworkEndpointGroup]{
		GA: rnode.CreateFuncsByScope[compute.NetworkEndpointGroup]{
			Zonal: gcp.NetworkEndpointGroups().Insert,
		},
		Alpha: rnode.CreateFuncsByScope[alpha.NetworkEndpointGroup]{
			Zonal: gcp.AlphaNetworkEndpointGroups().Insert,
		},
		Beta: rnode.CreateFuncsByScope[beta.NetworkEndpointGroup]{
			Zonal: gcp.BetaNetworkEndpointGroups().Insert,
		},
	}
}

func (*ops) UpdateFuncs(gcp cloud.Cloud) *rnode.UpdateFuncs[compute.NetworkEndpointGroup, alpha.NetworkEndpointGroup, beta.NetworkEndpointGroup] {
	return nil // Does not support generic Update.
}

func (*ops) DeleteFuncs(gcp cloud.Cloud) *rnode.DeleteFuncs[compute.NetworkEndpointGroup, alpha.NetworkEndpointGroup, beta.NetworkEndpointGroup] {
	return &rnode.DeleteFuncs[compute.NetworkEndpointGroup, alpha.NetworkEndpointGroup, beta.NetworkEndpointGroup]{
		GA: rnode.DeleteFuncsByScope[compute.NetworkEndpointGroup]{
			Zonal: gcp.NetworkEndpointGroups().Delete,
		},
		Alpha: rnode.DeleteFuncsByScope[alpha.NetworkEndpointGroup]{
			Zonal: gcp.AlphaNetworkEndpointGroups().Delete,
		},
		Beta: rnode.DeleteFuncsByScope[beta.NetworkEndpointGroup]{
			Zonal: gcp.BetaNetworkEndpointGroups().Delete,
		},
	}
}
