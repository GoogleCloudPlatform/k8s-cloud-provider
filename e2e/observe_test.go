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
	"reflect"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
)

type testObserver struct {
	name           string
	start, end     time.Time
	startCK, endCK cloud.CallContextKey
	err            error
}

func (o *testObserver) Start(ctx context.Context, ck *cloud.CallContextKey) {
	o.start = time.Now()
	o.startCK = *ck
}
func (o *testObserver) End(ctx context.Context, ck *cloud.CallContextKey, err error) {
	o.end = time.Now()
	o.endCK = *ck
	o.err = err
}

func TestObserve(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	o := &testObserver{}
	ctx = cloud.WithCallObserver(ctx, o)

	const invalidZoneName = "moonbase1-b"
	theCloud.Zones().Get(ctx, meta.GlobalKey(invalidZoneName))
	if o.start.IsZero() || o.end.IsZero() {
		t.Fatalf("testObserver start, end was not modified (testObserver = %+v)", o)
	}

	wantCK := cloud.CallContextKey{
		Operation: "Get",
		Version:   meta.VersionGA,
		Service:   "Zones",
		Resource:  &meta.Key{Name: "moonbase1-b"},
	}

	// Ignore differences in ProjectID as they could change when run manually
	// running manually.
	o.startCK.ProjectID = ""
	o.endCK.ProjectID = ""

	if !reflect.DeepEqual(o.startCK, wantCK) {
		t.Fatalf("o.startCK = %+v, want %+v", o.startCK, wantCK)
	}
	if !reflect.DeepEqual(o.endCK, wantCK) {
		t.Fatalf("o.endCK = %+v, want %+v", o.endCK, wantCK)
	}
	if o.err == nil {
		t.Fatal("testObserver.err = nil, want err")
	}
}
