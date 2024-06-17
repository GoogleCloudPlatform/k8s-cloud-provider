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
	"context"
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/api"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/fake"
	"github.com/google/go-cmp/cmp"
	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	"google.golang.org/api/compute/v1"
)

const (
	proj           = "proj-1"
	hcSelfLink     = "https://www.googleapis.com/compute/v1/projects/proj-1/global/healthChecks/hcName"
	fingerprintStr = "abcds"
)

func TestBackendServiceSchema(t *testing.T) {
	key := meta.GlobalKey("key-1")
	x := NewMutableBackendService(proj, key)
	if err := x.CheckSchema(); err != nil {
		t.Fatalf("CheckSchema() = %v, want nil", err)
	}
}

func createBackendServiceNode(name string, setFun func(x MutableBackendService) error) (*backendServiceNode, error) {
	bsID := ID(proj, meta.GlobalKey(name))
	bsMutResource := NewMutableBackendService(proj, bsID.Key)
	err := setFun(bsMutResource)
	if err != nil {
		return nil, fmt.Errorf("setFun(_) = %v, want nil", err)
	}
	// set fingerprint for update action
	bsMutResource.Access(func(x *compute.BackendService) {
		x.Fingerprint = fingerprintStr
	})
	bsResource, err := bsMutResource.Freeze()
	if err != nil {
		return nil, fmt.Errorf("bsMutResource.Freeze() = %v, want nil", err)
	}

	bsBuilder := NewBuilder(bsID)
	bsBuilder.SetOwnership(rnode.OwnershipManaged)
	bsBuilder.SetState(rnode.NodeExists)
	bsBuilder.SetResource(bsResource)
	bsNode, err := bsBuilder.Build()
	if err != nil {
		return nil, fmt.Errorf("bsBuilder.Build() = %v, want nil", err)
	}
	gotNode := bsNode.(*backendServiceNode)
	return gotNode, nil
}

func createBackendServiceResource(t *testing.T, bsID *cloud.ResourceID, modifyFun func(x MutableBackendService) error) rnode.UntypedResource {
	bsMutResource := NewMutableBackendService(proj, bsID.Key)
	err := bsMutResource.Access(func(x *compute.BackendService) {
		x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
		x.Protocol = "TCP"
		x.Port = 80
		x.CompressionMode = "DISABLED"
		x.ConnectionDraining = &compute.ConnectionDraining{}
		x.SessionAffinity = "NONE"
		x.TimeoutSec = 30
	})
	if err != nil {
		t.Fatalf("setFun(_) = %v, want nil", err)
	}
	if modifyFun != nil {
		modifyFun(bsMutResource)
	}
	bsResource, err := bsMutResource.Freeze()
	if err != nil {
		t.Fatalf("bsMutResource.Freeze() = %v, want nil", err)
	}
	return bsResource
}

func TestActionUpdate(t *testing.T) {
	for _, tc := range []struct {
		desc           string
		setUpFn        func(m MutableBackendService) error
		wantGAError    bool
		wantAlphaError bool
		wantBetaError  bool
	}{
		{
			desc: "ga",
			setUpFn: func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.CompressionMode = "DISABLED"
					x.ConnectionDraining = &compute.ConnectionDraining{}
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
				})
			},
		},
		{
			desc: "alpha",
			setUpFn: func(m MutableBackendService) error {
				return m.AccessAlpha(func(x *alpha.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &alpha.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.IpAddressSelectionPolicy = "NONE"
					x.VpcNetworkScope = "IPV6"
					x.ExternalManagedMigrationState = "TEST"
					x.SecuritySettings = &alpha.SecuritySettings{
						Authentication:  "abcd",
						SubjectAltNames: []string{"name"},
					}
				})
			},
			wantGAError:   true,
			wantBetaError: true,
		},
		{
			desc: "beta",
			setUpFn: func(m MutableBackendService) error {
				return m.AccessBeta(func(x *beta.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &beta.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.IpAddressSelectionPolicy = "NONE"
				})
			},
			wantGAError: true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {

			gotNode, err := createBackendServiceNode("bs-name", tc.setUpFn)
			if err != nil {
				t.Fatalf("createBackendServiceNode(bs-name, _) = %v, want nil", err)
			}
			gotBs := gotNode.Resource().(BackendService)

			_, gaErr := gotBs.ToGA()
			gotGAError := gaErr != nil
			if gotGAError != tc.wantGAError {
				t.Errorf("gotBs.ToGA() = %v, got %v want %v", gaErr, gotGAError, tc.wantGAError)
			}
			_, alphaErr := gotBs.ToAlpha()
			gotAlphaError := alphaErr != nil
			if gotAlphaError != tc.wantAlphaError {
				t.Errorf("gotBs.ToAlpha() = %v, got %v want %v", alphaErr, gotAlphaError, tc.wantAlphaError)
			}
			_, betaErr := gotBs.ToBeta()
			gotBetaError := betaErr != nil
			if gotBetaError != tc.wantBetaError {
				t.Errorf("gotBs.ToBeta() = %v, got %v want %v", betaErr, gotBetaError, tc.wantBetaError)
			}

			fingerprint, err := fingerprint(gotNode)
			if err != nil {
				t.Fatalf("fingerprint(_) = %v, want nil", err)
			}
			actions, err := rnode.UpdateActions[compute.BackendService, alpha.BackendService, beta.BackendService](&ops{}, gotNode, gotNode, gotNode.resource, fingerprint)
			if err != nil {
				t.Fatalf("rnode.UpdateActions[]() = %v, want nil", err)
			}
			if len(actions) == 0 {
				t.Fatalf("no actions to update")
			}
			a := actions[0]
			mockCloud := cloud.NewMockGCE(&cloud.SingleProjectRouter{ID: proj})
			updateHook := func(ctx context.Context, key *meta.Key, bs *compute.BackendService, m *cloud.MockBackendServices, o ...cloud.Option) error {
				if bs.Fingerprint != fingerprint {
					t.Fatalf("Update BackendService Hook: fingerprint mismatch got: %s, want %s", bs.Fingerprint, fingerprint)
				}
				return nil
			}
			mockCloud.MockBackendServices.UpdateHook = updateHook
			_, err = a.Run(context.Background(), mockCloud)
			if err != nil {
				t.Fatalf("a.Run(_, mockCloud) = %v, want nil", err)
			}
		})
	}
}

func TestBackendServiceDiff(t *testing.T) {
	bsName := "bs-name"
	for _, tc := range []struct {
		desc         string
		setUpFn      func(m MutableBackendService) error
		updateFn     func(m MutableBackendService) error
		expectedOp   rnode.Operation
		expectedDiff bool
	}{
		{
			desc:       "no diff ga",
			expectedOp: rnode.OpNothing,
			setUpFn: func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &compute.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
				})
			},
			updateFn: func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &compute.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
				})
			},
		},
		{
			desc:         "expected recreation on internal schema change",
			expectedOp:   rnode.OpRecreate,
			expectedDiff: true,
			setUpFn: func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &compute.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
				})
			},
			updateFn: func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.LoadBalancingScheme = "EXTERNAL"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &compute.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
				})
			},
		},
		{
			desc:         "expected recreation on network change",
			expectedOp:   rnode.OpRecreate,
			expectedDiff: true,
			setUpFn: func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &compute.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
				})
			},
			updateFn: func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &compute.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "new-net"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
				})
			},
		},
		{
			desc:         "expected update on port change",
			expectedOp:   rnode.OpUpdate,
			expectedDiff: true,
			setUpFn: func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &compute.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
				})
			},
			updateFn: func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 100
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &compute.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
				})
			},
		},
		{
			desc:       "no diff beta",
			expectedOp: rnode.OpNothing,
			setUpFn: func(m MutableBackendService) error {
				return m.AccessBeta(func(x *beta.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &beta.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.IpAddressSelectionPolicy = "NONE"
				})
			},
			updateFn: func(m MutableBackendService) error {
				return m.AccessBeta(func(x *beta.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &beta.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.IpAddressSelectionPolicy = "NONE"
				})
			},
		},
		{
			desc:         "expected update for beta",
			expectedOp:   rnode.OpUpdate,
			expectedDiff: true,
			setUpFn: func(m MutableBackendService) error {
				return m.AccessBeta(func(x *beta.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &beta.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.IpAddressSelectionPolicy = "NONE"
					x.SecuritySettings = &beta.SecuritySettings{
						Authentication: "NONE",
					}
				})
			},
			updateFn: func(m MutableBackendService) error {
				return m.AccessBeta(func(x *beta.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &beta.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.IpAddressSelectionPolicy = "NONE"
					x.SecuritySettings = &beta.SecuritySettings{
						Authentication: "abcd",
					}
				})
			},
		},
		{
			desc:         "expected update between beta and ga",
			expectedOp:   rnode.OpUpdate,
			expectedDiff: true,
			setUpFn: func(m MutableBackendService) error {
				return m.AccessBeta(func(x *beta.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &beta.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.IpAddressSelectionPolicy = "NONE"
					x.SecuritySettings = &beta.SecuritySettings{
						Authentication: "NONE",
					}
				})
			},
			updateFn: func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &compute.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.SecuritySettings = &compute.SecuritySettings{
						ClientTlsPolicy: "abcd",
					}
				})
			},
		},
		{
			desc:         "expected update between ga and beta",
			expectedOp:   rnode.OpUpdate,
			expectedDiff: true,
			setUpFn: func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &compute.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
				})
			},
			updateFn: func(m MutableBackendService) error {
				return m.AccessBeta(func(x *beta.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &beta.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.IpAddressSelectionPolicy = "NONE"
					x.SecuritySettings = &beta.SecuritySettings{
						Authentication: "abcd",
					}
				})
			},
		},
		{
			desc:       "no diff alpha",
			expectedOp: rnode.OpNothing,
			setUpFn: func(m MutableBackendService) error {
				return m.AccessAlpha(func(x *alpha.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &alpha.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.IpAddressSelectionPolicy = "NONE"
					x.VpcNetworkScope = "IPV6"
					x.ExternalManagedMigrationState = "TEST"
					x.SecuritySettings = &alpha.SecuritySettings{
						Authentication:  "abcd",
						SubjectAltNames: []string{"name"},
					}
				})
			},
			updateFn: func(m MutableBackendService) error {
				return m.AccessAlpha(func(x *alpha.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &alpha.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.IpAddressSelectionPolicy = "NONE"
					x.VpcNetworkScope = "IPV6"
					x.ExternalManagedMigrationState = "TEST"
					x.SecuritySettings = &alpha.SecuritySettings{
						Authentication:  "abcd",
						SubjectAltNames: []string{"name"},
					}
				})
			},
		},
		{
			desc:         "expected update for alpha",
			expectedOp:   rnode.OpUpdate,
			expectedDiff: true,
			setUpFn: func(m MutableBackendService) error {
				return m.AccessAlpha(func(x *alpha.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &alpha.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.IpAddressSelectionPolicy = "NONE"
					x.VpcNetworkScope = "IPV6"
					x.ExternalManagedMigrationState = "TEST"
					x.SecuritySettings = &alpha.SecuritySettings{
						Authentication:  "abcd",
						SubjectAltNames: []string{"name"},
					}
				})
			},
			updateFn: func(m MutableBackendService) error {
				return m.AccessAlpha(func(x *alpha.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &alpha.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.IpAddressSelectionPolicy = "NONE"
					x.VpcNetworkScope = "IPV6"
					x.ExternalManagedMigrationState = "NONE"
					x.SecuritySettings = &alpha.SecuritySettings{
						Authentication:  "abcd",
						SubjectAltNames: []string{"name"},
					}
				})
			},
		},
		{
			desc:         "expected update between ga and alpha",
			expectedOp:   rnode.OpUpdate,
			expectedDiff: true,
			setUpFn: func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &compute.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
				})
			},
			updateFn: func(m MutableBackendService) error {
				return m.AccessAlpha(func(x *alpha.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &alpha.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.IpAddressSelectionPolicy = "NONE"
					x.VpcNetworkScope = "IPV6"
					x.ExternalManagedMigrationState = "TEST"
					x.SecuritySettings = &alpha.SecuritySettings{
						Authentication:  "abcd",
						SubjectAltNames: []string{"name"},
					}
				})
			},
		},
		{
			desc:         "expected update between  alpha and ga",
			expectedOp:   rnode.OpUpdate,
			expectedDiff: true,
			setUpFn: func(m MutableBackendService) error {
				return m.AccessAlpha(func(x *alpha.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &alpha.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
					x.IpAddressSelectionPolicy = "NONE"
					x.VpcNetworkScope = "IPV6"
					x.ExternalManagedMigrationState = "TEST"
					x.SecuritySettings = &alpha.SecuritySettings{
						Authentication:  "abcd",
						SubjectAltNames: []string{"name"},
					}
				})
			},
			updateFn: func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
					x.Protocol = "TCP"
					x.Port = 80
					x.HealthChecks = []string{hcSelfLink}
					x.ConnectionDraining = &compute.ConnectionDraining{}
					x.CompressionMode = "DISABLED"
					x.Network = "default"
					x.SessionAffinity = "NONE"
					x.TimeoutSec = 30
				})
			},
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {

			gotNode, err := createBackendServiceNode(bsName, tc.setUpFn)
			if err != nil {
				t.Fatalf("createBackendServiceNode(%s, _) = %v, want nil", bsName, err)
			}
			wantBS, err := createBackendServiceNode(bsName, tc.updateFn)
			if err != nil {
				t.Fatalf("createBackendServiceNode(%s, _) = %v, want nil", bsName, err)
			}
			plan, err := gotNode.Diff(wantBS)
			if err != nil || plan == nil {
				t.Fatalf("gotNode.Diff(_) = (%v, %v), want plan,  nil", plan, err)
			}
			if plan.Operation != tc.expectedOp {
				t.Errorf("%v != %v", plan.Operation, tc.expectedOp)
			}

			if tc.expectedDiff && (plan.Diff == nil || len(plan.Diff.Items) == 0) {
				t.Errorf("Result did not returned diff")
			}
			t.Logf("Diff results %+v", plan)
		})
	}
}

func TestBackendServiceDiffError(t *testing.T) {
	bsName := "bs-name"
	setUpFn := func(m MutableBackendService) error {
		return m.Access(func(x *compute.BackendService) {
			x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
			x.Protocol = "TCP"
			x.Port = 100
			x.HealthChecks = []string{hcSelfLink}
			x.ConnectionDraining = &compute.ConnectionDraining{}
			x.CompressionMode = "DISABLED"
			x.Network = "default"
			x.SessionAffinity = "NONE"
			x.TimeoutSec = 30
		})
	}
	bsNode, err := createBackendServiceNode(bsName, setUpFn)
	if err != nil {
		t.Fatalf("createBackendServiceNode(%s, _) = %v, want nil", bsName, err)
	}
	fakeId := ID(proj, meta.GlobalKey(bsName))
	fakeBuilder := fake.NewBuilder(fakeId)
	fakeRes := fake.NewMutableFake(proj, fakeId.Key)
	res, err := fakeRes.Freeze()
	fakeBuilder.SetResource(res)
	fakeNode, err := fakeBuilder.Build()

	_, err = bsNode.Diff(fakeNode)
	if err == nil {
		t.Fatal("wantNode.Diff(fakeNode) = nil, want error")
	}
}

func TestGAFields(t *testing.T) {
	bsID := ID(proj, meta.GlobalKey("bs-test"))
	bsMutResource := NewMutableBackendService(proj, bsID.Key)
	err := bsMutResource.Access(func(x *compute.BackendService) {
		x.HealthChecks = []string{hcSelfLink}
		x.Network = "default"
		x.SessionAffinity = "NONE"
		x.TimeoutSec = 3
	})
	// expect error no all required fields are set
	if err == nil {
		t.Fatal("bsMutResource.Access(_) = nil, want error")
	}
	err = bsMutResource.Access(func(x *compute.BackendService) {
		x.Protocol = "TCP"
		x.Port = 80
		x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
		x.ConnectionDraining = &compute.ConnectionDraining{}
		x.CompressionMode = "DISABLED"
	})
	if err != nil {
		t.Fatalf("bsMutResource.Access(_) = %v, want nil", err)
	}
	// Check that Output Only field should not be set
	err = bsMutResource.Access(func(x *compute.BackendService) {
		x.Kind = "some kind"
	})
	if err == nil {
		t.Fatalf("bsMutResource.Access(_) = %v, want nil", err)
	}
	err = bsMutResource.Access(func(x *compute.BackendService) {
		x.Kind = ""
		x.Fingerprint = fingerprintStr
	})
	bsResource, err := bsMutResource.Freeze()
	if err != nil {
		t.Fatalf("bsMutResource.Freeze() = %v, want nil", err)
	}
	bsBuilder := NewBuilderWithResource(bsResource)
	_, err = bsBuilder.Build()
	if err != nil {
		t.Fatalf("bsBuilder.Build() = %v, want nil", err)
	}
	outRefs, err := bsBuilder.OutRefs()
	if len(outRefs) == 0 {
		t.Fatalf("Out refs length mismatch got:%v, want: >0 ", len(outRefs))
	}
}
func TestAlphaFields(t *testing.T) {
	bsID := ID(proj, meta.GlobalKey("bs-test"))
	bsMutResource := NewMutableBackendService(proj, bsID.Key)
	err := bsMutResource.Access(func(x *compute.BackendService) {
		x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
		x.Protocol = "TCP"
		x.Port = 80
		x.HealthChecks = []string{hcSelfLink}
		x.ConnectionDraining = &compute.ConnectionDraining{}
		x.CompressionMode = "DISABLED"
		x.Network = "default"
		x.SessionAffinity = "NONE"
		x.TimeoutSec = 30
	})
	if err != nil {
		t.Fatalf("bsMutResource.Access(_) = %v, want nil", err)
	}
	err = bsMutResource.AccessAlpha(func(x *alpha.BackendService) {
		x.Subsetting = &alpha.Subsetting{Policy: "NONE"}
		x.ExternalManagedMigrationTestingRate = 10
		x.IpAddressSelectionPolicy = "IPV6_ONLY"
	})
	// error expected not all Alpha RequiredFields are set
	if err == nil {
		t.Fatalf("bsMutResource.AccessAlpha(_) = %v, want error", err)
	}
	err = bsMutResource.AccessAlpha(func(x *alpha.BackendService) {
		x.ExternalManagedMigrationState = "FINALIZE"
		x.VpcNetworkScope = "GLOBAL_VPC_NETWORK"
	})
	if err != nil {
		t.Fatalf("bsMutResource.AccessAlpha(_) = %v, want nil", err)
	}

	bsMutResource.AccessAlpha(func(x *alpha.BackendService) {
		x.Fingerprint = fingerprintStr
	})
	bsResource, err := bsMutResource.Freeze()
	if err != nil {
		t.Fatalf("bsMutResource.Freeze() = %v, want nil", err)
	}
	_, err = bsResource.ToGA()
	if err == nil {
		t.Fatalf("bsResource.ToGA() = %v, want error", err)
	}
	_, err = bsResource.ToBeta()
	if err == nil {
		t.Fatalf("bsResource.ToBeta() = %v, want error", err)
	}
	_, err = bsResource.ToAlpha()
	if err != nil {
		t.Fatalf("bsResource.ToBeta() = %v, want nil", err)
	}
	bsBuilder := NewBuilder(bsID)
	bsBuilder.SetResource(bsResource)
	bsNode, err := bsBuilder.Build()
	if err != nil {
		t.Fatalf("bsBuilder.Build() = %v, want nil", err)
	}
	gotNode := bsNode.(*backendServiceNode)
	gotFingerprint, err := fingerprint(gotNode)
	if err != nil {
		t.Fatalf("fingerprint(_) = %v, want nil", err)
	}
	if gotFingerprint != fingerprintStr {
		t.Fatalf("Fingerprint mismatch got: %s want: %s", gotFingerprint, fingerprintStr)
	}
}
func TestBetaFields(t *testing.T) {
	bsID := ID(proj, meta.GlobalKey("bs-test"))
	bsMutResource := NewMutableBackendService(proj, bsID.Key)
	err := bsMutResource.Access(func(x *compute.BackendService) {
		x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
		x.Protocol = "TCP"
		x.Port = 80
		x.HealthChecks = []string{hcSelfLink}
		x.ConnectionDraining = &compute.ConnectionDraining{}
		x.CompressionMode = "DISABLED"
		x.Network = "default"
		x.SessionAffinity = "NONE"
		x.TimeoutSec = 30
	})
	if err != nil {
		t.Fatalf("bsMutResource.Access(_) = %v, want nil", err)
	}
	err = bsMutResource.AccessBeta(func(x *beta.BackendService) {
		x.Subsetting = &beta.Subsetting{Policy: "NONE"}
	})
	// error expected not all Beta RequiredFields are set
	if err == nil {
		t.Fatalf("bsMutResource.AccessBeta(_) = %v, want error", err)
	}
	err = bsMutResource.AccessBeta(func(x *beta.BackendService) {
		x.IpAddressSelectionPolicy = "IPV6_ONLY"
	})
	if err != nil {
		t.Fatalf("bsMutResource.AccessBeta(_) = %v, want nil", err)
	}
	bsMutResource.AccessBeta(func(x *beta.BackendService) {
		x.Fingerprint = fingerprintStr
	})
	bsResource, err := bsMutResource.Freeze()
	if err != nil {
		t.Fatalf("bsMutResource.Freeze() = %v, want nil", err)
	}
	_, err = bsResource.ToGA()
	if err == nil {
		t.Fatalf("bsResource.ToGA() = %v, want error", err)
	}
	_, err = bsResource.ToBeta()
	if err != nil {
		t.Fatalf("bsResource.ToBeta() = %v, want nil", err)
	}

	bsBuilder := NewBuilderWithResource(bsResource)
	bsNode, err := bsBuilder.Build()
	if err != nil {
		t.Fatalf("bsBuilder.Build() = %v, want nil", err)
	}
	gotNode := bsNode.(*backendServiceNode)
	gotFingerprint, err := fingerprint(gotNode)
	if err != nil {
		t.Fatalf("fingerprint(_) = %v, want nil", err)
	}
	if gotFingerprint != fingerprintStr {
		t.Fatalf("Fingerprint mismatch got: %s want: %s", gotFingerprint, fingerprintStr)
	}
}

func TestBackendServiceActions(t *testing.T) {
	setUpResource := func(m MutableBackendService) error {
		return m.Access(func(x *compute.BackendService) {
			x.LoadBalancingScheme = "INTERNAL_SELF_MANAGED"
			x.Protocol = "TCP"
			x.Port = 80
			x.HealthChecks = []string{hcSelfLink}
			x.CompressionMode = "DISABLED"
			x.ConnectionDraining = &compute.ConnectionDraining{}
			x.SessionAffinity = "NONE"
			x.TimeoutSec = 30
		})
	}

	n1, err := createBackendServiceNode("bs-name", setUpResource)
	if err != nil {
		t.Fatalf("createBackendServiceNode(bs-name, _) = %v, want nil", err)
	}

	for _, tc := range []struct {
		desc    string
		op      rnode.Operation
		wantErr bool
		want    []exec.ActionType
	}{
		{
			desc: "create action",
			op:   rnode.OpCreate,
			want: []exec.ActionType{exec.ActionTypeCreate},
		},
		{
			desc: "delete action",
			op:   rnode.OpDelete,
			want: []exec.ActionType{exec.ActionTypeDelete},
		},
		{
			desc: "recreate action",
			op:   rnode.OpRecreate,
			want: []exec.ActionType{exec.ActionTypeDelete, exec.ActionTypeCreate},
		},
		{
			desc: "no action",
			op:   rnode.OpNothing,
			want: []exec.ActionType{exec.ActionTypeMeta},
		},
		{
			desc: "update action",
			op:   rnode.OpUpdate,
			want: []exec.ActionType{exec.ActionTypeUpdate},
		},
		{
			desc:    "default",
			op:      rnode.OpUnknown,
			wantErr: true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			b := n1.Builder()
			b.SetResource(n1.resource)
			n2, err := b.Build()
			if err != nil {
				t.Fatalf("b.Build() = %v, want nil", err)
			}

			n1.Plan().Set(rnode.PlanDetails{
				Operation: tc.op,
				Why:       "test plan",
			})
			actions, err := n1.Actions(n2)
			isError := (err != nil)
			if tc.wantErr != isError {
				t.Fatalf("n.Actions(_) =%v got error %v, want %v", err, tc.wantErr, isError)
			}
			if tc.wantErr {
				return
			}
			if err != nil {
				t.Fatalf("n.Actions(_) = %v, want nil", err)
			}
			if len(actions) != len(tc.want) {
				t.Fatalf("n.Actions(%q) returned list with elements %d want %d", tc.op, len(actions), len(tc.want))
			}
			for i, a := range actions {
				if a.Metadata().Type != tc.want[i] {
					t.Errorf("Actions mismatch: got: %s, want: %s", a.Metadata().Name, tc.want[i])
				}
			}
		})
	}
}

func TestOutRefs(t *testing.T) {
	bsID := ID(proj, meta.GlobalKey("bs-test"))
	hcID := &cloud.ResourceID{
		Resource:  "healthChecks",
		APIGroup:  meta.APIGroupCompute,
		ProjectID: proj,
		Key:       meta.GlobalKey("hc-name"),
	}
	negID := &cloud.ResourceID{
		Resource:  "networkEndpointGroups",
		APIGroup:  meta.APIGroupCompute,
		ProjectID: proj,
		Key:       meta.GlobalKey("hc-name"),
	}
	espID := &cloud.ResourceID{
		Resource:  "edgeSecurityPolicy",
		APIGroup:  meta.APIGroupCompute,
		ProjectID: proj,
		Key:       meta.GlobalKey("esp-name"),
	}
	spID := &cloud.ResourceID{
		Resource:  "ecurityPolicy",
		APIGroup:  meta.APIGroupCompute,
		ProjectID: proj,
		Key:       meta.GlobalKey("esp-name"),
	}
	for _, tc := range []struct {
		desc        string
		resource    rnode.UntypedResource
		wantErr     bool
		wantOutRefs []rnode.ResourceRef
	}{
		{
			desc: "nil",
		},
		{
			desc:     "without OutRefs",
			resource: createBackendServiceResource(t, bsID, nil),
		},
		{
			desc: "with health check",
			resource: createBackendServiceResource(t, bsID, func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.HealthChecks = []string{hcID.SelfLink(meta.VersionGA)}
				})
			}),
			wantOutRefs: []rnode.ResourceRef{
				{
					From: bsID,
					Path: api.Path{}.Field("HealthChecks").Index(0),
					To:   hcID,
				},
			},
		},
		{
			desc: "with health check wrong format",
			resource: createBackendServiceResource(t, bsID, func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.HealthChecks = []string{"https://apigroup.googleapis.com/alpha/projects/proj1/global/healthchecks/hcname"}
				})
			}),
			wantErr: true,
		},
		{
			desc: "with backends",
			resource: createBackendServiceResource(t, bsID, func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.Backends = []*compute.Backend{
						{Group: negID.SelfLink(meta.VersionGA)},
					}
				})
			}),
			wantOutRefs: []rnode.ResourceRef{
				{
					From: bsID,
					Path: api.Path{}.Field("Backends").Index(0).Field("Group"),
					To:   negID,
				},
			},
		},
		{
			desc: "with health check wrong format",
			resource: createBackendServiceResource(t, bsID, func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.Backends = []*compute.Backend{
						{Group: "https://apigroup.googleapis.com/alpha/projects/proj1/global/negs/negname"},
					}
				})
			}),
			wantErr: true,
		},
		{
			desc: "with  securityPolicy",
			resource: createBackendServiceResource(t, bsID, func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.SecurityPolicy = spID.SelfLink(meta.VersionGA)
				})
			}),
			wantOutRefs: []rnode.ResourceRef{
				{
					From: bsID,
					Path: api.Path{}.Field("SecurityPolicy"),
					To:   spID,
				},
			},
		},
		{
			desc: "with securityPolicy wrong format",
			resource: createBackendServiceResource(t, bsID, func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.SecurityPolicy = "https://apigroup.googleapis.com/alpha/projects/proj1/global/negs/negname"
				})
			}),
			wantErr: true,
		},
		{
			desc: "with edge securityPolicy",
			resource: createBackendServiceResource(t, bsID, func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.EdgeSecurityPolicy = espID.SelfLink(meta.VersionGA)
				})
			}),
			wantOutRefs: []rnode.ResourceRef{
				{
					From: bsID,
					Path: api.Path{}.Field("EdgeSecurityPolicy"),
					To:   espID,
				},
			},
		},
		{
			desc: "with edge securityPolicy wrong format",
			resource: createBackendServiceResource(t, bsID, func(m MutableBackendService) error {
				return m.Access(func(x *compute.BackendService) {
					x.EdgeSecurityPolicy = "https://apigroup.googleapis.com/alpha/projects/proj1/global/negs/negname"
				})
			}),
			wantErr: true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			bsBuilder := NewBuilder(bsID)
			bsBuilder.SetResource(tc.resource)
			bsBuilder.Build()
			outRefs, err := bsBuilder.OutRefs()
			gotErr := err != nil
			if tc.wantErr != gotErr {
				t.Fatalf("bsBuilder.OutRefs() = %v want error %v, got %v", err, tc.wantErr, gotErr)
			}
			if tc.wantErr {
				return
			}
			if diff := cmp.Diff(outRefs, tc.wantOutRefs); diff != "" {
				t.Fatalf("Out refs mismatch: %s ", diff)

			}
		})
	}
}
