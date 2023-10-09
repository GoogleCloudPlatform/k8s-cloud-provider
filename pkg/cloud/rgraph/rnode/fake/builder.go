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

package fake

import (
	"context"
	"fmt"
	"sync"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"k8s.io/klog/v2"
)

// NewBuilder returns a Node builder.
func NewBuilder(id *cloud.ResourceID) *Builder {
	b := &Builder{}
	b.Defaults(id)
	return b
}

// Builder for Fake resource. Used only for testing.
type Builder struct {
	rnode.BuilderBase

	FakeOutRefs []rnode.ResourceRef
	OutRefsErr  error

	resource Fake
}

// builder implements node.Builder.
var _ rnode.Builder = (*Builder)(nil)

func (b *Builder) Resource() rnode.UntypedResource { return nil }

func (b *Builder) SetResource(u rnode.UntypedResource) error {
	r, ok := u.(Fake)
	if !ok {
		return fmt.Errorf("Fake: invalid type for SetResource: %T", u)
	}
	b.resource = r
	return nil
}

func (b *Builder) SyncFromCloud(ctx context.Context, gcp cloud.Cloud) error {
	// TODO: add ability to inject errors for the SyncFromCloud.
	Mocks.initialize(b)
	return nil
}

func (b *Builder) OutRefs() ([]rnode.ResourceRef, error) {
	if b.OutRefsErr != nil {
		return nil, b.OutRefsErr
	}
	return b.FakeOutRefs, nil
}

func (b *Builder) Build() (rnode.Node, error) {
	ret := &fakeNode{resource: b.resource}
	if err := ret.InitFromBuilder(b); err != nil {
		return nil, err
	}
	return ret, nil
}

var Mocks = newFakeBuilderMocks()

func newFakeBuilderMocks() *FakeBuilderMocks {
	return &FakeBuilderMocks{
		m: map[string]*Builder{},
	}
}

type FakeBuilderMocks struct {
	lock sync.Mutex
	m    map[string]*Builder
}

func (m *FakeBuilderMocks) Clear() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.m = map[string]*Builder{}
}

// Add the mocked fake. Returns true if the mock exists for the given b.ID().
// Warning: this is a global, which means that tests that depend on this CANNOT
// be run in parallel.
func (m *FakeBuilderMocks) Add(b *Builder) bool {
	m.lock.Lock()
	defer m.lock.Unlock()

	if _, ok := m.m[b.ID().String()]; ok {
		return true
	}
	m.m[b.ID().String()] = b
	return false
}

func (m *FakeBuilderMocks) initialize(b *Builder) {
	m.lock.Lock()
	defer m.lock.Unlock()

	klog.Infof("FakeBuilderMocks.initialize(%s)", b.ID())

	if mock, ok := m.m[b.ID().String()]; ok {
		b.SetState(mock.State())
		b.SetOwnership(mock.Ownership())
		b.SetResource(mock.Resource())
		b.FakeOutRefs = mock.FakeOutRefs
		b.OutRefsErr = mock.OutRefsErr
	} else {
		// If the mock doesn't exist, we view this as the resource not existing.
		b.SetState(rnode.NodeDoesNotExist)
	}
}
