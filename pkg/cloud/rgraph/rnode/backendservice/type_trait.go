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
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

// https://cloud.google.com/compute/docs/reference/rest/v1/backendServices
type typeTrait struct {
	api.BaseTypeTrait[compute.BackendService, alpha.BackendService, beta.BackendService]
}

func (*typeTrait) FieldTraits(v meta.Version) *api.FieldTraits {
	dt := api.NewFieldTraits()
	// Built-ins
	dt.OutputOnly(api.Path{}.Pointer().Field("Fingerprint"))

	// [Output Only]
	dt.OutputOnly(api.Path{}.Pointer().Field("CreationTimestamp"))
	dt.OutputOnly(api.Path{}.Pointer().Field("EdgeSecurityPolicy"))
	dt.OutputOnly(api.Path{}.Pointer().Field("Id"))
	dt.OutputOnly(api.Path{}.Pointer().Field("Kind"))
	dt.OutputOnly(api.Path{}.Pointer().Field("Region"))
	dt.OutputOnly(api.Path{}.Pointer().Field("SecurityPolicy"))
	dt.OutputOnly(api.Path{}.Pointer().Field("SelfLink"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("Iap"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Iap").Pointer().Field("Enabled"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Iap").Pointer().Field("Oauth2ClientId"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Iap").Pointer().Field("Oauth2ClientSecret"))
	dt.OutputOnly(api.Path{}.Pointer().Field("Iap").Pointer().Field("Oauth2ClientSecretSha256"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("Port"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("PortName"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("ServiceBindings"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("ServiceLbPolicy"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Subsetting"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("UsedBy"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Network"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy"))
	dt.OutputOnly(api.Path{}.Pointer().Field("CdnPolicy").Field("SignedUrlKeyNames"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("BypassCacheOnRequestHeaders"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("BypassCacheOnRequestHeaders").AnySliceIndex().Pointer().Field("HeaderName"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("IncludeHost"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("IncludeHttpHeaders"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("IncludeNamedCookies"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("IncludeProtocol"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("IncludeQueryString"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("QueryStringBlacklist"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("QueryStringWhitelist"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("DefaultTtl"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("MaxTtl"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("NegativeCachingPolicy"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("NegativeCachingPolicy").AnySliceIndex().Pointer().Field("Ttl"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("RequestCoalescing"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("ServeWhileStale"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("SignedUrlCacheMaxAgeSec"))
	dt.OutputOnly(api.Path{}.Pointer().Field("CdnPolicy").Pointer().Field("CacheKeyPolicy").Pointer().Field("SignedUrlKeyNames"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("CircuitBreakers"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CircuitBreakers").Pointer().Field("MaxConnections"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CircuitBreakers").Pointer().Field("MaxPendingRequests"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CircuitBreakers").Pointer().Field("MaxRequests"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CircuitBreakers").Pointer().Field("MaxRequestsPerConnection"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("CircuitBreakers").Pointer().Field("MaxRetries"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("ConnectionDraining").Pointer().Field("DrainingTimeoutSec"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("ConnectionTrackingPolicy"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("ConnectionTrackingPolicy").Pointer().Field("EnableStrongAffinity"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("ConsistentHash"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("ConsistentHash").Pointer().Field("HttpCookie"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("ConsistentHash").Pointer().Field("HttpCookie").Pointer().Field("Path"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("ConsistentHash").Pointer().Field("HttpCookie").Pointer().Field("Ttl"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("ConsistentHash").Pointer().Field("HttpCookie").Pointer().Field("Ttl").Pointer().Field("Nanos"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("ConsistentHash").Pointer().Field("HttpCookie").Pointer().Field("Ttl").Pointer().Field("Seconds"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("ConsistentHash").Pointer().Field("HttpHeaderName"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("CustomRequestHeaders"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Description"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("EnableCDN"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("FailoverPolicy"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("FailoverPolicy").Pointer().Field("DisableConnectionDrainOnFailover"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("FailoverPolicy").Pointer().Field("DropTrafficIfUnhealthy"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("FailoverPolicy").Pointer().Field("FailoverRatio"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("HealthChecks"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("CustomResponseHeaders"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("AffinityCookieTtlSec"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Backends"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Backends").AnySliceIndex().Pointer().Field("CapacityScaler"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Backends").AnySliceIndex().Pointer().Field("Failover"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Backends").AnySliceIndex().Pointer().Field("MaxConnections"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Backends").AnySliceIndex().Pointer().Field("MaxConnectionsPerEndpoint"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Backends").AnySliceIndex().Pointer().Field("MaxConnectionsPerInstance"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Backends").AnySliceIndex().Pointer().Field("MaxRate"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Backends").AnySliceIndex().Pointer().Field("MaxRatePerEndpoint"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Backends").AnySliceIndex().Pointer().Field("MaxRatePerInstance"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("Backends").AnySliceIndex().Pointer().Field("MaxUtilization"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("LocalityLbPolicies"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("LocalityLbPolicies").AnySliceIndex().Pointer().Field("CustomPolicy"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("LocalityLbPolicies").AnySliceIndex().Pointer().Field("CustomPolicy").Pointer().Field("Data"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("LocalityLbPolicies").AnySliceIndex().Pointer().Field("Policy"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("LocalityLbPolicy"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("LogConfig"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("LogConfig").Pointer().Field("Enable"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("LogConfig").Pointer().Field("OptionalFields"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("LogConfig").Pointer().Field("SampleRate"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("MaxStreamDuration"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("MaxStreamDuration").Pointer().Field("Nanos"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("MaxStreamDuration").Pointer().Field("Seconds"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("Metadatas"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("OutlierDetection"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("OutlierDetection").Pointer().Field("EnforcingConsecutiveErrors"))

	dt.AllowZeroValue(api.Path{}.Pointer().Field("SecuritySettings"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("SecuritySettings").Pointer().Field("AwsV4Authentication"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("SecuritySettings").Pointer().Field("AwsV4Authentication").Pointer().Field("AccessKeyId"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("SecuritySettings").Pointer().Field("AwsV4Authentication").Pointer().Field("AccessKeyVersion"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("SecuritySettings").Pointer().Field("ClientTlsPolicy"))
	dt.AllowZeroValue(api.Path{}.Pointer().Field("SecuritySettings").Pointer().Field("SubjectAltNames"))

	if v == meta.VersionBeta || v == meta.VersionAlpha {
		dt.AllowZeroValue(api.Path{}.Pointer().Field("SecuritySettings").Pointer().Field("Authentication"))
		dt.AllowZeroValue(api.Path{}.Pointer().Field("CircuitBreakers").Pointer().Field("ConnectTimeout"))
		dt.AllowZeroValue(api.Path{}.Pointer().Field("CircuitBreakers").Pointer().Field("ConnectTimeout").Pointer().Field("Nanos"))
		dt.AllowZeroValue(api.Path{}.Pointer().Field("CircuitBreakers").Pointer().Field("ConnectTimeout").Pointer().Field("Seconds"))
		dt.AllowZeroValue(api.Path{}.Pointer().Field("Subsetting").Pointer().Field("SubsetSize"))

	}
	if v == meta.VersionAlpha {
		dt.AllowZeroValue(api.Path{}.Pointer().Field("SecuritySettings").Pointer().Field("AuthenticationPolicy"))
		dt.AllowZeroValue(api.Path{}.Pointer().Field("SecuritySettings").Pointer().Field("AuthorizationConfig"))
		dt.AllowZeroValue(api.Path{}.Pointer().Field("SecuritySettings").Pointer().Field("ClientTlsSettings"))
		dt.AllowZeroValue(api.Path{}.Pointer().Field("SecuritySettings").Pointer().Field("ClientTlsSettings").Pointer().Field("ClientTlsContext"))
		dt.AllowZeroValue(api.Path{}.Pointer().Field("SecuritySettings").Pointer().Field("ClientTlsSettings").Pointer().Field("Sni"))
		dt.AllowZeroValue(api.Path{}.Pointer().Field("SecuritySettings").Pointer().Field("ClientTlsSettings").Pointer().Field("SubjectAltNames"))
		dt.AllowZeroValue(api.Path{}.Pointer().Field("SecuritySettings").Pointer().Field("SubjectAltNames"))

		dt.AllowZeroValue(api.Path{}.Pointer().Field("ExternalManagedMigrationTestingRate"))
		dt.OutputOnly(api.Path{}.Pointer().Field("SelfLinkWithId"))

		// not supported
		dt.OutputOnly(api.Path{}.Pointer().Field("HaPolicy"))

		dt.AllowZeroValue(api.Path{}.Pointer().Field("Iap").Pointer().Field("Oauth2ClientInfo"))
		dt.AllowZeroValue(api.Path{}.Pointer().Field("Iap").Pointer().Field("Oauth2ClientInfo").Pointer().Field("ApplicationName"))
		dt.AllowZeroValue(api.Path{}.Pointer().Field("Iap").Pointer().Field("Oauth2ClientInfo").Pointer().Field("ClientName"))
		dt.AllowZeroValue(api.Path{}.Pointer().Field("Iap").Pointer().Field("Oauth2ClientInfo").Pointer().Field("DeveloperEmailAddress"))
	}
	return dt
}
