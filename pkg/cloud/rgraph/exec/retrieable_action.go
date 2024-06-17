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
	"time"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
)

// retriableAction is an action with retry mechanism
type retriableAction struct {
	Action
	canRetry func(error) (bool, time.Duration)
}

// NewRetriableAction is an Action which check if a given action can be retired
// after error. On error the action will be retried when canRetry(err) returns
// true and duration for backoff. Duration equals 0 means that the action needs
// to be retried right away.
func NewRetriableAction(a Action, canRetry func(error) (bool, time.Duration)) Action {
	return &retriableAction{a, canRetry}
}

// Run executes Action. On error `canRetry` function is used to check time
// period after which the action should be retried. If canRetry returns false or
// context is canceled action returns with error.
func (ra *retriableAction) Run(ctx context.Context, c cloud.Cloud) (EventList, error) {
	for {
		events, err := ra.Action.Run(ctx, c)

		if err == nil {
			return events, nil
		}
		if canRetry, backOffTime := ra.canRetry(err); canRetry {
			timer := time.NewTimer(backOffTime)
			select {
			case <-timer.C:
				timer.Stop()
				continue
			case <-ctx.Done():
				timer.Stop()
				return nil, fmt.Errorf("context canceled")
			}
		}
		return events, err
	}
}

// String wraps Action name with retry information
func (ra *retriableAction) String() string {
	return ra.Action.String() + " with retry"
}
