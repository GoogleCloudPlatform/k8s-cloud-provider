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

package backendservice

import (
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"

	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

type ops struct{}

func (*ops) GetFuncs(gcp cloud.Cloud) *rnode.GetFuncs[compute.BackendService, alpha.BackendService, beta.BackendService] {
	return &rnode.GetFuncs[compute.BackendService, alpha.BackendService, beta.BackendService]{
		GA: rnode.GetFuncsByScope[compute.BackendService]{
			Global:   gcp.BackendServices().Get,
			Regional: gcp.RegionBackendServices().Get,
		},
		Alpha: rnode.GetFuncsByScope[alpha.BackendService]{
			Global:   gcp.AlphaBackendServices().Get,
			Regional: gcp.AlphaRegionBackendServices().Get,
		},
		Beta: rnode.GetFuncsByScope[beta.BackendService]{
			Global:   gcp.BetaBackendServices().Get,
			Regional: gcp.BetaRegionBackendServices().Get,
		},
	}
}

func (*ops) CreateFuncs(gcp cloud.Cloud) *rnode.CreateFuncs[compute.BackendService, alpha.BackendService, beta.BackendService] {
	return &rnode.CreateFuncs[compute.BackendService, alpha.BackendService, beta.BackendService]{
		GA: rnode.CreateFuncsByScope[compute.BackendService]{
			Global:   gcp.BackendServices().Insert,
			Regional: gcp.RegionBackendServices().Insert,
		},
		Alpha: rnode.CreateFuncsByScope[alpha.BackendService]{
			Global:   gcp.AlphaBackendServices().Insert,
			Regional: gcp.AlphaRegionBackendServices().Insert,
		},
		Beta: rnode.CreateFuncsByScope[beta.BackendService]{
			Global:   gcp.BetaBackendServices().Insert,
			Regional: gcp.BetaRegionBackendServices().Insert,
		},
	}
}

func (*ops) UpdateFuncs(gcp cloud.Cloud) *rnode.UpdateFuncs[compute.BackendService, alpha.BackendService, beta.BackendService] {
	return &rnode.UpdateFuncs[compute.BackendService, alpha.BackendService, beta.BackendService]{
		GA: rnode.UpdateFuncsByScope[compute.BackendService]{
			Global:   gcp.BackendServices().Update,
			Regional: gcp.RegionBackendServices().Update,
		},
		Alpha: rnode.UpdateFuncsByScope[alpha.BackendService]{
			Global:   gcp.AlphaBackendServices().Update,
			Regional: gcp.AlphaRegionBackendServices().Update,
		},
		Beta: rnode.UpdateFuncsByScope[beta.BackendService]{
			Global:   gcp.BetaBackendServices().Update,
			Regional: gcp.BetaRegionBackendServices().Update,
		},
	}
}

func (*ops) DeleteFuncs(gcp cloud.Cloud) *rnode.DeleteFuncs[compute.BackendService, alpha.BackendService, beta.BackendService] {
	return &rnode.DeleteFuncs[compute.BackendService, alpha.BackendService, beta.BackendService]{
		GA: rnode.DeleteFuncsByScope[compute.BackendService]{
			Global:   gcp.BackendServices().Delete,
			Regional: gcp.RegionBackendServices().Delete,
		},
		Alpha: rnode.DeleteFuncsByScope[alpha.BackendService]{
			Global:   gcp.AlphaBackendServices().Delete,
			Regional: gcp.AlphaRegionBackendServices().Delete,
		},
		Beta: rnode.DeleteFuncsByScope[beta.BackendService]{
			Global:   gcp.BetaBackendServices().Delete,
			Regional: gcp.BetaRegionBackendServices().Delete,
		},
	}
}
