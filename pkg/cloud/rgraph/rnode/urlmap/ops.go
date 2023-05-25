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
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"

	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

type urlMapOps struct{}

func (*urlMapOps) GetFuncs(gcp cloud.Cloud) *rnode.GetFuncs[compute.UrlMap, alpha.UrlMap, beta.UrlMap] {
	return &rnode.GetFuncs[compute.UrlMap, alpha.UrlMap, beta.UrlMap]{
		GA: rnode.GetFuncsByScope[compute.UrlMap]{
			Global:   gcp.UrlMaps().Get,
			Regional: gcp.RegionUrlMaps().Get,
		},
		Alpha: rnode.GetFuncsByScope[alpha.UrlMap]{
			Global:   gcp.AlphaUrlMaps().Get,
			Regional: gcp.AlphaRegionUrlMaps().Get,
		},
		Beta: rnode.GetFuncsByScope[beta.UrlMap]{
			Global:   gcp.BetaUrlMaps().Get,
			Regional: gcp.BetaRegionUrlMaps().Get,
		},
	}
}

func (*urlMapOps) CreateFuncs(gcp cloud.Cloud) *rnode.CreateFuncs[compute.UrlMap, alpha.UrlMap, beta.UrlMap] {
	return &rnode.CreateFuncs[compute.UrlMap, alpha.UrlMap, beta.UrlMap]{
		GA: rnode.CreateFuncsByScope[compute.UrlMap]{
			Global:   gcp.UrlMaps().Insert,
			Regional: gcp.RegionUrlMaps().Insert,
		},
		Alpha: rnode.CreateFuncsByScope[alpha.UrlMap]{
			Global:   gcp.AlphaUrlMaps().Insert,
			Regional: gcp.AlphaRegionUrlMaps().Insert,
		},
		Beta: rnode.CreateFuncsByScope[beta.UrlMap]{
			Global:   gcp.BetaUrlMaps().Insert,
			Regional: gcp.BetaRegionUrlMaps().Insert,
		},
	}
}

func (*urlMapOps) UpdateFuncs(gcp cloud.Cloud) *rnode.UpdateFuncs[compute.UrlMap, alpha.UrlMap, beta.UrlMap] {
	return &rnode.UpdateFuncs[compute.UrlMap, alpha.UrlMap, beta.UrlMap]{
		GA: rnode.UpdateFuncsByScope[compute.UrlMap]{
			Global:   gcp.UrlMaps().Update,
			Regional: gcp.RegionUrlMaps().Update,
		},
		Alpha: rnode.UpdateFuncsByScope[alpha.UrlMap]{
			Global:   gcp.AlphaUrlMaps().Update,
			Regional: gcp.AlphaRegionUrlMaps().Update,
		},
		Beta: rnode.UpdateFuncsByScope[beta.UrlMap]{
			Global:   gcp.BetaUrlMaps().Update,
			Regional: gcp.BetaRegionUrlMaps().Update,
		},
	}
}

func (*urlMapOps) DeleteFuncs(gcp cloud.Cloud) *rnode.DeleteFuncs[compute.UrlMap, alpha.UrlMap, beta.UrlMap] {
	return &rnode.DeleteFuncs[compute.UrlMap, alpha.UrlMap, beta.UrlMap]{
		GA: rnode.DeleteFuncsByScope[compute.UrlMap]{
			Global:   gcp.UrlMaps().Delete,
			Regional: gcp.RegionUrlMaps().Delete,
		},
		Alpha: rnode.DeleteFuncsByScope[alpha.UrlMap]{
			Global:   gcp.AlphaUrlMaps().Delete,
			Regional: gcp.AlphaRegionUrlMaps().Delete,
		},
		Beta: rnode.DeleteFuncsByScope[beta.UrlMap]{
			Global:   gcp.BetaUrlMaps().Delete,
			Regional: gcp.BetaRegionUrlMaps().Delete,
		},
	}
}
