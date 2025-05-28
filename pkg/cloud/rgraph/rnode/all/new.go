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

package all

import (
	"fmt"
	"sync"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
)

type newBuilderFunc func(id *cloud.ResourceID) rnode.Builder

var (
	builderFuncsLock sync.RWMutex
	builderFuncs     = map[string]newBuilderFunc{}
)

// RegisterBuilder registers constructors. shortName is the plural
// lowerCamelCase resource name (e.g. "addresses"). This should only be called
// from init().
func RegisterBuilder(pluralName string, f func(id *cloud.ResourceID) rnode.Builder) {
	builderFuncsLock.Lock()
	defer builderFuncsLock.Unlock()

	if _, ok := builderFuncs[pluralName]; ok {
		panic(fmt.Sprintf("duplicate registration of Builder %q", pluralName))
	}
	builderFuncs[pluralName] = f
}

func NewBuilderByID(id *cloud.ResourceID) (rnode.Builder, error) {
	builderFuncsLock.RLock()
	defer builderFuncsLock.RUnlock()

	if f, ok := builderFuncs[id.Resource]; ok {
		return f(id), nil
	}
	return nil, fmt.Errorf("NewBuilderByID: invalid Resource %q", id.Resource)
}
