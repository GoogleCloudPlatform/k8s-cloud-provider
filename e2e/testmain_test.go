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
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"golang.org/x/oauth2/google"
	"k8s.io/klog/v2"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

var (
	// theCloud is a global to be used in the e2e tests.
	theCloud cloud.Cloud
	// testFlags passed in from the command line.
	testFlags = struct {
		project            string
		resourcePrefix     string
		boskosResourceType string
		inProw             bool
	}{
		project:            "",
		resourcePrefix:     "k8scp-",
		boskosResourceType: "gke-internal-project",
		inProw:             false,
	}
	runID string
)

func init() {
	klog.InitFlags(flag.CommandLine)

	flag.BoolVar(&testFlags.inProw, "run-in-prow", testFlags.inProw, "is the test running in PROW")
	flag.StringVar(&testFlags.project, "project", testFlags.project, "GCP project ID. Only valid when run-in-prow is false.")
	flag.StringVar(&testFlags.resourcePrefix, "resourcePrefix", testFlags.resourcePrefix, "Prefix used to name all resources created in the tests. Any resources with this prefix will be removed during cleanup.")
	flag.StringVar(&testFlags.boskosResourceType, "boskos-resource-type", testFlags.boskosResourceType, "name of the boskos resource type to reserve. Only valid when run-in-prow is true")

	runID = fmt.Sprintf("%0x", rand.Int63()&0xffff)
}

func parseFlagsOrDie() {
	flag.Parse()

	if !testFlags.inProw && testFlags.project == "" {
		fmt.Println("-project must be set for test not run in prow")
		os.Exit(1)
	}
}

func resourceName(name string) string {
	return testFlags.resourcePrefix + runID + "-" + name
}

func setEnvProject(project string) error {
	if out, err := exec.Command("gcloud", "config", "set", "project", project).CombinedOutput(); err != nil {
		return fmt.Errorf("SetEnvProject(%q) failed: %q: %w", project, out, err)
	}

	return os.Setenv("PROJECT", project)
}

func TestMain(m *testing.M) {
	parseFlagsOrDie()

	if testFlags.inProw {
		ph, err := newProjectHolder()
		if err != nil {
			klog.Fatalf("newProjectHolder()=%v, want nil", err)
		}
		testFlags.project = ph.AcquireOrDie(testFlags.boskosResourceType)
		defer func() {
			out, err := exec.Command("bash", "test/cleanup-all.sh").CombinedOutput()
			if err != nil {
				// Fail now because we shouldn't continue testing if any step fails.
				klog.Errorf("failed to run ./test/cleanup-all.sh: %q, err: %v", out, err)
			}
			ph.Release()
		}()

		if _, ok := os.LookupEnv("USER"); !ok {
			if err := os.Setenv("USER", "prow"); err != nil {
				klog.Fatalf("failed to set user in prow to prow: %v, want nil", err)
			}
		}

		output, err := exec.Command("gcloud", "config", "get-value", "project").CombinedOutput()
		if err != nil {
			klog.Fatalf("failed to get gcloud project: %q: %v, want nil", string(output), err)
		}
		oldProject := strings.TrimSpace(string(output))
		klog.Infof("Using project %s for testing. Restore to existing project %s after testing.", testFlags.project, oldProject)

		if err := setEnvProject(testFlags.project); err != nil {
			klog.Fatalf("setEnvProject(%q) failed: %v, want nil", testFlags.project, err)
		}

		// After the test, reset the project
		defer func() {
			if err := setEnvProject(oldProject); err != nil {
				klog.Errorf("setEnvProject(%q) failed: %v, want nil", oldProject, err)
			}
		}()
	}

	ctx := context.Background()
	client, err := google.DefaultClient(ctx, compute.ComputeScope)
	if err != nil {
		log.Fatal(err)
	}
	svc, err := cloud.NewService(ctx, client, &cloud.SingleProjectRouter{ID: testFlags.project}, &cloud.NopRateLimiter{})
	if err != nil {
		log.Fatal(err)
	}
	theCloud = cloud.NewGCE(svc)

	os.Exit(m.Run())
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
