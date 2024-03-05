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

package tcproute

import (
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"google.golang.org/api/networkservices/v1"
	beta "google.golang.org/api/networkservices/v1beta1"
)

type tcpRouteOps struct{}

func (*tcpRouteOps) GetFuncs(gcp cloud.Cloud) *rnode.GetFuncs[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute] {
	return &rnode.GetFuncs[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute]{
		GA: rnode.GetFuncsByScope[networkservices.TcpRoute]{
			Global: gcp.TcpRoutes().Get,
		},
		Beta: rnode.GetFuncsByScope[beta.TcpRoute]{
			Global: gcp.BetaTcpRoutes().Get,
		},
	}
}

func (*tcpRouteOps) CreateFuncs(gcp cloud.Cloud) *rnode.CreateFuncs[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute] {
	return &rnode.CreateFuncs[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute]{
		GA: rnode.CreateFuncsByScope[networkservices.TcpRoute]{
			Global: gcp.TcpRoutes().Insert,
		},
		Beta: rnode.CreateFuncsByScope[beta.TcpRoute]{
			Global: gcp.BetaTcpRoutes().Insert,
		},
	}
}

func (*tcpRouteOps) UpdateFuncs(gcp cloud.Cloud) *rnode.UpdateFuncs[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute] {
	return &rnode.UpdateFuncs[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute]{
		GA: rnode.UpdateFuncsByScope[networkservices.TcpRoute]{
			Global: gcp.TcpRoutes().Patch,
		},
		Beta: rnode.UpdateFuncsByScope[beta.TcpRoute]{
			Global: gcp.BetaTcpRoutes().Patch,
		},
		Options: rnode.UpdateFuncsNoFingerprint,
	}
}

func (*tcpRouteOps) DeleteFuncs(gcp cloud.Cloud) *rnode.DeleteFuncs[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute] {
	return &rnode.DeleteFuncs[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute]{
		GA: rnode.DeleteFuncsByScope[networkservices.TcpRoute]{
			Global: gcp.TcpRoutes().Delete,
		},
		Beta: rnode.DeleteFuncsByScope[beta.TcpRoute]{
			Global: gcp.BetaTcpRoutes().Delete,
		},
	}
}
