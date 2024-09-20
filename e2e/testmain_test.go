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
	"log"
	"os"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

var (
	// theCloud is a global to be used in the e2e tests.
	theCloud cloud.Cloud
)

func resourceName(name string) string {
	return TestFlags.ResourcePrefix + runID + "-" + name
}

func TestMain(m *testing.M) {
	ParseFlagsOrDie()

	ctx := context.Background()

	credentials, err := google.FindDefaultCredentials(ctx, compute.ComputeScope)
	if err != nil {
		log.Fatal(err)
	}
	ts := credentials.TokenSource

	// Optionally, impersonate service account by replacing token source for http client.
	if TestFlags.ServiceAccountName != "" {
		ts, err = impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
			TargetPrincipal: TestFlags.ServiceAccountName,
			Scopes:          []string{compute.ComputeScope, compute.CloudPlatformScope},
		}, option.WithCredentials(credentials))
		if err != nil {
			log.Fatalf("Failed to use %q credentials: %v", TestFlags.ServiceAccountName, err)
		}
	}
	client := oauth2.NewClient(ctx, ts)

	mrl := &cloud.MinimumRateLimiter{RateLimiter: &cloud.NopRateLimiter{}, Minimum: 50 * time.Millisecond}
	crl := cloud.NewCompositeRateLimiter(mrl)

	// The default limit is 1500 per minute. Leave 200 buffer.
	computeRL := cloud.NewTickerRateLimiter(1300, time.Minute)
	crl.Register("HealthChecks", "", computeRL)
	crl.Register("BackendServices", "", computeRL)
	crl.Register("NetworkEndpointGroups", "", computeRL)

	// The default limit is 1200 per minute. Leave 200 buffer.
	networkServicesRL := cloud.NewTickerRateLimiter(1000, time.Minute)
	crl.Register("TcpRoutes", "", networkServicesRL)
	crl.Register("Meshes", "", networkServicesRL)

	// To ensure minimum time between operations, wrap the network services rate limiter.
	orl := &cloud.MinimumRateLimiter{RateLimiter: networkServicesRL, Minimum: 100 * time.Millisecond}
	crl.Register("Operations", "", orl)

	svc, err := cloud.NewService(ctx, client, &cloud.SingleProjectRouter{ID: TestFlags.Project}, crl)
	if err != nil {
		log.Fatal(err)
	}
	theCloud = cloud.NewGCE(svc)

	code := m.Run()
	FallbackCleanup(ctx)
	os.Exit(code)
}

func checkErrCode(t *testing.T, err error, wantCode int, fmtStr string, args ...interface{}) {
	t.Helper()

	gerr, ok := err.(*googleapi.Error)
	if !ok {
		t.Fatalf("%s: invalid error type, want *googleapi.Error, got %T", fmt.Sprintf(fmtStr, args...), err)
	}
	if gerr.Code != wantCode {
		t.Fatalf("%s: got code %d, want %d (err: %v)", fmt.Sprintf(fmtStr, args...), gerr.Code, wantCode, err)
	}
}
