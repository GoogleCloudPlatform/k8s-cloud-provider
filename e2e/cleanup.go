package e2e

import (
	"context"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/filter"
	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
)

func matchTestResource(name string) bool {
	if RunID == "" {
		return strings.HasPrefix(name, TestFlags.ResourcePrefix)
	} else {
		return strings.HasPrefix(name, fmt.Sprintf("%s%s-", TestFlags.ResourcePrefix, RunID))
	}
}

func cleanupMeshes(ctx context.Context) {
	tcprs, err := theCloud.Meshes().List(ctx, filter.None)
	if err != nil {
		log.Printf("FallbackCleanup: theCloud.Meshes().List(ctx, _): %v\n", err)
		return
	}
	for _, tcpr := range tcprs {
		name := path.Base(tcpr.Name)
		if !matchTestResource(name) {
			continue
		}
		key := meta.GlobalKey(name)
		err = theCloud.Meshes().Delete(ctx, key)
		log.Printf("FallbackCleanup: theCloud.Meshes().Delete(ctx, %s): %v\n", key, err)
	}
}

func cleanupTcpRoutes(ctx context.Context) {
	tcprs, err := theCloud.TcpRoutes().List(ctx, filter.None)
	if err != nil {
		log.Printf("FallbackCleanup: theCloud.TcpRoutes().List(ctx, _): %v\n", err)
		return
	}
	for _, tcpr := range tcprs {
		name := path.Base(tcpr.Name)
		if !matchTestResource(name) {
			continue
		}
		key := meta.GlobalKey(name)
		err = theCloud.TcpRoutes().Delete(ctx, key)
		log.Printf("FallbackCleanup: theCloud.TcpRoutes().Delete(ctx, %s): %v\n", key, err)
	}
}

func cleanupBackendServices(ctx context.Context) {
	bss, err := theCloud.BackendServices().List(ctx, filter.None)
	if err != nil {
		log.Printf("FallbackCleanup: theCloud.BackendServices().List(ctx, _): %v\n", err)
		return
	}
	for _, bs := range bss {
		if !matchTestResource(bs.Name) {
			continue
		}
		key := meta.GlobalKey(bs.Name)
		err = theCloud.BackendServices().Delete(ctx, key)
		log.Printf("FallbackCleanup: theCloud.BackendServices().Delete(ctx, %s): %v\n", key, err)
	}
}

func cleanupHealthChecks(ctx context.Context) {
	hcs, err := theCloud.HealthChecks().List(ctx, filter.None)
	if err != nil {
		log.Printf("FallbackCleanup: theCloud.HealthChecks().List(ctx, _): %v\n", err)
		return
	}
	for _, hc := range hcs {
		if !matchTestResource(hc.Name) {
			continue
		}
		key := meta.GlobalKey(hc.Name)
		err = theCloud.HealthChecks().Delete(ctx, key)
		log.Printf("FallbackCleanup: theCloud.HealthChecks().Delete(ctx, %s): %v\n", key, err)
	}
}

// FallbackCleanup cleans all the resources created during the test run.
func FallbackCleanup(ctx context.Context) {
	cleanupTcpRoutes(ctx)
	cleanupBackendServices(ctx)
	cleanupHealthChecks(ctx)
	cleanupMeshes(ctx)
}
