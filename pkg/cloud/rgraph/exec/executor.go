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

	"github.com/GoogleCloudPlatform/k8s-cloud-provider/pkg/cloud"
)

type Result struct {
	// Completed Actions with no errors.
	Completed []Action
	// Errors are Actions that failed with an error.
	Errors []ActionWithErr
	// Pending are Actions that could not be executed due to missing
	// preconditions.
	Pending []Action
}

type ActionWithErr struct {
	Action Action
	Err    error
}

// Executor peforms the operations given by a list of Actions.
type Executor interface {
	// Run the actions. Returns non-nil if there was an error in execution of
	// one or more Actions.
	Run(context.Context, cloud.Cloud) (*Result, error)
}

type Option func(*ExecutorConfig)

// TracerOption sets a tracer to accumulate the execution of the Actions.
func TracerOption(t Tracer) Option {
	return func(c *ExecutorConfig) { c.Tracer = t }
}

// DryRunOption will run in dry run mode if true.
func DryRunOption(dryRun bool) Option {
	return func(c *ExecutorConfig) { c.DryRun = dryRun }
}

// ErrorStrategy to use when an Action returns an error.
type ErrorStrategy string

var (
	// ContinueOnError tells the Executor to continue to execute as much of the
	// plan as possible. Note that the dependencies of failed Actions will
	// remain pending and not run.
	ContinueOnError ErrorStrategy = "ContinueOnError"
	// StopOnError attempts to stop execution early if there are errors. Due to
	// asynchronous execution, some Actions may continue to be executed after
	// error detection.
	StopOnError ErrorStrategy = "StopOnError"
)

// ErrorStrategyOption sets the error handling strategy.
func ErrorStrategyOption(s ErrorStrategy) Option {
	return func(c *ExecutorConfig) { c.ErrorStrategy = s }
}

func defaultExecutorConfig() *ExecutorConfig {
	return &ExecutorConfig{
		DryRun:        false,
		ErrorStrategy: StopOnError,
	}
}

// ExecutorConfig for the executor implementation.
type ExecutorConfig struct {
	Tracer        Tracer
	DryRun        bool
	ErrorStrategy ErrorStrategy
}

func (c *ExecutorConfig) validate() error {
	switch c.ErrorStrategy {
	case ContinueOnError, StopOnError:
	default:
		return fmt.Errorf("invalid ErrorStrategy: %q", c.ErrorStrategy)
	}
	return nil
}
