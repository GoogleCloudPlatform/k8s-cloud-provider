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

package trclosure

import (
	"context"
	"fmt"
	"sync"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/algo"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/rgraph/rnode/all"
	"github.com/kr/pretty"
	"k8s.io/klog/v2"
)

// Option for TransitiveClosure.
type Option func(c *Config)

// OnGetFunc is called on the Node after getting the resource from Cloud. This
// can modify properties of the Node, for example, set the appropriate Ownership
// state.
func OnGetFunc(f func(n rnode.Builder) error) Option {
	return func(c *Config) { c.onGet = f }
}

// Config for the algorithm.
type Config struct {
	onGet func(n rnode.Builder) error
}

func makeConfig(opts ...Option) Config {
	config := Config{
		onGet: func(rnode.Builder) error { return nil },
	}
	for _, o := range opts {
		o(&config)
	}
	return config
}

// work is a unit of work for the parallel queue.
type work struct{ b rnode.Builder }

func (wi work) String() string { return wi.b.ID().String() }

func makeErr(s string, args ...any) error { return fmt.Errorf("TransitiveClosure: "+s, args...) }

// Do traverses and fetches the graph, adding all the dependencies into
// the graph, pulling the resource from Cloud as needed.
func Do(ctx context.Context, cl cloud.Cloud, gr *rgraph.Builder, opts ...Option) error {
	subctx, cancel := context.WithCancel(ctx)
	pq := algo.NewParallelQueue[work]()

	err := doInternal(subctx, cl, gr, pq, opts...)
	cancel()

	// Cancel pending traverse operations if we get an error.
	if err != nil {
		klog.Errorf("doInternal() = %v", err)
		waitErr := pq.WaitForOrphans(ctx)
		if waitErr != nil {
			return fmt.Errorf("TransitiveClosure: WaitForOrphans: %w: inner error: %w", waitErr, err)
		}
		return err
	}

	klog.V(2).Info("Do() = nil")

	return nil
}

func doInternal(
	ctx context.Context,
	cl cloud.Cloud,
	gr *rgraph.Builder,
	pq *algo.ParallelQueue[work],
	opts ...Option,
) error {
	config := makeConfig(opts...)

	for _, nb := range gr.All() {
		if err := pq.Add(work{b: nb}); err != nil {
			return fmt.Errorf("transitive closure Do() = %w", err)
		}
	}

	// graphLock is held when updating gr (rgraph.Builder).
	//
	// Invariant: We traverse and add each Node exactly once. We maintain this
	// by holding graphLock while checking and potentially adding the newly
	// traversed Nodes to the graph.
	var graphLock sync.Mutex

	fn := func(ctx context.Context, w work) error {
		outRefs, err := syncNode(ctx, cl, config, w.b)
		if err != nil {
			return err
		}

		for _, ref := range outRefs {
			graphLock.Lock()

			if gr.Get(ref.To) != nil {
				// We have already fetched the Node, don't need to add to the
				// graph and the work queue.
				klog.V(2).Infof("ref.To %+v is already in the graph, ignoring", ref)
				graphLock.Unlock()
				continue
			}
			toNode, err := all.NewBuilderByID(ref.To)
			if err != nil {
				graphLock.Unlock()
				return makeErr("%w", err)
			}

			// Add the untraversed node to the graph.
			klog.V(2).Infof("ref.To %+v has not been traversed, adding to graph", ref)
			gr.Add(toNode)
			graphLock.Unlock()

			if err := pq.Add(work{b: toNode}); err != nil {
				return fmt.Errorf("transitive closure Do() = %w", err)
			}
		}

		return nil
	}

	return pq.Run(ctx, fn)
}

// syncNode loads the resource from the Cloud. This func MUST be threadsafe with
// respect to the Node it is syncing.
func syncNode(ctx context.Context, cl cloud.Cloud, config Config, b rnode.Builder) ([]rnode.ResourceRef, error) {
	// TODO: SyncFromCloud needs to be threadsafe.
	err := b.SyncFromCloud(ctx, cl)
	klog.V(2).Infof("node.SyncFromCloud(%s) = %v (%s)", b.ID(), err, pretty.Sprint(b))

	if err != nil {
		return nil, makeErr("%w", err)
	}
	err = config.onGet(b)
	if err != nil {
		return nil, makeErr("%w", err)
	}

	if b.State() != rnode.NodeExists {
		klog.V(2).Infof("Node %s resource state = %s, no outRefs", b.ID(), b.State())
		return nil, nil
	}

	if b.Ownership() == rnode.OwnershipExternal {
		// Nodes that are ExternallyOwned are not traversed for their references.
		klog.V(2).Infof("Node %s externally owned, no outRefs", b.ID())
		return nil, nil
	}

	outRefs, err := b.OutRefs()
	if err != nil {
		return nil, makeErr("%w", err)
	}

	return outRefs, nil
}
