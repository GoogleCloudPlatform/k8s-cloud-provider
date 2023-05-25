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

package address

import (
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"

	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

type ops struct{}

// ops implements GenericOps.
var _ rnode.GenericOps[compute.Address, alpha.Address, beta.Address] = (*ops)(nil)

func (*ops) GetFuncs(gcp cloud.Cloud) *rnode.GetFuncs[compute.Address, alpha.Address, beta.Address] {
	return &rnode.GetFuncs[compute.Address, alpha.Address, beta.Address]{
		GA: rnode.GetFuncsByScope[compute.Address]{
			Global:   gcp.GlobalAddresses().Get,
			Regional: gcp.Addresses().Get,
		},
		Alpha: rnode.GetFuncsByScope[alpha.Address]{
			Global:   gcp.AlphaGlobalAddresses().Get,
			Regional: gcp.AlphaAddresses().Get,
		},
		Beta: rnode.GetFuncsByScope[beta.Address]{
			Global:   gcp.BetaGlobalAddresses().Get,
			Regional: gcp.BetaAddresses().Get,
		},
	}
}

func (*ops) CreateFuncs(gcp cloud.Cloud) *rnode.CreateFuncs[compute.Address, alpha.Address, beta.Address] {
	return &rnode.CreateFuncs[compute.Address, alpha.Address, beta.Address]{
		GA: rnode.CreateFuncsByScope[compute.Address]{
			Global:   gcp.GlobalAddresses().Insert,
			Regional: gcp.Addresses().Insert,
		},
		Alpha: rnode.CreateFuncsByScope[alpha.Address]{
			Global:   gcp.AlphaGlobalAddresses().Insert,
			Regional: gcp.AlphaAddresses().Insert,
		},
		Beta: rnode.CreateFuncsByScope[beta.Address]{
			Global:   gcp.BetaGlobalAddresses().Insert,
			Regional: gcp.BetaAddresses().Insert,
		},
	}
}

func (*ops) UpdateFuncs(gcp cloud.Cloud) *rnode.UpdateFuncs[compute.Address, alpha.Address, beta.Address] {
	return nil // Does not support generic Update.
}

func (*ops) DeleteFuncs(gcp cloud.Cloud) *rnode.DeleteFuncs[compute.Address, alpha.Address, beta.Address] {
	return &rnode.DeleteFuncs[compute.Address, alpha.Address, beta.Address]{
		GA: rnode.DeleteFuncsByScope[compute.Address]{
			Global:   gcp.GlobalAddresses().Delete,
			Regional: gcp.Addresses().Delete,
		},
		Alpha: rnode.DeleteFuncsByScope[alpha.Address]{
			Global:   gcp.AlphaGlobalAddresses().Delete,
			Regional: gcp.AlphaAddresses().Delete,
		},
		Beta: rnode.DeleteFuncsByScope[beta.Address]{
			Global:   gcp.BetaGlobalAddresses().Delete,
			Regional: gcp.BetaAddresses().Delete,
		},
	}
}
