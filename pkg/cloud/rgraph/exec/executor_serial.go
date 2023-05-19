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
)

// NewSerialExecutor returns a new Executor that runs tasks single-threaded.
func NewSerialExecutor(pending []Action, opts ...ExecutorOption) *serialExecutor {
	ret := &serialExecutor{
		config: defaultExecutorConfig(),
		result: &ExecutorResult{Pending: pending},
	}
	for _, opt := range opts {
		opt(ret.config)
	}

	if ret.config.DryRun {
		ret.runFunc = func(ctx context.Context, c cloud.Cloud, a Action) ([]Event, error) {
			return a.DryRun(), nil
		}
	} else {
		ret.runFunc = func(ctx context.Context, c cloud.Cloud, a Action) ([]Event, error) {
			return a.Run(ctx, c)
		}
	}

	return ret
}

type serialExecutor struct {
	config *ExecutorConfig

	runFunc func(context.Context, cloud.Cloud, Action) ([]Event, error)
	result  *ExecutorResult
}

var _ Executor = (*serialExecutor)(nil)

func (ex *serialExecutor) Run(ctx context.Context, c cloud.Cloud) (*ExecutorResult, error) {
	for a := ex.next(); a != nil; a = ex.next() {
		err := ex.runAction(ctx, c, a)
		if err != nil {
			return ex.result, err
		}
	}
	if ex.config.Tracer != nil {
		ex.config.Tracer.Finish(ex.result.Pending)
	}
	if len(ex.result.Errors) > 0 {
		return ex.result, fmt.Errorf("serialExecutor: errors in execution %v", ex.result.Errors)
	}

	return ex.result, nil
}

func (ex *serialExecutor) runAction(ctx context.Context, c cloud.Cloud, a Action) error {
	te := &TraceEntry{
		Action: a,
		Start:  time.Now(),
	}
	events, runErr := ex.runFunc(ctx, c, a)
	te.End = time.Now()

	if runErr == nil {
		ex.result.Completed = append(ex.result.Completed, a)
	} else {
		ex.result.Errors = append(ex.result.Errors, ActionWithErr{Action: a, Err: runErr})
		switch ex.config.ErrorStrategy {
		case ContinueOnError:
		case StopOnError:
			return fmt.Errorf("serialExecutor: stopping execution (got %v)", runErr)
		default:
			return fmt.Errorf("serialExecutor: invalid ErrorStrategy %q", ex.config.ErrorStrategy)
		}
	}
	for _, ev := range events {
		signaled := ex.signal(ev)
		te.Signaled = append(te.Signaled, signaled...)
	}
	if ex.config.Tracer != nil {
		ex.config.Tracer.Record(te, runErr)
	}

	return nil
}

func (ex *serialExecutor) next() Action {
	for i, a := range ex.result.Pending {
		if a.CanRun() {
			ex.result.Pending = append(ex.result.Pending[0:i], ex.result.Pending[i+1:]...)
			return a
		}
	}
	return nil
}

func (ex *serialExecutor) signal(ev Event) []TraceSignal {
	var ret []TraceSignal
	for _, a := range ex.result.Pending {
		if a.Signal(ev) {
			ret = append(ret, TraceSignal{Event: ev, SignaledAction: a})
		}
	}
	return ret
}
