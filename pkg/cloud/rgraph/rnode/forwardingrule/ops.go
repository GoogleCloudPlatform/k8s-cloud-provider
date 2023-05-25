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

package forwardingrule

import (
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

type ops struct{}

func (*ops) GetFuncs(gcp cloud.Cloud) *rnode.GetFuncs[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule] {
	return &rnode.GetFuncs[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule]{
		GA: rnode.GetFuncsByScope[compute.ForwardingRule]{
			Global:   gcp.GlobalForwardingRules().Get,
			Regional: gcp.ForwardingRules().Get,
		},
		Alpha: rnode.GetFuncsByScope[alpha.ForwardingRule]{
			Global:   gcp.AlphaGlobalForwardingRules().Get,
			Regional: gcp.AlphaForwardingRules().Get,
		},
		Beta: rnode.GetFuncsByScope[beta.ForwardingRule]{
			Global:   gcp.BetaGlobalForwardingRules().Get,
			Regional: gcp.BetaForwardingRules().Get,
		},
	}
}

func (*ops) CreateFuncs(gcp cloud.Cloud) *rnode.CreateFuncs[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule] {
	return &rnode.CreateFuncs[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule]{
		GA: rnode.CreateFuncsByScope[compute.ForwardingRule]{
			Global:   gcp.GlobalForwardingRules().Insert,
			Regional: gcp.ForwardingRules().Insert,
		},
		Alpha: rnode.CreateFuncsByScope[alpha.ForwardingRule]{
			Global:   gcp.AlphaGlobalForwardingRules().Insert,
			Regional: gcp.AlphaForwardingRules().Insert,
		},
		Beta: rnode.CreateFuncsByScope[beta.ForwardingRule]{
			Global:   gcp.BetaGlobalForwardingRules().Insert,
			Regional: gcp.BetaForwardingRules().Insert,
		},
	}
}

func (*ops) UpdateFuncs(cloud.Cloud) *rnode.UpdateFuncs[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule] {
	return nil // Does not support generic Update.
}

func (*ops) DeleteFuncs(gcp cloud.Cloud) *rnode.DeleteFuncs[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule] {
	return &rnode.DeleteFuncs[compute.ForwardingRule, alpha.ForwardingRule, beta.ForwardingRule]{
		GA: rnode.DeleteFuncsByScope[compute.ForwardingRule]{
			Global:   gcp.GlobalForwardingRules().Delete,
			Regional: gcp.ForwardingRules().Delete,
		},
		Alpha: rnode.DeleteFuncsByScope[alpha.ForwardingRule]{
			Global:   gcp.AlphaGlobalForwardingRules().Delete,
			Regional: gcp.AlphaForwardingRules().Delete,
		},
		Beta: rnode.DeleteFuncsByScope[beta.ForwardingRule]{
			Global:   gcp.BetaGlobalForwardingRules().Delete,
			Regional: gcp.BetaForwardingRules().Delete,
		},
	}
}
