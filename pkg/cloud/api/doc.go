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

// Package api are wrappers for working with the versioned API data types that
// are part of the compute APIs.
//
// # Working with versioned API types.
//
// VersionedObject is used to write version-agnostic code such as
// Kubernetes-style API translators.
//
//	 // Instantiate the adapter.
//	 type Address = VersionedObject[compute.Address, alpha.Address, beta.Address]
//	 addr := Address{}
//
//	// Manipulate the fields in Address.
//	addr.Access(func(x *compute.Address) {
//	  x.Name = "my-addr"
//	  x.Description = ...
//	  x.Network = ...
//	  // Meta-fields are handled as well:
//	  x.ForceSendFields = []string{"Region"}
//	})
//
//	// Edit fields that are present in Beta. Fields that common with the
//	// compute.Address be propagated to all associated types, e.g. a field
//	// like "Name" will be set in all versions of the resource.
//	addr.AccessBeta(func(x *beta.Address) {
//	 x.Name = "different-name"
//	 x.Labels = ...
//	})
//
//	// Fetch the required API object. The code should handle
//	// checking for missing fields that may have been dropped as
//	// part of version translation.
//	betaObj, err := addr.ToBeta()
//	  if err != nil {
//	    var objErrors *Errors
//	    if errors.As(err, &objErrors) { /* handle MissingFields, etc. */ }
//	}
//
// # Assumptions
//
// The code currently handles a narrow range of Go types:
//
//   - Root of the resource is a struct.
//   - Struct fields are: basic type (e.g. int or string), pointer to a basic type,
//     struct or pointer to struct, slice or map.
//   - Map keys are basic type. map values are the same as struct fields.
//   - Slice values are the same types as struct fields.
//   - There are no recursive (cyclic) definitions (e.g. struct S that points (directly
//     or indirectly) to S again.
//   - Fields of the same name between version MUST be of the same basic, slice and
//     map type. Struct can differ and are handled recusively.
//
// Exceptions to these rules can be handled by adding custom type conversions (see below).
//
// # Checking type assumptions with unit tests
//
// VersionedObject.CheckSchema() can be used to check if the types referenced meet
// the above criteria.
//
//	type Address = VersionedObject[compute.Address, alpha.Address, beta.Address]
//	addr := Address{}
//	if err := addr.CheckSchema(); err != nil { /* unsupported type schema */ }
package api
