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
package e2e

import (
	"context"
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/kr/pretty"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/networkservices/v1"
)

func TestTcpRoute(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	bs := &compute.BackendService{
		Name:                resourceName("bs1"),
		Backends:            []*compute.Backend{},
		LoadBalancingScheme: "INTERNAL_SELF_MANAGED",
	}
	bsKey := meta.GlobalKey(bs.Name)

	t.Cleanup(func() {
		err := theCloud.BackendServices().Delete(ctx, bsKey)
		t.Logf("bs delete: %v", err)
	})

	// TcpRoute needs a BackendService to point to.
	err := theCloud.BackendServices().Insert(ctx, bsKey, bs)
	t.Logf("bs insert: %v", err)
	if err != nil {
		t.Fatal(err)
	}

	// Current API does not support the new URL scheme.
	serviceName := fmt.Sprintf("https://compute.googleapis.com/v1/projects/%s/global/backendServices/%s", testFlags.project, bs.Name)
	tcpr := &networkservices.TcpRoute{
		Name: resourceName("route1"),
		Rules: []*networkservices.TcpRouteRouteRule{
			{
				Action: &networkservices.TcpRouteRouteAction{
					Destinations: []*networkservices.TcpRouteRouteDestination{
						{ServiceName: serviceName},
					},
				},
			},
		},
	}
	t.Logf("tcpr = %s", pretty.Sprint(tcpr))
	tcprKey := meta.GlobalKey(tcpr.Name)

	// Insert
	t.Cleanup(func() {
		err := theCloud.TcpRoutes().Delete(ctx, tcprKey)
		t.Logf("tcpRoute delete: %v", err)
	})

	err = theCloud.TcpRoutes().Insert(ctx, tcprKey, tcpr)
	t.Logf("tcproutes insert: %v", err)
	if err != nil {
		t.Fatalf("Insert() = %v", err)
	}

	// Get
	tcpRoute, err := theCloud.TcpRoutes().Get(ctx, tcprKey)
	t.Logf("tcpRoute = %s", pretty.Sprint(tcpRoute))
	if err != nil {
		t.Fatalf("Get(%s) = %v", tcprKey, err)
	}

	if len(tcpRoute.Rules) < 1 || len(tcpRoute.Rules[0].Action.Destinations) < 1 {
		t.Fatalf("gotTcpRoute = %s, need at least one destination", pretty.Sprint(tcpRoute))
	}
	gotServiceName := tcpRoute.Rules[0].Action.Destinations[0].ServiceName
	if gotServiceName != serviceName {
		t.Fatalf("gotTcpRoute = %s, gotServiceName = %q, want %q", pretty.Sprint(tcpRoute), gotServiceName, serviceName)
	}
}
