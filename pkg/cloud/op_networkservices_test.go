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

package cloud

import (
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/google/go-cmp/cmp"
)

func TestParseNetworkServiceOpURL(t *testing.T) {
	t.Parallel()

	type values struct {
		Project string
		Name    string
	}

	for _, tc := range []struct {
		name    string
		in      string
		want    values
		wantErr bool
	}{
		{
			name:    "empty string",
			wantErr: true,
		},
		{
			name: "valid URL",
			in:   "projects/project1/locations/global/operations/operation-name",
			want: values{Project: "project1", Name: "operation-name"},
		},
		{
			name:    "invalid URL path parts",
			in:      "projects/project1/invalid/global/operations/operation-name",
			wantErr: true,
		},
		{
			name:    "invalid scope (only supports global)",
			in:      "projects/project1/locations/us-central1/operations/operation-name",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			in:      "projects/x/y/z",
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := parseNetworkServiceOpURL(tc.in)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("parseNetworkServiceOpURL() = _, %v; gotErr = %t, want %t", err, gotErr, tc.wantErr)
			}
			if err != nil {
				return
			}
			if r.key.Type() != meta.Global {
				t.Errorf("parseNetworkServiceOpURL() = %v; want Global key", r)
			}
			got := values{r.projectID, r.key.Name}
			if diff := cmp.Diff(got, tc.want); diff != "" {
				t.Errorf("parseNetworkServiceOpURL() = %v; cmp.Diff +got/-want: %s", r, diff)
			}
		})
	}
}
