/*
Copyright 2021 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package intotoite6

import (
	"fmt"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// BuildConfig is the custom Chains format to fill out the
// "buildConfig" section of the slsa-provenance predicate
type BuildConfig struct {
	Steps []Step `json:"steps"`
}

// Step corresponds to one step in the TaskRun
type Step struct {
	EntryPoint  string            `json:"entryPoint"`
	Arguments   interface{}       `json:"arguments,omitempty"`
	Environment interface{}       `json:"environment,omitempty"`
	Annotations map[string]string `json:"annotations"`
}

func buildConfig(tr *v1beta1.TaskRun) BuildConfig {
	steps := []Step{}
	for _, step := range tr.Status.Steps {
		fmt.Println(step)
		s := Step{}
		c := container(step, tr)
		// get the entrypoint
		entrypoint := strings.Join(c.Command, " ")
		if c.Script != "" {
			entrypoint = c.Script
		}
		s.EntryPoint = entrypoint
		s.Arguments = c.Args

		// env comprises of:
		env := map[string]interface{}{}
		env["image"] = step.ImageID
		env["container"] = step.Name
		s.Environment = env

		// append to all of the steps
		steps = append(steps, s)
	}
	return BuildConfig{Steps: steps}
}

func container(stepState v1beta1.StepState, tr *v1beta1.TaskRun) v1beta1.Step {
	name := stepState.Name
	if tr.Status.TaskSpec != nil {
		for _, s := range tr.Status.TaskSpec.Steps {
			if s.Name == name {
				return s
			}
		}
	}
	return v1beta1.Step{}
}
