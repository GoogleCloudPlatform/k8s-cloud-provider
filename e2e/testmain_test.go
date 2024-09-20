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
	"os"
	"testing"

	"google.golang.org/api/googleapi"
)

func resourceName(name string) string {
	return TestFlags.ResourcePrefix + RunID + "-" + name
}

func TestMain(m *testing.M) {
	ParseFlagsOrDie()

	ctx := context.Background()
	SetupCloudOrDie(ctx)

	code := m.Run()
	FallbackCleanup(ctx)
	os.Exit(code)
}

func checkErrCode(t *testing.T, err error, wantCode int, fmtStr string, args ...interface{}) {
	t.Helper()

	gerr, ok := err.(*googleapi.Error)
	if !ok {
		t.Fatalf("%s: invalid error type, want *googleapi.Error, got %T", fmt.Sprintf(fmtStr, args...), err)
	}
	if gerr.Code != wantCode {
		t.Fatalf("%s: got code %d, want %d (err: %v)", fmt.Sprintf(fmtStr, args...), gerr.Code, wantCode, err)
	}
}
