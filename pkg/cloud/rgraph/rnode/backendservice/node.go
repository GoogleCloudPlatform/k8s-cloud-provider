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
	"fmt"
	"strings"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

type backendServiceNode struct {
	rnode.NodeBase
	resource BackendService
}

var _ rnode.Node = (*backendServiceNode)(nil)

func (n *backendServiceNode) Resource() rnode.UntypedResource { return n.resource }

func (n *backendServiceNode) Diff(gotNode rnode.Node) (*rnode.PlanDetails, error) {
	got, ok := gotNode.(*backendServiceNode)
	if !ok {
		return nil, fmt.Errorf("BackendServiceNode: invalid type to Diff: %T", gotNode)
	}
	diff, err := got.resource.Diff(n.resource)
	if err != nil {
		return nil, fmt.Errorf("BackendServiceNode: Diff %w", err)
	}

	if !diff.HasDiff() {
		return &rnode.PlanDetails{
			Operation: rnode.OpNothing,
			Why:       "No diff between got and want",
		}, nil
	}

	var (
		needsRecreate bool
		details       []string
	)

	planRecreate := func(s string, args ...any) {
		details = append(details, fmt.Sprintf(s, args...))
		needsRecreate = true
	}
	planUpdate := func(s string, args ...any) {
		details = append(details, fmt.Sprintf(s, args...))
	}

	for _, delta := range diff.Items {
		// These fields cannot be changed in place and require the
		// resource to be recreated.
		switch {
		case delta.Path.Equal(api.Path{}.Pointer().Field("LoadBalancingScheme")),
			delta.Path.Equal(api.Path{}.Pointer().Field("Network")):
			planRecreate("LoadBalancingScheme change: '%v' -> '%v'", delta.A, delta.B)
		default:
			planUpdate("%s change: '%v' -> '%v'", delta.Path, delta.A, delta.B)
		}
	}

	if needsRecreate {
		return &rnode.PlanDetails{
			Operation: rnode.OpRecreate,
			Why:       "BackendService needs to be recreated: " + strings.Join(details, ", "),
			Diff:      diff,
		}, nil
	}
	return &rnode.PlanDetails{
		Operation: rnode.OpUpdate,
		Why:       "BackendService needs to be updated: " + strings.Join(details, ", "),
		Diff:      diff,
	}, nil
}

func fingerprint(gotNode *backendServiceNode) (string, error) {
	gotRes := gotNode.resource
	switch gotRes.Version() {
	case meta.VersionGA:
		obj, err := gotRes.ToGA()
		if err != nil {
			return "", err
		}
		return obj.Fingerprint, nil
	case meta.VersionAlpha:
		obj, err := gotRes.ToAlpha()
		if err != nil {
			return "", err
		}
		return obj.Fingerprint, nil

	case meta.VersionBeta:
		obj, err := gotRes.ToBeta()
		if err != nil {
			return "", err
		}
		return obj.Fingerprint, nil
	}
	return "", fmt.Errorf("Unsupported backend service resource version %v", gotRes.Version())
}

func (n *backendServiceNode) Actions(got rnode.Node) ([]exec.Action, error) {
	op := n.Plan().Op()

	switch op {
	case rnode.OpCreate:
		return rnode.CreateActions[compute.BackendService, alpha.BackendService, beta.BackendService](&ops{}, n, n.resource)

	case rnode.OpDelete:
		return rnode.DeleteActions[compute.BackendService, alpha.BackendService, beta.BackendService](&ops{}, got, n)

	case rnode.OpNothing:
		return []exec.Action{exec.NewExistsAction(n.ID())}, nil

	case rnode.OpRecreate:
		return rnode.RecreateActions[compute.BackendService, alpha.BackendService, beta.BackendService](&ops{}, got, n, n.resource)

	case rnode.OpUpdate:
		gotNode := got.(*backendServiceNode)
		f, err := fingerprint(gotNode)
		if err != nil {
			return nil, fmt.Errorf("Cannot get fingerprint from BackendService: %w", err)
		}
		return rnode.UpdateActions[compute.BackendService, alpha.BackendService, beta.BackendService](&ops{}, got, n, n.resource, f)
	}

	return nil, fmt.Errorf("BackendServiceNode: invalid plan op %s", op)
}

func (n *backendServiceNode) Builder() rnode.Builder {
	b := &builder{}
	b.Init(n.ID(), n.State(), n.Ownership(), n.resource)
	return b
}

/*
name
string

Name of the resource. Provided by the client when the resource is created. The name must be 1-63 characters long, and comply with RFC1035. Specifically, the name must be 1-63 characters long and match the regular expression [a-z]([-a-z0-9]*[a-z0-9])? which means the first character must be a lowercase letter, and all following characters must be a dash, lowercase letter, or digit, except the last character, which cannot be a dash.

description
string

An optional description of this resource. Provide this property when you create the resource.

selfLink
string

[Output Only] Server-defined URL for the resource.

backends[]
object

The list of backends that serve this BackendService.

backends[].description
string

An optional description of this resource. Provide this property when you create the resource.

backends[].group
string

The fully-qualified URL of an instance group or network endpoint group (NEG) resource. To determine what types of backends a load balancer supports, see the Backend services overview.

You must use the fully-qualified URL (starting with https://www.googleapis.com/) to specify the instance group or NEG. Partial URLs are not supported.

backends[].balancingMode
enum

Specifies how to determine whether the backend of a load balancer can handle additional traffic or is fully loaded. For usage guidelines, see Connection balancing mode.

Backends must use compatible balancing modes. For more information, see Supported balancing modes and target capacity settings and Restrictions and guidance for instance groups.

Note: Currently, if you use the API to configure incompatible balancing modes, the configuration might be accepted even though it has no impact and is ignored. Specifically, Backend.maxUtilization is ignored when Backend.balancingMode is RATE. In the future, this incompatible combination will be rejected.

backends[].maxUtilization
number

Optional parameter to define a target capacity for the UTILIZATION balancing mode. The valid range is [0.0, 1.0].

For usage guidelines, see Utilization balancing mode.

backends[].maxRate
integer

Defines a maximum number of HTTP requests per second (RPS). For usage guidelines, see Rate balancing mode and Utilization balancing mode.

Not available if the backend's balancingMode is CONNECTION.

backends[].maxRatePerInstance
number

Defines a maximum target for requests per second (RPS). For usage guidelines, see Rate balancing mode and Utilization balancing mode.

Not available if the backend's balancingMode is CONNECTION.

backends[].maxRatePerEndpoint
number

Defines a maximum target for requests per second (RPS). For usage guidelines, see Rate balancing mode and Utilization balancing mode.

Not available if the backend's balancingMode is CONNECTION.

backends[].maxConnections
integer

Defines a target maximum number of simultaneous connections. For usage guidelines, see Connection balancing mode and Utilization balancing mode. Not available if the backend's balancingMode is RATE.

backends[].maxConnectionsPerInstance
integer

Defines a target maximum number of simultaneous connections. For usage guidelines, see Connection balancing mode and Utilization balancing mode.

Not available if the backend's balancingMode is RATE.

backends[].maxConnectionsPerEndpoint
integer

Defines a target maximum number of simultaneous connections. For usage guidelines, see Connection balancing mode and Utilization balancing mode.

Not available if the backend's balancingMode is RATE.

backends[].capacityScaler
number

A multiplier applied to the backend's target capacity of its balancing mode. The default value is 1, which means the group serves up to 100% of its configured capacity (depending on balancingMode). A setting of 0 means the group is completely drained, offering 0% of its available capacity. The valid ranges are 0.0 and [0.1,1.0]. You cannot configure a setting larger than 0 and smaller than 0.1. You cannot configure a setting of 0 when there is only one backend attached to the backend service.

Not available with backends that don't support using a balancingMode. This includes backends such as global internet NEGs, regional serverless NEGs, and PSC NEGs.

backends[].failover
boolean

This field designates whether this is a failover backend. More than one failover backend can be configured for a given BackendService.

healthChecks[]
string

The list of URLs to the healthChecks, httpHealthChecks (legacy), or httpsHealthChecks (legacy) resource for health checking this backend service. Not all backend services support legacy health checks. See Load balancer guide. Currently, at most one health check can be specified for each backend service. Backend services with instance group or zonal NEG backends must have a health check. Backend services with internet or serverless NEG backends must not have a health check.

timeoutSec
integer

The backend service timeout has a different meaning depending on the type of load balancer. For more information see, Backend service settings. The default is 30 seconds. The full range of timeout values allowed goes from 1 through 2,147,483,647 seconds.

This value can be overridden in the PathMatcher configuration of the UrlMap that references this backend service.

Not supported when the backend service is referenced by a URL map that is bound to target gRPC proxy that has validateForProxyless field set to true. Instead, use maxStreamDuration.

port
(deprecated)
integer

This item is deprecated!

Deprecated in favor of portName. The TCP port to connect on the backend. The default value is 80. For Internal TCP/UDP Load Balancing and Network Load Balancing, omit port.

protocol
enum

The protocol this BackendService uses to communicate with backends.

Possible values are HTTP, HTTPS, HTTP2, TCP, SSL, UDP or GRPC. depending on the chosen load balancer or Traffic Director configuration. Refer to the documentation for the load balancers or for Traffic Director for more information.

Must be set to GRPC when the backend service is referenced by a URL map that is bound to target gRPC proxy.

fingerprint
string (bytes format)

Fingerprint of this resource. A hash of the contents stored in this object. This field is used in optimistic locking. This field will be ignored when inserting a BackendService. An up-to-date fingerprint must be provided in order to update the BackendService, otherwise the request will fail with error 412 conditionNotMet.

To see the latest fingerprint, make a get() request to retrieve a BackendService.

A base64-encoded string.

portName
string

A named port on a backend instance group representing the port for communication to the backend VMs in that group. The named port must be defined on each backend instance group. This parameter has no meaning if the backends are NEGs. For Internal TCP/UDP Load Balancing and Network Load Balancing, omit portName.

enableCDN
boolean

If true, enables Cloud CDN for the backend service of an external HTTP(S) load balancer.

sessionAffinity
enum

Type of session affinity to use. The default is NONE.

Only NONE and HEADER_FIELD are supported when the backend service is referenced by a URL map that is bound to target gRPC proxy that has validateForProxyless field set to true.

For more details, see: Session Affinity.

affinityCookieTtlSec
integer

Lifetime of cookies in seconds. This setting is applicable to external and internal HTTP(S) load balancers and Traffic Director and requires GENERATED_COOKIE or HTTP_COOKIE session affinity.

If set to 0, the cookie is non-persistent and lasts only until the end of the browser session (or equivalent). The maximum allowed value is two weeks (1,209,600).

Not supported when the backend service is referenced by a URL map that is bound to target gRPC proxy that has validateForProxyless field set to true.

region
string

[Output Only] URL of the region where the regional backend service resides. This field is not applicable to global backend services. You must specify this field as part of the HTTP request URL. It is not settable as a field in the request body.

failoverPolicy
object

Requires at least one backend instance group to be defined as a backup (failover) backend. For load balancers that have configurable failover: Internal TCP/UDP Load Balancing and external TCP/UDP Load Balancing.

failoverPolicy.disableConnectionDrainOnFailover
boolean

This can be set to true only if the protocol is TCP.

The default is false.

failoverPolicy.dropTrafficIfUnhealthy
boolean

If set to true, connections to the load balancer are dropped when all primary and all backup backend VMs are unhealthy.If set to false, connections are distributed among all primary VMs when all primary and all backup backend VMs are unhealthy. For load balancers that have configurable failover: Internal TCP/UDP Load Balancing and external TCP/UDP Load Balancing. The default is false.

failoverPolicy.failoverRatio
number

The value of the field must be in the range [0, 1]. If the value is 0, the load balancer performs a failover when the number of healthy primary VMs equals zero. For all other values, the load balancer performs a failover when the total number of healthy primary VMs is less than this ratio. For load balancers that have configurable failover: Internal TCP/UDP Load Balancing and external TCP/UDP Load Balancing.

loadBalancingScheme
enum

Specifies the load balancer type. A backend service created for one type of load balancer cannot be used with another. For more information, refer to Choosing a load balancer.

connectionDraining
object

connectionDraining.drainingTimeoutSec
integer

Configures a duration timeout for existing requests on a removed backend instance. For supported load balancers and protocols, as described in Enabling connection draining.

iap
object

The configurations for Identity-Aware Proxy on this resource. Not available for Internal TCP/UDP Load Balancing and Network Load Balancing.

iap.enabled
boolean

Whether the serving infrastructure will authenticate and authorize all incoming requests. If true, the oauth2ClientId and oauth2ClientSecret fields must be non-empty.

iap.oauth2ClientId
string

OAuth2 client ID to use for the authentication flow.

iap.oauth2ClientSecret
string

OAuth2 client secret to use for the authentication flow. For security reasons, this value cannot be retrieved via the API. Instead, the SHA-256 hash of the value is returned in the oauth2ClientSecretSha256 field.

@InputOnly

iap.oauth2ClientSecretSha256
string

[Output Only] SHA256 hash value for the field oauth2ClientSecret above.

cdnPolicy
object

Cloud CDN configuration for this BackendService. Only available for specified load balancer types.

cdnPolicy.cacheKeyPolicy
object

The CacheKeyPolicy for this CdnPolicy.

cdnPolicy.cacheKeyPolicy.includeProtocol
boolean

If true, http and https requests will be cached separately.

cdnPolicy.cacheKeyPolicy.includeHost
boolean

If true, requests to different hosts will be cached separately.

cdnPolicy.cacheKeyPolicy.includeQueryString
boolean

If true, include query string parameters in the cache key according to queryStringWhitelist and queryStringBlacklist. If neither is set, the entire query string will be included. If false, the query string will be excluded from the cache key entirely.

cdnPolicy.cacheKeyPolicy.queryStringWhitelist[]
string

Names of query string parameters to include in cache keys. All other parameters will be excluded. Either specify queryStringWhitelist or queryStringBlacklist, not both. '&' and '=' will be percent encoded and not treated as delimiters.

cdnPolicy.cacheKeyPolicy.queryStringBlacklist[]
string

Names of query string parameters to exclude in cache keys. All other parameters will be included. Either specify queryStringWhitelist or queryStringBlacklist, not both. '&' and '=' will be percent encoded and not treated as delimiters.

cdnPolicy.cacheKeyPolicy.includeHttpHeaders[]
string

Allows HTTP request headers (by name) to be used in the cache key.

cdnPolicy.cacheKeyPolicy.includeNamedCookies[]
string

Allows HTTP cookies (by name) to be used in the cache key. The name=value pair will be used in the cache key Cloud CDN generates.

cdnPolicy.signedUrlKeyNames[]
string

[Output Only] Names of the keys for signing request URLs.

cdnPolicy.signedUrlCacheMaxAgeSec
string (int64 format)

Maximum number of seconds the response to a signed URL request will be considered fresh. After this time period, the response will be revalidated before being served. Defaults to 1hr (3600s). When serving responses to signed URL requests, Cloud CDN will internally behave as though all responses from this backend had a "Cache-Control: public, max-age=[TTL]" header, regardless of any existing Cache-Control header. The actual headers served in responses will not be altered.

cdnPolicy.requestCoalescing
boolean

If true then Cloud CDN will combine multiple concurrent cache fill requests into a small number of requests to the origin.

cdnPolicy.cacheMode
enum

Specifies the cache setting for all responses from this backend. The possible values are: USE_ORIGIN_HEADERS Requires the origin to set valid caching headers to cache content. Responses without these headers will not be cached at Google's edge, and will require a full trip to the origin on every request, potentially impacting performance and increasing load on the origin server. FORCE_CACHE_ALL Cache all content, ignoring any "private", "no-store" or "no-cache" directives in Cache-Control response headers. Warning: this may result in Cloud CDN caching private, per-user (user identifiable) content. CACHE_ALL_STATIC Automatically cache static content, including common image formats, media (video and audio), and web assets (JavaScript and CSS). Requests and responses that are marked as uncacheable, as well as dynamic content (including HTML), will not be cached.

cdnPolicy.defaultTtl
integer

Specifies the default TTL for cached content served by this origin for responses that do not have an existing valid TTL (max-age or s-max-age). Setting a TTL of "0" means "always revalidate". The value of defaultTTL cannot be set to a value greater than that of maxTTL, but can be equal. When the cacheMode is set to FORCE_CACHE_ALL, the defaultTTL will overwrite the TTL set in all responses. The maximum allowed value is 31,622,400s (1 year), noting that infrequently accessed objects may be evicted from the cache before the defined TTL.

cdnPolicy.maxTtl
integer

Specifies the maximum allowed TTL for cached content served by this origin. Cache directives that attempt to set a max-age or s-maxage higher than this, or an Expires header more than maxTTL seconds in the future will be capped at the value of maxTTL, as if it were the value of an s-maxage Cache-Control directive. Headers sent to the client will not be modified. Setting a TTL of "0" means "always revalidate". The maximum allowed value is 31,622,400s (1 year), noting that infrequently accessed objects may be evicted from the cache before the defined TTL.

cdnPolicy.clientTtl
integer

Specifies a separate client (e.g. browser client) maximum TTL. This is used to clamp the max-age (or Expires) value sent to the client. With FORCE_CACHE_ALL, the lesser of clientTtl and defaultTtl is used for the response max-age directive, along with a "public" directive. For cacheable content in CACHE_ALL_STATIC mode, clientTtl clamps the max-age from the origin (if specified), or else sets the response max-age directive to the lesser of the clientTtl and defaultTtl, and also ensures a "public" cache-control directive is present. If a client TTL is not specified, a default value (1 hour) will be used. The maximum allowed value is 31,622,400s (1 year).

cdnPolicy.negativeCaching
boolean

Negative caching allows per-status code TTLs to be set, in order to apply fine-grained caching for common errors or redirects. This can reduce the load on your origin and improve end-user experience by reducing response latency. When the cache mode is set to CACHE_ALL_STATIC or USE_ORIGIN_HEADERS, negative caching applies to responses with the specified response code that lack any Cache-Control, Expires, or Pragma: no-cache directives. When the cache mode is set to FORCE_CACHE_ALL, negative caching applies to all responses with the specified response code, and override any caching headers. By default, Cloud CDN will apply the following default TTLs to these status codes: HTTP 300 (Multiple Choice), 301, 308 (Permanent Redirects): 10m HTTP 404 (Not Found), 410 (Gone), 451 (Unavailable For Legal Reasons): 120s HTTP 405 (Method Not Found), 421 (Misdirected Request), 501 (Not Implemented): 60s. These defaults can be overridden in negativeCachingPolicy.

cdnPolicy.negativeCachingPolicy[]
object

Sets a cache TTL for the specified HTTP status code. negativeCaching must be enabled to configure negativeCachingPolicy. Omitting the policy and leaving negativeCaching enabled will use Cloud CDN's default cache TTLs. Note that when specifying an explicit negativeCachingPolicy, you should take care to specify a cache TTL for all response codes that you wish to cache. Cloud CDN will not apply any default negative caching when a policy exists.

cdnPolicy.negativeCachingPolicy[].code
integer

The HTTP status code to define a TTL against. Only HTTP status codes 300, 301, 302, 307, 308, 404, 405, 410, 421, 451 and 501 are can be specified as values, and you cannot specify a status code more than once.

cdnPolicy.negativeCachingPolicy[].ttl
integer

The TTL (in seconds) for which to cache responses with the corresponding status code. The maximum allowed value is 1800s (30 minutes), noting that infrequently accessed objects may be evicted from the cache before the defined TTL.

cdnPolicy.bypassCacheOnRequestHeaders[]
object

Bypass the cache when the specified request headers are matched - e.g. Pragma or Authorization headers. Up to 5 headers can be specified. The cache is bypassed for all cdnPolicy.cacheMode settings.

cdnPolicy.bypassCacheOnRequestHeaders[].headerName
string

The header field name to match on when bypassing cache. Values are case-insensitive.

cdnPolicy.serveWhileStale
integer

Serve existing content from the cache (if available) when revalidating content with the origin, or when an error is encountered when refreshing the cache. This setting defines the default "max-stale" duration for any cached responses that do not specify a max-stale directive. Stale responses that exceed the TTL configured here will not be served. The default limit (max-stale) is 86400s (1 day), which will allow stale content to be served up to this limit beyond the max-age (or s-max-age) of a cached response. The maximum allowed value is 604800 (1 week). Set this to zero (0) to disable serve-while-stale.

customRequestHeaders[]
string

Headers that the load balancer adds to proxied requests. See Creating custom headers.

customResponseHeaders[]
string

Headers that the load balancer adds to proxied responses. See Creating custom headers.

securityPolicy
string

[Output Only] The resource URL for the security policy associated with this backend service.

edgeSecurityPolicy
string

[Output Only] The resource URL for the edge security policy associated with this backend service.

logConfig
object

This field denotes the logging options for the load balancer traffic served by this backend service. If logging is enabled, logs will be exported to Stackdriver.

logConfig.enable
boolean

Denotes whether to enable logging for the load balancer traffic served by this backend service. The default value is false.

logConfig.sampleRate
number

This field can only be specified if logging is enabled for this backend service. The value of the field must be in [0, 1]. This configures the sampling rate of requests to the load balancer where 1.0 means all logged requests are reported and 0.0 means no logged requests are reported. The default value is 1.0.

logConfig.optionalMode
enum

This field can only be specified if logging is enabled for this backend service. Configures whether all, none or a subset of optional fields should be added to the reported logs. One of [INCLUDE_ALL_OPTIONAL, EXCLUDE_ALL_OPTIONAL, CUSTOM]. Default is EXCLUDE_ALL_OPTIONAL.

logConfig.optionalFields[]
string

This field can only be specified if logging is enabled for this backend service and "logConfig.optionalMode" was set to CUSTOM. Contains a list of optional fields you want to include in the logs. For example: serverInstance, serverGkeDetails.cluster, serverGkeDetails.pod.podNamespace

securitySettings
object

This field specifies the security settings that apply to this backend service. This field is applicable to a global backend service with the loadBalancingScheme set to INTERNAL_SELF_MANAGED.

securitySettings.clientTlsPolicy
string

Optional. A URL referring to a networksecurity.ClientTlsPolicy resource that describes how clients should authenticate with this service's backends.

clientTlsPolicy only applies to a global BackendService with the loadBalancingScheme set to INTERNAL_SELF_MANAGED.

If left blank, communications are not encrypted.

Note: This field currently has no impact.

securitySettings.subjectAltNames[]
string

Optional. A list of Subject Alternative Names (SANs) that the client verifies during a mutual TLS handshake with an server/endpoint for this BackendService. When the server presents its X.509 certificate to the client, the client inspects the certificate's subjectAltName field. If the field contains one of the specified values, the communication continues. Otherwise, it fails. This additional check enables the client to verify that the server is authorized to run the requested service.

Note that the contents of the server certificate's subjectAltName field are configured by the Public Key Infrastructure which provisions server identities.

Only applies to a global BackendService with loadBalancingScheme set to INTERNAL_SELF_MANAGED. Only applies when BackendService has an attached clientTlsPolicy with clientCertificate (mTLS mode).

Note: This field currently has no impact.

localityLbPolicy
enum

The load balancing algorithm used within the scope of the locality. The possible values are:

ROUND_ROBIN: This is a simple policy in which each healthy backend is selected in round robin order. This is the default.
LEAST_REQUEST: An O(1) algorithm which selects two random healthy hosts and picks the host which has fewer active requests.
RING_HASH: The ring/modulo hash load balancer implements consistent hashing to backends. The algorithm has the property that the addition/removal of a host from a set of N hosts only affects 1/N of the requests.
RANDOM: The load balancer selects a random healthy host.
ORIGINAL_DESTINATION: Backend host is selected based on the client connection metadata, i.e., connections are opened to the same address as the destination address of the incoming connection before the connection was redirected to the load balancer.
MAGLEV: used as a drop in replacement for the ring hash load balancer. Maglev is not as stable as ring hash but has faster table lookup build times and host selection times. For more information about Maglev, see https://ai.google/research/pubs/pub44824
This field is applicable to either:

A regional backend service with the serviceProtocol set to HTTP, HTTPS, or HTTP2, and loadBalancingScheme set to INTERNAL_MANAGED.
A global backend service with the loadBalancingScheme set to INTERNAL_SELF_MANAGED.
If sessionAffinity is not NONE, and this field is not set to MAGLEV or RING_HASH, session affinity settings will not take effect.

Only ROUND_ROBIN and RING_HASH are supported when the backend service is referenced by a URL map that is bound to target gRPC proxy that has validateForProxyless field set to true.

consistentHash
object

Consistent Hash-based load balancing can be used to provide soft session affinity based on HTTP headers, cookies or other properties. This load balancing policy is applicable only for HTTP connections. The affinity to a particular destination host will be lost when one or more hosts are added/removed from the destination service. This field specifies parameters that control consistent hashing. This field is only applicable when localityLbPolicy is set to MAGLEV or RING_HASH.

This field is applicable to either:

A regional backend service with the serviceProtocol set to HTTP, HTTPS, or HTTP2, and loadBalancingScheme set to INTERNAL_MANAGED.
A global backend service with the loadBalancingScheme set to INTERNAL_SELF_MANAGED.
consistentHash.httpCookie
object

Hash is based on HTTP Cookie. This field describes a HTTP cookie that will be used as the hash key for the consistent hash load balancer. If the cookie is not present, it will be generated. This field is applicable if the sessionAffinity is set to HTTP_COOKIE.

Not supported when the backend service is referenced by a URL map that is bound to target gRPC proxy that has validateForProxyless field set to true.

consistentHash.httpCookie.name
string

Name of the cookie.

consistentHash.httpCookie.path
string

Path to set for the cookie.

consistentHash.httpCookie.ttl
object

Lifetime of the cookie.

consistentHash.httpCookie.ttl.seconds
string (int64 format)

Span of time at a resolution of a second. Must be from 0 to 315,576,000,000 inclusive. Note: these bounds are computed from: 60 sec/min * 60 min/hr * 24 hr/day * 365.25 days/year * 10000 years

consistentHash.httpCookie.ttl.nanos
integer

Span of time that's a fraction of a second at nanosecond resolution. Durations less than one second are represented with a 0 seconds field and a positive nanos field. Must be from 0 to 999,999,999 inclusive.

consistentHash.httpHeaderName
string

The hash based on the value of the specified header field. This field is applicable if the sessionAffinity is set to HEADER_FIELD.

consistentHash.minimumRingSize
string (int64 format)

The minimum number of virtual nodes to use for the hash ring. Defaults to 1024. Larger ring sizes result in more granular load distributions. If the number of hosts in the load balancing pool is larger than the ring size, each host will be assigned a single virtual node.

circuitBreakers
object

circuitBreakers.maxRequestsPerConnection
integer

Maximum requests for a single connection to the backend service. This parameter is respected by both the HTTP/1.1 and HTTP/2 implementations. If not specified, there is no limit. Setting this parameter to 1 will effectively disable keep alive.

Not supported when the backend service is referenced by a URL map that is bound to target gRPC proxy that has validateForProxyless field set to true.

circuitBreakers.maxConnections
integer

The maximum number of connections to the backend service. If not specified, there is no limit.

Not supported when the backend service is referenced by a URL map that is bound to target gRPC proxy that has validateForProxyless field set to true.

circuitBreakers.maxPendingRequests
integer

The maximum number of pending requests allowed to the backend service. If not specified, there is no limit.

Not supported when the backend service is referenced by a URL map that is bound to target gRPC proxy that has validateForProxyless field set to true.

circuitBreakers.maxRequests
integer

The maximum number of parallel requests that allowed to the backend service. If not specified, there is no limit.

circuitBreakers.maxRetries
integer

The maximum number of parallel retries allowed to the backend cluster. If not specified, the default is 1.

Not supported when the backend service is referenced by a URL map that is bound to target gRPC proxy that has validateForProxyless field set to true.

outlierDetection
object

Settings controlling the eviction of unhealthy hosts from the load balancing pool for the backend service. If not set, this feature is considered disabled.

This field is applicable to either:

A regional backend service with the serviceProtocol set to HTTP, HTTPS, HTTP2, or GRPC, and loadBalancingScheme set to INTERNAL_MANAGED.
A global backend service with the loadBalancingScheme set to INTERNAL_SELF_MANAGED.
outlierDetection.consecutiveErrors
integer

Number of errors before a host is ejected from the connection pool. When the backend host is accessed over HTTP, a 5xx return code qualifies as an error. Defaults to 5.

Not supported when the backend service is referenced by a URL map that is bound to target gRPC proxy that has validateForProxyless field set to true.

outlierDetection.interval
object

Time interval between ejection analysis sweeps. This can result in both new ejections as well as hosts being returned to service. Defaults to 1 second.

outlierDetection.interval.seconds
string (int64 format)

Span of time at a resolution of a second. Must be from 0 to 315,576,000,000 inclusive. Note: these bounds are computed from: 60 sec/min * 60 min/hr * 24 hr/day * 365.25 days/year * 10000 years

outlierDetection.interval.nanos
integer

Span of time that's a fraction of a second at nanosecond resolution. Durations less than one second are represented with a 0 seconds field and a positive nanos field. Must be from 0 to 999,999,999 inclusive.

outlierDetection.baseEjectionTime
object

The base time that a host is ejected for. The real ejection time is equal to the base ejection time multiplied by the number of times the host has been ejected. Defaults to 30000ms or 30s.

outlierDetection.baseEjectionTime.seconds
string (int64 format)

Span of time at a resolution of a second. Must be from 0 to 315,576,000,000 inclusive. Note: these bounds are computed from: 60 sec/min * 60 min/hr * 24 hr/day * 365.25 days/year * 10000 years

outlierDetection.baseEjectionTime.nanos
integer

Span of time that's a fraction of a second at nanosecond resolution. Durations less than one second are represented with a 0 seconds field and a positive nanos field. Must be from 0 to 999,999,999 inclusive.

outlierDetection.maxEjectionPercent
integer

Maximum percentage of hosts in the load balancing pool for the backend service that can be ejected. Defaults to 50%.

outlierDetection.enforcingConsecutiveErrors
integer

The percentage chance that a host will be actually ejected when an outlier status is detected through consecutive 5xx. This setting can be used to disable ejection or to ramp it up slowly. Defaults to 0.

Not supported when the backend service is referenced by a URL map that is bound to target gRPC proxy that has validateForProxyless field set to true.

outlierDetection.enforcingSuccessRate
integer

The percentage chance that a host will be actually ejected when an outlier status is detected through success rate statistics. This setting can be used to disable ejection or to ramp it up slowly. Defaults to 100.

outlierDetection.successRateMinimumHosts
integer

The number of hosts in a cluster that must have enough request volume to detect success rate outliers. If the number of hosts is less than this setting, outlier detection via success rate statistics is not performed for any host in the cluster. Defaults to 5.

outlierDetection.successRateRequestVolume
integer

The minimum number of total requests that must be collected in one interval (as defined by the interval duration above) to include this host in success rate based outlier detection. If the volume is lower than this setting, outlier detection via success rate statistics is not performed for that host. Defaults to 100.

outlierDetection.successRateStdevFactor
integer

This factor is used to determine the ejection threshold for success rate outlier ejection. The ejection threshold is the difference between the mean success rate, and the product of this factor and the standard deviation of the mean success rate: mean - (stdev * successRateStdevFactor). This factor is divided by a thousand to get a double. That is, if the desired factor is 1.9, the runtime value should be 1900. Defaults to 1900.

outlierDetection.consecutiveGatewayFailure
integer

The number of consecutive gateway failures (502, 503, 504 status or connection errors that are mapped to one of those status codes) before a consecutive gateway failure ejection occurs. Defaults to 3.

Not supported when the backend service is referenced by a URL map that is bound to target gRPC proxy that has validateForProxyless field set to true.

outlierDetection.enforcingConsecutiveGatewayFailure
integer

The percentage chance that a host will be actually ejected when an outlier status is detected through consecutive gateway failures. This setting can be used to disable ejection or to ramp it up slowly. Defaults to 100.

Not supported when the backend service is referenced by a URL map that is bound to target gRPC proxy that has validateForProxyless field set to true.

network
string

The URL of the network to which this backend service belongs. This field can only be specified when the load balancing scheme is set to INTERNAL.

subsetting
object

subsetting.policy
enum

connectionTrackingPolicy
object

Connection Tracking configuration for this BackendService. Connection tracking policy settings are only available for Network Load Balancing and Internal TCP/UDP Load Balancing.

connectionTrackingPolicy.trackingMode
enum

Specifies the key used for connection tracking. There are two options:

PER_CONNECTION: This is the default mode. The Connection Tracking is performed as per the Connection Key (default Hash Method) for the specific protocol.
PER_SESSION: The Connection Tracking is performed as per the configured Session Affinity. It matches the configured Session Affinity.
For more details, see Tracking Mode for Network Load Balancing and Tracking Mode for Internal TCP/UDP Load Balancing.

connectionTrackingPolicy.connectionPersistenceOnUnhealthyBackends
enum

Specifies connection persistence when backends are unhealthy. The default value is DEFAULT_FOR_PROTOCOL.

If set to DEFAULT_FOR_PROTOCOL, the existing connections persist on unhealthy backends only for connection-oriented protocols (TCP and SCTP) and only if the Tracking Mode is PER_CONNECTION (default tracking mode) or the Session Affinity is configured for 5-tuple. They do not persist for UDP.

If set to NEVER_PERSIST, after a backend becomes unhealthy, the existing connections on the unhealthy backend are never persisted on the unhealthy backend. They are always diverted to newly selected healthy backends (unless all backends are unhealthy).

If set to ALWAYS_PERSIST, existing connections always persist on unhealthy backends regardless of protocol and session affinity. It is generally not recommended to use this mode overriding the default.

For more details, see Connection Persistence for Network Load Balancing and Connection Persistence for Internal TCP/UDP Load Balancing.

connectionTrackingPolicy.idleTimeoutSec
integer

Specifies how long to keep a Connection Tracking entry while there is no matching traffic (in seconds).

For Internal TCP/UDP Load Balancing:

The minimum (default) is 10 minutes and the maximum is 16 hours.
It can be set only if Connection Tracking is less than 5-tuple (i.e. Session Affinity is CLIENT_IP_NO_DESTINATION, CLIENT_IP or CLIENT_IP_PROTO, and Tracking Mode is PER_SESSION).
For Network Load Balancer the default is 60 seconds. This option is not available publicly.

connectionTrackingPolicy.enableStrongAffinity
boolean

Enable Strong Session Affinity for Network Load Balancing. This option is not available publicly.

maxStreamDuration
object

Specifies the default maximum duration (timeout) for streams to this service. Duration is computed from the beginning of the stream until the response has been completely processed, including all retries. A stream that does not complete in this duration is closed.

If not specified, there will be no timeout limit, i.e. the maximum duration is infinite.

This value can be overridden in the PathMatcher configuration of the UrlMap that references this backend service.

This field is only allowed when the loadBalancingScheme of the backend service is INTERNAL_SELF_MANAGED.

maxStreamDuration.seconds
string (int64 format)

Span of time at a resolution of a second. Must be from 0 to 315,576,000,000 inclusive. Note: these bounds are computed from: 60 sec/min * 60 min/hr * 24 hr/day * 365.25 days/year * 10000 years

maxStreamDuration.nanos
integer

Span of time that's a fraction of a second at nanosecond resolution. Durations less than one second are represented with a 0 seconds field and a positive nanos field. Must be from 0 to 999,999,999 inclusive.

compressionMode
enum

Compress text responses using Brotli or gzip compression, based on the client's Accept-Encoding header.

serviceBindings[]
string

URLs of networkservices.ServiceBinding resources.

Can only be set if load balancing scheme is INTERNAL_SELF_MANAGED. If set, lists of backends and health checks must be both empty.

localityLbPolicies[]
object

A list of locality load-balancing policies to be used in order of preference. When you use localityLbPolicies, you must set at least one value for either the localityLbPolicies[].policy or the localityLbPolicies[].customPolicy field. localityLbPolicies overrides any value set in the localityLbPolicy field.

For an example of how to use this field, see Define a list of preferred policies.

Caution: This field and its children are intended for use in a service mesh that includes gRPC clients only. Envoy proxies can't use backend services that have this configuration.

localityLbPolicies[].policy
object

localityLbPolicies[].policy.name
enum

The name of a locality load-balancing policy. Valid values include ROUND_ROBIN and, for Java clients, LEAST_REQUEST. For information about these values, see the description of localityLbPolicy.

Do not specify the same policy more than once for a backend. If you do, the configuration is rejected.

localityLbPolicies[].customPolicy
object

localityLbPolicies[].customPolicy.name
string

Identifies the custom policy.

The value should match the name of a custom implementation registered on the gRPC clients. It should follow protocol buffer message naming conventions and include the full path (for example, myorg.CustomLbPolicy). The maximum length is 256 characters.

Do not specify the same custom policy more than once for a backend. If you do, the configuration is rejected.

For an example of how to use this field, see Use a custom policy.

localityLbPolicies[].customPolicy.data
string

An optional, arbitrary JSON object with configuration data, understood by a locally installed custom policy implementation.
*/
