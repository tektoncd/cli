/*
Copyright 2022 The Tekton Authors

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

package attest

import (
	"fmt"
	"strings"

	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const (
	CommitParam                  = "CHAINS-GIT_COMMIT"
	URLParam                     = "CHAINS-GIT_URL"
	ChainsReproducibleAnnotation = "chains.tekton.dev/reproducible"
)

type StepAttestation struct {
	EntryPoint  string            `json:"entryPoint"`
	Arguments   interface{}       `json:"arguments,omitempty"`
	Environment interface{}       `json:"environment,omitempty"`
	Annotations map[string]string `json:"annotations"`
}

func Step(step *v1beta1.Step, stepState *v1beta1.StepState) StepAttestation {
	attestation := StepAttestation{}

	entrypoint := strings.Join(step.Command, " ")
	if step.Script != "" {
		entrypoint = step.Script
	}
	attestation.EntryPoint = entrypoint
	attestation.Arguments = step.Args

	env := map[string]interface{}{}
	env["image"] = stepState.ImageID
	env["container"] = stepState.Name
	attestation.Environment = env

	return attestation
}

func Invocation(params []v1beta1.Param, paramSpecs []v1beta1.ParamSpec) slsa.ProvenanceInvocation {
	i := slsa.ProvenanceInvocation{}
	iParams := make(map[string]v1beta1.ArrayOrString)

	// get implicit parameters from defaults
	for _, p := range paramSpecs {
		if p.Default != nil {
			iParams[p.Name] = *p.Default
		}
	}

	// get explicit parameters
	for _, p := range params {
		iParams[p.Name] = p.Value
	}

	i.Parameters = iParams
	return i
}

// supports the SPDX format which is recommended by in-toto
// ref: https://spdx.dev/spdx-specification-21-web-version/#h.49x2ik5
// ref: https://github.com/in-toto/attestation/blob/849867bee97e33678f61cc6bd5da293097f84c25/spec/field_types.md
func SPDXGit(url, revision string) string {
	prefix := "git+"
	if revision == "" {
		return prefix + url + ".git"
	}
	return prefix + url + fmt.Sprintf("@%s", revision)
}
