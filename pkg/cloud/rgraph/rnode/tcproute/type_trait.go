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
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"google.golang.org/api/networkservices/v1"
	beta "google.golang.org/api/networkservices/v1beta1"
)

// https://cloud.google.com/traffic-director/docs/reference/network-services/rest/v1beta1/projects.locations.tcpRoutes
type tcpRouteTypeTrait struct {
	api.BaseTypeTrait[networkservices.TcpRoute, api.PlaceholderType, beta.TcpRoute]
}

func (*tcpRouteTypeTrait) FieldTraits(meta.Version) *api.FieldTraits {
	dt := api.NewFieldTraits()
	dt.OutputOnly(api.Path{}.Pointer().Field("SelfLink"))
	dt.OutputOnly(api.Path{}.Pointer().Field("CreateTime"))
	dt.OutputOnly(api.Path{}.Pointer().Field("UpdateTime"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("Gateways"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Labels"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Meshes"))
	// TODO(kl52752) fix Required values for nested struct fields when parent field is optional.
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Rules").AnySliceIndex().Pointer().Field("Matches"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Rules").AnySliceIndex().Pointer().Field("Action").Pointer().Field("Destinations").AnySliceIndex().Pointer().Field("Weight"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Rules").AnySliceIndex().Pointer().Field("Action").Pointer().Field("Destinations"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Rules").AnySliceIndex().Pointer().Field("Action").Pointer().Field("OriginalDestination"))

	// For TcpRoute "Name" field is marked as System field to avoid updating TcpRoute every time the graph is being processed.
	// The updates are consequence of different name formatting in rgraph and in the cloud.
	dt.System(api.Path{}.Pointer().Field("Name"))
	return dt
}
