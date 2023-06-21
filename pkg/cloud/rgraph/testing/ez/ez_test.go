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

// Package ez is a utility to create complex resource graphs for testing from a
// concise description by use of naming conventions and default values.
package ez

import compute "google.golang.org/api/compute/v0.beta"

func Example() {
	ezg := Graph{
		Nodes: []Node{
			// Resource types are inferred automatically by naming convention.
			// Anything prefixed with "addr" is an Address. See *Factory.match
			// for the prefixes available.
			{Name: "addr"},
			{
				Name: "fr",
				// Refs handles the complexity of referencing other resources.
				// FIeld refers to the name of the field in the structure.
				Refs: []Ref{
					{Field: "IPAddress", To: "addr"},
					{Field: "Target", To: "thp"},
				},
				// SetupFunc allows for customization of arbitrary resource.
				// This function is called after the automatic values have been
				// set on the resource.
				SetupFunc: func(x *compute.Address) {
					x.Description = "my address"
				},
			},
			{Name: "thp", Refs: []Ref{{Field: "UrlMap", To: "um"}}},
			{Name: "um", Refs: []Ref{{Field: "DefaultService", To: "bs"}}},
			{
				Name: "bs",
				Refs: []Ref{
					// Zonal or regional resources should be referenced by "<scope>/<name>".
					{Field: "Backends.Group", To: "us-central1-a/neg"},
					// Multiple references are expressed by multiple entries to
					// the same Field.
					{Field: "Backends.Group", To: "us-central1-b/neg"},
					{Field: "Healthchecks", To: "hc"},
				},
			},
			{Name: "hc"},
			// Note we have two NEGs with the same name but different scopes.
			{Name: "neg", Zone: "us-central1-a"},
			{Name: "neg", Zone: "us-central1-b"},
		},
	}
	// Build the graph for use in a test.
	ezg.Builder().MustBuild()
}
