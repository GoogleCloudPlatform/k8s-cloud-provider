/*
Copyright 2024 Google LLC

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

package cerrors

import (
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/api/googleapi"
)

func TestIsGoogleApiErrorCode(t *testing.T) {
	for _, tc := range []struct {
		desc     string
		err      error
		want     bool
		wantCode int
	}{
		{
			desc:     "Nil error",
			wantCode: http.StatusBadRequest,
		},
		{
			desc:     "Not a google API error",
			err:      fmt.Errorf("some error"),
			wantCode: http.StatusOK,
		},
		{
			desc:     "Google API error",
			err:      &googleapi.Error{Message: "some message"},
			wantCode: http.StatusBadRequest,
		},
		{
			desc:     "Google API error status match",
			err:      &googleapi.Error{Code: http.StatusBadRequest, Message: "some message"},
			wantCode: http.StatusBadRequest,
			want:     true,
		},
		{
			desc:     "Google API error status mismatch",
			err:      &googleapi.Error{Code: http.StatusBadRequest, Message: "some message"},
			wantCode: http.StatusBadGateway,
			want:     false,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			got := isGoogleAPIErrorCode(tc.err, tc.wantCode)
			if got != tc.want {
				t.Errorf("isGoogleAPIErrorCode(%v, %v) = %v, want %v", tc.err, tc.wantCode, got, tc.want)
			}
		})
	}
}

func TestIsGoogleAPINotFound(t *testing.T) {
	for _, tc := range []struct {
		desc string
		err  error
		want bool
	}{
		{
			desc: "Nil error",
		},
		{
			desc: "Not a google API error",
			err:  fmt.Errorf("some error"),
		},
		{
			desc: "Google API error",
			err:  &googleapi.Error{Message: "some message"},
		},
		{
			desc: "Google API NotFound error",
			err:  &googleapi.Error{Code: http.StatusNotFound, Message: "some message"},
			want: true,
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			got := IsGoogleAPINotFound(tc.err)
			if got != tc.want {
				t.Errorf("IsGoogleAPINotFound(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
