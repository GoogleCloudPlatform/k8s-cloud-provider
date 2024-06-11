/*
Copyright 2024 Google LLC

You may obtain a copy of the License at

https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package exec

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
)

// fakeAction will return error for n actions defined in errorRunThreshold,
// runCtr counts all action executions.
// errorRunThreshold set to -1 means that Action should always return error.
type fakeAction struct {
	runCtr            int
	errorRunThreshold int
}

func (fa *fakeAction) CanRun() bool {
	return true
}

func (fa *fakeAction) Signal(e Event) bool {
	return false
}

func (fa *fakeAction) Run(ctx context.Context, c cloud.Cloud) (EventList, error) {
	fa.runCtr++
	if fa.errorRunThreshold > fa.runCtr {
		return EventList{}, fmt.Errorf("Action in error")
	}
	return EventList{}, nil
}

func (fa *fakeAction) DryRun() EventList {
	return EventList{}
}

func (fa *fakeAction) String() string {
	return "fakeAction"
}

func (fa *fakeAction) PendingEvents() EventList {
	return EventList{}
}

func (fa *fakeAction) Metadata() *ActionMetadata {
	return &ActionMetadata{
		Name: "fakeAction",
	}
}

// fakeRetryProvider provides retry mechanism for tests.
// ctr - counts all retry provider calls
// shouldRetry - tells if Action should be rerun
type fakeRetryProvider struct {
	ctr         int
	shouldRetry bool
	backOff     time.Duration
}

// IsRetriable returns info if action should be rerun. Every call to this
// function increments counter.
func (frp *fakeRetryProvider) IsRetriable(error) (bool, time.Duration) {
	frp.ctr++
	return frp.shouldRetry, frp.backOff
}

func TestRetriableAction(t *testing.T) {
	for _, tc := range []struct {
		name               string
		shouldRetry        bool
		wantError          bool
		wantRunThreshold   int
		wantRetriableCalls int
		wantRun            int
	}{
		{

			name:               "should not retry",
			shouldRetry:        false,
			wantError:          true,
			wantRunThreshold:   5,
			wantRetriableCalls: 1,
			wantRun:            1,
		},
		{

			name:               "should retry",
			shouldRetry:        true,
			wantError:          false,
			wantRunThreshold:   5,
			wantRetriableCalls: 4,
			wantRun:            5,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fa := &fakeAction{errorRunThreshold: tc.wantRunThreshold}
			frp := &fakeRetryProvider{shouldRetry: tc.shouldRetry, backOff: 10 * time.Millisecond}
			t.Logf("Create frp with %+v", frp)
			ra := NewRetriableAction(fa, frp.IsRetriable)
			_, err := ra.Run(context.Background(), nil)
			gotErr := err != nil
			if gotErr != tc.wantError {
				t.Fatalf("ra.Run(context.Background(), nil) = %v, gotErr: %v, wantErr : %v", err, gotErr, tc.wantError)
			}

			if fa.runCtr != tc.wantRun {
				t.Errorf("action run mismatch: got %d, want %d", fa.runCtr, tc.wantRun)
			}

			if frp.ctr != tc.wantRetriableCalls {
				t.Errorf("retires mismatch: got %d, want %d", frp.ctr, tc.wantRetriableCalls)
			}
		})
	}
}

func TestRetriableActionWithContextCancel(t *testing.T) {
	fa := &fakeAction{errorRunThreshold: 100}
	frp := &fakeRetryProvider{shouldRetry: true, backOff: 1 * time.Second}
	ra := NewRetriableAction(fa, frp.IsRetriable)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	_, err := ra.Run(ctx, nil)
	cancel()
	t.Logf("ra.Run(_, nil) = %v, want nil", err)
	if err == nil {
		t.Fatalf("ra.Run(_, nil) = %v, want error", err)
	}

	if fa.runCtr > 1 {
		t.Errorf("action run mismatch: got %v, want 1", fa.runCtr)
	}

	if frp.ctr > 1 {
		t.Errorf("retires mismatch: got %v, want 1", frp.ctr)
	}
}
