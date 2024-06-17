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

package plan

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/algo/graphviz"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/address"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/all"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/backendservice"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/forwardingrule"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/healthcheck"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/networkendpointgroup"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/targethttpproxy"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/urlmap"
	"google.golang.org/api/compute/v1"
)

func TestLB(t *testing.T) {
	// TODO: this is not a real test

	b := all.ResourceBuilder{Project: "proj"}

	gr := rgraph.NewBuilder()

	for _, f := range []func() rnode.Builder{
		// Address
		func() rnode.Builder {
			m := b.N("addr").Address().Resource()
			r, _ := m.Freeze()
			return address.NewBuilderWithResource(r)
		},
		// ForwardingRule
		func() rnode.Builder {
			m := b.N("fr").ForwardingRule().Resource()
			m.Access(func(x *compute.ForwardingRule) {
				x.IPAddress = b.N("addr").Address().SelfLink()
				x.Target = b.N("tp").TargetHttpProxy().SelfLink()
			})
			r, _ := m.Freeze()
			return forwardingrule.NewBuilderWithResource(r)
		},
		// TargetHttpProxy
		func() rnode.Builder {
			m := b.N("tp").TargetHttpProxy().Resource()
			m.Access(func(x *compute.TargetHttpProxy) {
				x.UrlMap = b.N("um").UrlMap().SelfLink()
			})
			r, _ := m.Freeze()
			return targethttpproxy.NewBuilderWithResource(r)
		},
		// UrlMap
		func() rnode.Builder {
			m := b.N("um").UrlMap().Resource()
			m.Access(func(x *compute.UrlMap) {
				x.DefaultService = b.N("bs").BackendService().SelfLink()
			})
			r, _ := m.Freeze()
			return urlmap.NewBuilderWithResource(r)
		},
		// BackendService
		func() rnode.Builder {
			m := b.N("bs").BackendService().Resource()
			m.Access(func(x *compute.BackendService) {
				x.Backends = append(x.Backends, &compute.Backend{
					Group: b.N("neg").DefaultZone().NetworkEndpointGroup().SelfLink(),
				})
				x.HealthChecks = []string{
					b.N("hc").HealthCheck().SelfLink(),
				}
			})
			r, _ := m.Freeze()
			return backendservice.NewBuilderWithResource(r)
		},
		// HealthCheck
		func() rnode.Builder {
			m := b.N("hc").HealthCheck().Resource()
			r, _ := m.Freeze()
			return healthcheck.NewBuilderWithResource(r)
		},
		// NEG
		func() rnode.Builder {
			m := b.N("neg").DefaultZone().NetworkEndpointGroup().Resource()
			r, _ := m.Freeze()
			return networkendpointgroup.NewBuilderWithResource(r)
		},
	} {
		b := f()
		b.SetOwnership(rnode.OwnershipManaged)
		b.SetState(rnode.NodeExists)
		gr.Add(b)
	}

	mock := cloud.NewMockGCE(&cloud.SingleProjectRouter{ID: b.Project})

	mock.HealthChecks().Insert(context.Background(), meta.GlobalKey("hc"), &compute.HealthCheck{})
	mock.BackendServices().Insert(context.Background(), meta.GlobalKey("bs"), &compute.BackendService{Description: "blahblah"})
	mock.TargetHttpProxies().Insert(context.Background(), meta.GlobalKey("tp"), &compute.TargetHttpProxy{
		UrlMap: b.N("umx").UrlMap().SelfLink(),
	})

	mock.UrlMaps().Insert(context.Background(), meta.GlobalKey("umx"), &compute.UrlMap{})

	want, err := gr.Build()
	if err != nil {
		t.Fatalf("Build() = %v, want nil", err)
	}

	res, err := Do(context.Background(), mock, want)
	if err != nil {
		t.Fatalf("Do() = %v, want nil", err)
	}

	var viz exec.GraphvizTracer
	ex, err := exec.NewSerialExecutor(nil, res.Actions, exec.DryRunOption(true), exec.TracerOption(&viz))
	if err != nil {
		t.Fatalf("NewSerialExecutor() = %v, want nil", err)
	}
	execResult, err := ex.Run(context.TODO())
	for _, p := range execResult.Pending {
		t.Logf("%+v", p.Metadata())
	}
	//t.Error(err)
	//t.Error(viz.String())

	t.Log(err)
	t.Log(viz.String())
	t.Log(execResult)
	t.Logf("got: %s", graphviz.Do(res.Got))
	t.Logf("want: %s", graphviz.Do(res.Want))
}
