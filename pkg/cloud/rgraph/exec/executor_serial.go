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

package exec

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
	"k8s.io/klog/v2"
)

// NewSerialExecutor returns a new Executor that runs tasks single-threaded.
func NewSerialExecutor(c cloud.Cloud, opts ...Option) (*serialExecutor, error) {
	ret := &serialExecutor{
		config: defaultExecutorConfig(),
		cloud:  c,
	}
	for _, opt := range opts {
		opt(ret.config)
	}

	if err := ret.config.validate(); err != nil {
		return nil, err
	}

	if ret.config.DryRun {
		ret.runFunc = func(ctx context.Context, c cloud.Cloud, a Action) (EventList, error) {
			return a.DryRun(), nil
		}
	} else {
		ret.runFunc = func(ctx context.Context, c cloud.Cloud, a Action) (EventList, error) {
			return a.Run(ctx, c)
		}
	}

	return ret, nil
}

type serialExecutor struct {
	config *ExecutorConfig

	runFunc func(context.Context, cloud.Cloud, Action) (EventList, error)
	result  *Result
	cloud   cloud.Cloud
}

var _ Executor = (*serialExecutor)(nil)

func (ex *serialExecutor) Run(ctx context.Context, pending []Action) (*Result, error) {

	result := ex.runAction(ctx, ex.cloud, pending)
	if result == nil {
		return result, fmt.Errorf("Executor returned empty result")
	}

	if ex.config.Tracer != nil {
		ex.config.Tracer.Finish(result.Pending)
	}
	if len(result.Errors) > 0 {
		return result, fmt.Errorf("serialExecutor: errors in execution %v", result.Errors)
	}

	return result, nil
}

func (ex *serialExecutor) runAction(ctx context.Context, c cloud.Cloud, actions []Action) *Result {

	result := &Result{Pending: actions}
	for {
		a, i := ex.next(result.Pending)
		if a == nil {
			return result
		}
		result.Pending = append(result.Pending[0:i], result.Pending[i+1:]...)
		klog.Infof("runAction %s", a)

		te := &TraceEntry{
			Action: a,
			Start:  time.Now(),
		}
		events, runErr := ex.runFunc(ctx, c, a)
		te.End = time.Now()

		if runErr == nil {
			result.Completed = append(result.Completed, a)
		} else {
			result.Errors = append(result.Errors, ActionWithErr{Action: a, Err: runErr})
			switch ex.config.ErrorStrategy {
			case ContinueOnError:
			case StopOnError:
				return result
			default:
				// this should never happened config is validated
				klog.Errorf("serialExecutor: invalid ErrorStrategy %q", ex.config.ErrorStrategy)
				return result
			}
		}
		for _, ev := range events {
			signaled := ex.signal(ev, result.Pending)
			te.Signaled = append(te.Signaled, signaled...)
		}
		if ex.config.Tracer != nil {
			ex.config.Tracer.Record(te, runErr)
		}
	}
}

func (ex *serialExecutor) next(pending []Action) (Action, int) {
	for i, a := range pending {
		if a.CanRun() {
			return a, i
		}
	}
	return nil, 0
}

func (ex *serialExecutor) signal(ev Event, pending []Action) []TraceSignal {
	var ret []TraceSignal
	for _, a := range pending {
		if a.Signal(ev) {
			ret = append(ret, TraceSignal{Event: ev, SignaledAction: a})
		}
	}
	return ret
}
