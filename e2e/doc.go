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

// Package e2e tests the functionality of cloud adaptor against actual GCP.
// For this test to work, you must have a valid credential and access to the
// APIs tested:
//
//	$ gcloud auth application-default login
//	$ go test ./e2e
//
// Run with coverage:
//
//	$ go test -coverpkg ./pkg/cloud -coverprofile cov.out ./e2e ./pkg/cloud
//	$ go tool cover -html cov.out
package e2e
