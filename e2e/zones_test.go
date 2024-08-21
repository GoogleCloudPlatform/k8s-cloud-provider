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
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"google.golang.org/api/googleapi"
)

func TestZones(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	zones, err := theCloud.Zones().List(ctx, nil)
	if err != nil {
		t.Fatalf("Error listing zones: %v", err)
	}

	const zoneName = "us-central1-b"

	t.Logf("Got %d zones", len(zones))

	var found bool
	for _, z := range zones {
		if z.Name == zoneName {
			found = true
		}
	}
	if !found {
		t.Fatalf("%q was not in the list of zones", zoneName)
	}

	_, err = theCloud.Zones().Get(ctx, meta.GlobalKey(zoneName))
	if err != nil {
		t.Fatalf("Get(%q) = _, %v; want _, nil", zoneName, err)
	}

	const invalidZone = "moonlab1-c"
	_, err = theCloud.Zones().Get(ctx, meta.GlobalKey(invalidZone))
	gerr, ok := err.(*googleapi.Error)
	if !ok {
		t.Fatalf("Zones.Get(): invalid error type, want *googleapi.Error, got %T", err)
	}
	if gerr.Code != 400 && gerr.Code != 404 {
		t.Fatalf("Zones.Get(): got code %d, want {400, 404} (err: %v)", gerr.Code, err)
	}
}
