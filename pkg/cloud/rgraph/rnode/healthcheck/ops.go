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

package healthcheck

import (
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

type healthCheckOps struct{}

func (*healthCheckOps) GetFuncs(gcp cloud.Cloud) *rnode.GetFuncs[compute.HealthCheck, alpha.HealthCheck, beta.HealthCheck] {
	return &rnode.GetFuncs[compute.HealthCheck, alpha.HealthCheck, beta.HealthCheck]{
		GA: rnode.GetFuncsByScope[compute.HealthCheck]{
			Global:   gcp.HealthChecks().Get,
			Regional: gcp.RegionHealthChecks().Get,
		},
		Alpha: rnode.GetFuncsByScope[alpha.HealthCheck]{
			Global:   gcp.AlphaHealthChecks().Get,
			Regional: gcp.AlphaRegionHealthChecks().Get,
		},
		Beta: rnode.GetFuncsByScope[beta.HealthCheck]{
			Global:   gcp.BetaHealthChecks().Get,
			Regional: gcp.BetaRegionHealthChecks().Get,
		},
	}
}

func (*healthCheckOps) CreateFuncs(gcp cloud.Cloud) *rnode.CreateFuncs[compute.HealthCheck, alpha.HealthCheck, beta.HealthCheck] {
	return &rnode.CreateFuncs[compute.HealthCheck, alpha.HealthCheck, beta.HealthCheck]{
		GA: rnode.CreateFuncsByScope[compute.HealthCheck]{
			Global:   gcp.HealthChecks().Insert,
			Regional: gcp.RegionHealthChecks().Insert,
		},
		Alpha: rnode.CreateFuncsByScope[alpha.HealthCheck]{
			Global:   gcp.AlphaHealthChecks().Insert,
			Regional: gcp.AlphaRegionHealthChecks().Insert,
		},
		Beta: rnode.CreateFuncsByScope[beta.HealthCheck]{
			Global:   gcp.BetaHealthChecks().Insert,
			Regional: gcp.BetaRegionHealthChecks().Insert,
		},
	}
}

func (*healthCheckOps) UpdateFuncs(gcp cloud.Cloud) *rnode.UpdateFuncs[compute.HealthCheck, alpha.HealthCheck, beta.HealthCheck] {
	return &rnode.UpdateFuncs[compute.HealthCheck, alpha.HealthCheck, beta.HealthCheck]{
		GA: rnode.UpdateFuncsByScope[compute.HealthCheck]{
			Global:   gcp.HealthChecks().Update,
			Regional: gcp.RegionHealthChecks().Update,
		},
		Alpha: rnode.UpdateFuncsByScope[alpha.HealthCheck]{
			Global:   gcp.AlphaHealthChecks().Update,
			Regional: gcp.AlphaRegionHealthChecks().Update,
		},
		Beta: rnode.UpdateFuncsByScope[beta.HealthCheck]{
			Global:   gcp.BetaHealthChecks().Update,
			Regional: gcp.BetaRegionHealthChecks().Update,
		},
		Options: rnode.UpdateFuncsNoFingerprint,
	}
}

func (*healthCheckOps) DeleteFuncs(gcp cloud.Cloud) *rnode.DeleteFuncs[compute.HealthCheck, alpha.HealthCheck, beta.HealthCheck] {
	return &rnode.DeleteFuncs[compute.HealthCheck, alpha.HealthCheck, beta.HealthCheck]{
		GA: rnode.DeleteFuncsByScope[compute.HealthCheck]{
			Global:   gcp.HealthChecks().Delete,
			Regional: gcp.RegionHealthChecks().Delete,
		},
		Alpha: rnode.DeleteFuncsByScope[alpha.HealthCheck]{
			Global:   gcp.AlphaHealthChecks().Delete,
			Regional: gcp.AlphaRegionHealthChecks().Delete,
		},
		Beta: rnode.DeleteFuncsByScope[beta.HealthCheck]{
			Global:   gcp.BetaHealthChecks().Delete,
			Regional: gcp.BetaRegionHealthChecks().Delete,
		},
	}
}
