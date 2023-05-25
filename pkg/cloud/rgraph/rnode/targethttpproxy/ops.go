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

package targethttpproxy

import (
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

type targetHttpProxyOps struct{}

func (*targetHttpProxyOps) GetFuncs(gcp cloud.Cloud) *rnode.GetFuncs[compute.TargetHttpProxy, alpha.TargetHttpProxy, beta.TargetHttpProxy] {
	return &rnode.GetFuncs[compute.TargetHttpProxy, alpha.TargetHttpProxy, beta.TargetHttpProxy]{
		GA: rnode.GetFuncsByScope[compute.TargetHttpProxy]{
			Global:   gcp.TargetHttpProxies().Get,
			Regional: gcp.RegionTargetHttpProxies().Get,
		},
		Alpha: rnode.GetFuncsByScope[alpha.TargetHttpProxy]{
			Global:   gcp.AlphaTargetHttpProxies().Get,
			Regional: gcp.AlphaRegionTargetHttpProxies().Get,
		},
		Beta: rnode.GetFuncsByScope[beta.TargetHttpProxy]{
			Global:   gcp.BetaTargetHttpProxies().Get,
			Regional: gcp.BetaRegionTargetHttpProxies().Get,
		},
	}
}

func (*targetHttpProxyOps) CreateFuncs(gcp cloud.Cloud) *rnode.CreateFuncs[compute.TargetHttpProxy, alpha.TargetHttpProxy, beta.TargetHttpProxy] {
	return &rnode.CreateFuncs[compute.TargetHttpProxy, alpha.TargetHttpProxy, beta.TargetHttpProxy]{
		GA: rnode.CreateFuncsByScope[compute.TargetHttpProxy]{
			Global:   gcp.TargetHttpProxies().Insert,
			Regional: gcp.RegionTargetHttpProxies().Insert,
		},
		Alpha: rnode.CreateFuncsByScope[alpha.TargetHttpProxy]{
			Global:   gcp.AlphaTargetHttpProxies().Insert,
			Regional: gcp.AlphaRegionTargetHttpProxies().Insert,
		},
		Beta: rnode.CreateFuncsByScope[beta.TargetHttpProxy]{
			Global:   gcp.BetaTargetHttpProxies().Insert,
			Regional: gcp.BetaRegionTargetHttpProxies().Insert,
		},
	}
}

func (*targetHttpProxyOps) UpdateFuncs(gcp cloud.Cloud) *rnode.UpdateFuncs[compute.TargetHttpProxy, alpha.TargetHttpProxy, beta.TargetHttpProxy] {
	return nil // Does not support generic Update.
}

func (*targetHttpProxyOps) DeleteFuncs(gcp cloud.Cloud) *rnode.DeleteFuncs[compute.TargetHttpProxy, alpha.TargetHttpProxy, beta.TargetHttpProxy] {
	return &rnode.DeleteFuncs[compute.TargetHttpProxy, alpha.TargetHttpProxy, beta.TargetHttpProxy]{
		GA: rnode.DeleteFuncsByScope[compute.TargetHttpProxy]{
			Global:   gcp.TargetHttpProxies().Delete,
			Regional: gcp.RegionTargetHttpProxies().Delete,
		},
		Alpha: rnode.DeleteFuncsByScope[alpha.TargetHttpProxy]{
			Global:   gcp.AlphaTargetHttpProxies().Delete,
			Regional: gcp.AlphaRegionTargetHttpProxies().Delete,
		},
		Beta: rnode.DeleteFuncsByScope[beta.TargetHttpProxy]{
			Global:   gcp.BetaTargetHttpProxies().Delete,
			Regional: gcp.BetaRegionTargetHttpProxies().Delete,
		},
	}
}
