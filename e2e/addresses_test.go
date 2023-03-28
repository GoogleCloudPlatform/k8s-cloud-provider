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
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud/meta"
	"github.com/kr/pretty"
	"google.golang.org/api/compute/v1"
)

func TestAddresses(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	const regionName = "us-central1"
	addr1 := &compute.Address{
		AddressType: "EXTERNAL",
		Description: "k8s-cloud-provider-test",
		Name:        resourceName("addr1"),
		NetworkTier: "STANDARD",
		Region:      regionName,
	}
	addr1Key := meta.RegionalKey(addr1.Name, regionName)

	t.Logf("addr1 = %s", fmt.Sprint(addr1))

	t.Cleanup(func() { theCloud.Addresses().Delete(context.Background(), addr1Key) })

	// Insert
	err := theCloud.Addresses().Insert(ctx, addr1Key, addr1)
	if err != nil {
		t.Fatalf("Addresses.Insert(addr1) = %v", err)
	}

	// Get
	a, err := theCloud.Addresses().Get(ctx, addr1Key)
	if err != nil {
		t.Fatalf("Addresses.Get(addr1) = %v", err)
	}
	if a.Name != addr1.Name || a.Description != addr1.Description {
		t.Fatalf("Addresses.Get() did not match, got %s\nwant %s", pretty.Sprint(a), pretty.Sprint(addr1))
	}

	// List
	al, err := theCloud.Addresses().List(ctx, regionName, nil)
	if err != nil {
		t.Fatalf("Error listing Addresses: %v", err)
	}

	var found bool
	for _, a := range al {
		if a.Name == addr1.Name {
			found = true
		}
	}

	if !found {
		t.Fatalf("Expected to find Address %q but it didn't exist", addr1.Name)
	}

	// Delete
	err = theCloud.Addresses().Delete(ctx, addr1Key)
	if err != nil {
		t.Fatalf("Addresses.Delete(addr1) = %v", err)
	}

	// AggregatedList
}

func TestGlobalAddresses(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	addr1 := &compute.Address{
		AddressType: "EXTERNAL",
		Description: "k8s-cloud-provider-test",
		Name:        resourceName("addr1"),
		NetworkTier: "PREMIUM",
	}
	addr1Key := meta.GlobalKey(addr1.Name)

	t.Logf("addr1 = %s", fmt.Sprint(addr1))

	t.Cleanup(func() { theCloud.GlobalAddresses().Delete(context.Background(), addr1Key) })

	// Insert
	err := theCloud.GlobalAddresses().Insert(ctx, addr1Key, addr1)
	if err != nil {
		t.Fatalf("GlobalAddresses.Insert(addr1) = %v", err)
	}

	// Get
	a, err := theCloud.GlobalAddresses().Get(ctx, addr1Key)
	if err != nil {
		t.Fatalf("GlobalAddresses.Get(addr1) = %v", err)
	}
	if a.Name != addr1.Name || a.Description != addr1.Description {
		t.Fatalf("GlobalAddresses.Get() did not match, got %s\nwant %s", pretty.Sprint(a), pretty.Sprint(addr1))
	}

	// Delete
	err = theCloud.GlobalAddresses().Delete(ctx, addr1Key)
	if err != nil {
		t.Fatalf("GlobalAddresses.Delete(addr1) = %v", err)
	}
}
