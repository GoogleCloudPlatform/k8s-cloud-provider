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
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

// Default TypeTrait for HealthCheck.
//
// https://cloud.google.com/compute/docs/reference/rest/v1/HealthChecks
type TypeTrait struct {
	api.BaseTypeTrait[compute.HealthCheck, alpha.HealthCheck, beta.HealthCheck]
}

func (*TypeTrait) FieldTraits(v meta.Version) *api.FieldTraits {
	dt := api.NewFieldTraits()
	// [Output Only]
	dt.OutputOnly(api.Path{}.Pointer().Field("CreationTimestamp"))
	dt.OutputOnly(api.Path{}.Pointer().Field("Id"))
	dt.OutputOnly(api.Path{}.Pointer().Field("Kind"))
	dt.OutputOnly(api.Path{}.Pointer().Field("Region"))
	dt.OutputOnly(api.Path{}.Pointer().Field("SelfLink"))

	// This field is not supported
	dt.OutputOnly(api.Path{}.Pointer().Field("GrpcHealthCheck").Pointer().Field("PortName"))
	dt.OutputOnly(api.Path{}.Pointer().Field("Http2HealthCheck").Pointer().Field("PortName"))
	dt.OutputOnly(api.Path{}.Pointer().Field("HttpHealthCheck").Pointer().Field("PortName"))
	dt.OutputOnly(api.Path{}.Pointer().Field("SslHealthCheck").Pointer().Field("PortName"))
	dt.OutputOnly(api.Path{}.Pointer().Field("HttpsHealthCheck").Pointer().Field("PortName"))

	// required fields
	dt.NonZeroValue(api.Path{}.Pointer().Field("HealthyThreshold"))
	dt.NonZeroValue(api.Path{}.Pointer().Field("UnhealthyThreshold"))
	dt.NonZeroValue(api.Path{}.Pointer().Field("CheckIntervalSec"))
	dt.NonZeroValue(api.Path{}.Pointer().Field("TimeoutSec"))
	dt.NonZeroValue(api.Path{}.Pointer().Field("Type"))

	if v == meta.VersionAlpha {
		dt.OutputOnly(api.Path{}.Pointer().Field("SelfLinkWithId"))
		dt.OutputOnly(api.Path{}.Pointer().Field("UdpHealthCheck").Pointer().Field("PortName"))
	}

	return dt
}
