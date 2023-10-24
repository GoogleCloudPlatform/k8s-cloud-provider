/*
Copyright 2018 Google LLC

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

package cloud

import (
	"context"
	"errors"
	"testing"
	"time"

	alpha "google.golang.org/api/compute/v0.alpha"
	beta "google.golang.org/api/compute/v0.beta"
	ga "google.golang.org/api/compute/v1"
	"google.golang.org/api/networkservices/v1"
	networkservicesbeta "google.golang.org/api/networkservices/v1beta1"
)

func TestPollOperation(t *testing.T) {
	testErr := errors.New("test error")
	tests := []struct {
		name                  string
		op                    *fakeOperation
		cancel                bool
		wantErr               error
		wantRemainingAttempts int
	}{
		{
			name: "Retry",
			op:   &fakeOperation{attemptsRemaining: 10},
		},
		{
			name: "OperationFailed",
			op: &fakeOperation{
				attemptsRemaining: 2,
				err:               testErr,
			},
			wantErr: testErr,
		},
		{
			name: "DoneFailed",
			op: &fakeOperation{
				attemptsRemaining: 2,
				doneErr:           testErr,
			},
			wantErr: testErr,
		},
		{
			name:                  "Cancel",
			op:                    &fakeOperation{attemptsRemaining: 1},
			cancel:                true,
			wantErr:               context.Canceled,
			wantRemainingAttempts: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s := Service{RateLimiter: &NopRateLimiter{}}
			ctx, cfn := context.WithTimeout(context.Background(), 3*time.Second)
			defer cfn()
			if test.cancel {
				cfn()
			}
			if gotErr := s.pollOperation(ctx, test.op); gotErr != test.wantErr {
				t.Errorf("pollOperation: got %v, want %v", gotErr, test.wantErr)
			}
			if test.op.attemptsRemaining != test.wantRemainingAttempts {
				t.Errorf("%d attempts remaining, want %d", test.op.attemptsRemaining, test.wantRemainingAttempts)
			}
		})
	}
}

type fakeOperation struct {
	attemptsRemaining int
	doneErr           error
	err               error
}

func (f *fakeOperation) isDone(ctx context.Context) (bool, error) {
	f.attemptsRemaining--
	if f.attemptsRemaining <= 0 {
		return f.doneErr == nil, f.doneErr
	}
	return false, nil
}

func (f *fakeOperation) error() error {
	return f.err
}

func (f *fakeOperation) rateLimitKey() *RateLimitKey {
	return nil
}

func TestWrapOperation(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		in      any
		want    string // oneof(ga, alpha, beta, nsga)
		wantErr bool
	}{
		{
			in: &ga.Operation{
				Kind:          "compute#operation",
				Id:            4697186249068962421,
				Name:          "operation-1698127001961-6087000bc4354-7d13ece8-99e595cb",
				OperationType: "insert",
				TargetLink:    "https://www.googleapis.com/compute/v1/projects/my-project/global/healthChecks/foo",
				TargetId:      462163996302828149,
				Status:        "DONE",
				SelfLink:      "https://www.googleapis.com/compute/v1/projects/my-project/global/operations/operation-1698127001961-6087000bc4354-7d13ece8-99e595cb",
			},
			want: "ga",
		},
		{
			in: &alpha.Operation{
				Kind:          "compute#operation",
				Id:            4697186249068962421,
				Name:          "operation-1698127001961-6087000bc4354-7d13ece8-99e595cb",
				OperationType: "insert",
				TargetLink:    "https://www.googleapis.com/compute/v1/projects/my-project/global/healthChecks/foo",
				TargetId:      462163996302828149,
				Status:        "DONE",
				SelfLink:      "https://www.googleapis.com/compute/v1/projects/my-project/global/operations/operation-1698127001961-6087000bc4354-7d13ece8-99e595cb",
			},
			want: "alpha",
		},
		{
			in: &beta.Operation{
				Kind:          "compute#operation",
				Id:            4697186249068962421,
				Name:          "operation-1698127001961-6087000bc4354-7d13ece8-99e595cb",
				OperationType: "insert",
				TargetLink:    "https://www.googleapis.com/compute/v1/projects/my-project/global/healthChecks/foo",
				TargetId:      462163996302828149,
				Status:        "DONE",
				SelfLink:      "https://www.googleapis.com/compute/v1/projects/my-project/global/operations/operation-1698127001961-6087000bc4354-7d13ece8-99e595cb",
			},
			want: "beta",
		},
		{
			in: &networkservices.Operation{
				Name: "projects/my-project/locations/global/operations/operation-1234",
			},
			want: "nsga",
		},
		{
			in: &networkservicesbeta.Operation{
				Name: "projects/my-project/locations/global/operations/operation-1234",
			},
			want: "nsga",
		},
		{
			in:      struct{}{},
			wantErr: true,
		},
	} {
		t.Run(tc.want, func(t *testing.T) {
			svc := Service{}
			op, err := svc.wrapOperation(tc.in)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Fatalf("gotErr = %t, want %t", gotErr, tc.wantErr)
			}
			if err != nil {
				return
			}
			var gotType string
			switch op.(type) {
			case *gaOperation:
				gotType = "ga"
			case *alphaOperation:
				gotType = "alpha"
			case *betaOperation:
				gotType = "beta"
			case *networkServicesOperation:
				gotType = "nsga"
			default:
				gotType = "invalid"
			}
			if gotType != tc.want {
				t.Errorf("gotType = %q, want %q", gotType, tc.want)
			}
		})
	}
}
