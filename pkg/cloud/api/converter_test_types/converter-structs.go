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

package convert_test_types

// This file contains structs for testing conversion. Type to test conversion
// need to have the same names but different schema that's new file with
// duplicated structs are needed.
type St struct {
	I  int
	PS *string
	LS []string
	M  map[string]string
}

type sti struct {
	C  int
	LS []string
}

type StA struct {
	I  int
	RT *sti
	PS *string
	LS []string
	M  map[string]string
}

type sti2 struct {
	AI string
	BI int
}
type StBI struct {
	Name            string
	SelfLink        string
	SI              *sti2
	NullFields      []string
	ForceSendFields []string
}
