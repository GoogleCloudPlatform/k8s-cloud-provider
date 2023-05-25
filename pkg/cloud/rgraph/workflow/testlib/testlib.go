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

package testlib

import (
	"sort"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/exec"
)

// TestCase for a given sequence of configurations.
type TestCase struct {
	// Name of the TestCase. The format should be
	// "<category>/<test-case>/<subtestcase>". This allows for easy filter by
	// prefix matching.
	Name string
	// Description is a human-readable description of what is being tested.
	Description string
	Steps       []Step
}

// Step in the configuration changes.
type Step struct {
	Description string
	// SetUp if non-nil will be called with the cloud interface. Use this to
	// modify the environment, manipulate objects, etc.
	SetUp       func(cloud.Cloud)
	Graph       *rgraph.Graph
	WantActions []exec.Action
}

var (
	all map[string]*TestCase
)

func init() { all = map[string]*TestCase{} }

func Register(tc *TestCase) { all[tc.Name] = tc }
func Cases() []*TestCase {
	var keys []string
	for k := range all {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var ret []*TestCase
	for _, k := range keys {
		ret = append(ret, all[k])
	}

	return ret
}

func Case(name string) *TestCase { return all[name] }
