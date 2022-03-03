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

package provenance

import (
	"fmt"
	"sort"
	"strings"

	"github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/pkg/errors"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/chains/provenance"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"

	"github.com/google/go-containerregistry/pkg/name"
)

const (
	commitParam                  = "CHAINS-GIT_COMMIT"
	urlParam                     = "CHAINS-GIT_URL"
	ChainsReproducibleAnnotation = "chains.tekton.dev/reproducible"
	PredicateType                = "https://tekton.dev/chains/provenance"
	statementType                = "https://in-toto.io/Statement/v0.1"
)

type Provenance struct {
	builderID string
	logger    *zap.SugaredLogger
}

func NewFormatter(cfg config.Config, logger *zap.SugaredLogger) (formats.Payloader, error) {
	errorMsg := `The 'tekton-provenance' format is deprecated, and support will be removed in the next release.
	
	Please switch to the in-toto format by running:
	
	kubectl patch configmap chains-config -n tekton-chains -p='{"data":{"artifacts.taskrun.format": "in-toto"}}'	
	`
	return &Provenance{
		builderID: cfg.Builder.ID,
		logger:    logger,
	}, errors.New(errorMsg)
}

func (i *Provenance) Wrap() bool {
	return true
}

func (i *Provenance) CreatePayload(obj interface{}) (interface{}, error) {
	var tr *v1beta1.TaskRun
	switch v := obj.(type) {
	case *v1beta1.TaskRun:
		tr = v
	default:
		return nil, fmt.Errorf("intoto does not support type: %s", v)
	}
	subjects := GetSubjectDigests(tr, i.logger)
	att, err := i.generateProvenanceFromSubject(tr, subjects)
	if err != nil {
		return nil, errors.Wrapf(err, "generating provenance for subject %s", subjects)
	}
	return att, nil
}

func (i *Provenance) generateProvenanceFromSubject(tr *v1beta1.TaskRun, subjects []in_toto.Subject) (interface{}, error) {
	att := in_toto.Statement{
		StatementHeader: in_toto.StatementHeader{
			Type:          statementType,
			Subject:       subjects,
			PredicateType: PredicateType,
		},
	}

	pred := provenance.ProvenancePredicate{
		Metadata:   metadata(tr),
		Invocation: invocation(i.builderID, tr),
		Materials:  materials(tr),
		Recipe:     provenance.ProvenanceRecipe{Steps: Steps(tr)},
	}

	att.Predicate = pred
	return att, nil
}

func metadata(tr *v1beta1.TaskRun) provenance.ProvenanceMetadata {
	m := provenance.ProvenanceMetadata{}

	if tr.Status.StartTime != nil {
		m.BuildStartedOn = &tr.Status.StartTime.Time
	}
	if tr.Status.CompletionTime != nil {
		m.BuildFinishedOn = &tr.Status.CompletionTime.Time
	}
	for label, value := range tr.Labels {
		if label == ChainsReproducibleAnnotation && value == "true" {
			m.Reproducible = true
		}
	}
	return m
}

// add any Git specification to materials
func materials(tr *v1beta1.TaskRun) []provenance.ProvenanceMaterial {
	var mats []provenance.ProvenanceMaterial
	gitCommit, gitURL := gitInfo(tr)

	// Store git rev as Materials and Recipe.Material
	if gitCommit != "" && gitURL != "" {
		mats = append(mats, provenance.ProvenanceMaterial{
			URI:    gitURL,
			Digest: map[string]string{"revision": gitCommit},
		})
		return mats
	}

	if tr.Spec.Resources == nil {
		return mats
	}

	// check for a Git PipelineResource
	for _, input := range tr.Spec.Resources.Inputs {
		if input.ResourceSpec == nil || input.ResourceSpec.Type != v1alpha1.PipelineResourceTypeGit {
			continue
		}

		m := provenance.ProvenanceMaterial{
			Digest: provenance.DigestSet{},
		}

		for _, rr := range tr.Status.ResourcesResult {
			if rr.ResourceName != input.Name {
				continue
			}
			if rr.Key == "url" {
				m.URI = rr.Value
			} else if rr.Key == "commit" {
				m.Digest["revision"] = rr.Value
			}
		}

		for _, param := range input.ResourceSpec.Params {
			if param.Name == "url" {
				m.URI = param.Value
			}
			if param.Name == "revision" {
				m.Digest[param.Name] = param.Value
			}
		}
		mats = append(mats, m)
	}

	return mats
}

func invocation(builderID string, tr *v1beta1.TaskRun) provenance.Invocation {
	// take care of the eventID
	invocation := provenance.Invocation{
		EventID: string(tr.UID),
	}

	// get parameters
	var params []string
	for _, p := range tr.Spec.Params {
		params = append(params, fmt.Sprintf("%s=%v", p.Name, p.Value))
	}
	// add params
	if ts := tr.Status.TaskSpec; ts != nil {
		for _, p := range ts.Params {
			if p.Default != nil {
				v := p.Default.StringVal
				if v == "" {
					v = fmt.Sprintf("%v", p.Default.ArrayVal)
				}
				params = append(params, fmt.Sprintf("%s=%s", p.Name, v))
			}
		}
	}

	invocation.Parameters = params

	// get URI
	invocation.RecipeURI = recipeURI(tr)
	invocation.ID = builderID
	return invocation
}

// This would be nil for an "inline task", a URI for the Task definition itself
// if it was created in-cluster, or an OCI uri if it came from OCI
// Something else if it came from a pipeline
func recipeURI(tr *v1beta1.TaskRun) string {
	if tr.Spec.TaskRef == nil {
		return ""
	}
	if oci := tr.Spec.TaskRef.Bundle; oci != "" {
		return fmt.Sprintf("oci://%s", oci)
	}
	// look for the task ref
	if tr.Spec.TaskRef.Kind == v1beta1.NamespacedTaskKind {
		return fmt.Sprintf("task://%s", tr.Spec.TaskRef.Name)
	}
	if tr.Spec.TaskRef.Kind == v1beta1.ClusterTaskKind {
		return fmt.Sprintf("clusterTask://%s", tr.Spec.TaskRef.Name)
	}
	// look for the pipeline
	if pipeline, ok := tr.ObjectMeta.Annotations["tekton.dev/pipeline"]; ok {
		return fmt.Sprintf("pipeline://%s", pipeline)
	}
	return ""
}

func Steps(tr *v1beta1.TaskRun) []provenance.RecipeStep {
	steps := []provenance.RecipeStep{}

	for _, step := range tr.Status.Steps {
		s := provenance.RecipeStep{}
		c := container(step, tr)
		// get the entrypoint
		entrypoint := strings.Join(c.Command, " ")
		if c.Script != "" {
			entrypoint = c.Script
		}
		s.EntryPoint = entrypoint
		s.Arguments = c.Args

		// env comprises of:
		// 1. The image
		env := map[string]interface{}{}
		env["image"] = step.ImageID
		env["container"] = step.Name
		s.Environment = env

		// append to all of the steps
		steps = append(steps, s)
	}
	return steps
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

func (i *Provenance) Type() formats.PayloadType {
	return formats.PayloadTypeProvenance
}

// gitInfo scans over the input parameters and looks for parameters
// with specified names.
func gitInfo(tr *v1beta1.TaskRun) (commit string, url string) {
	// Scan for git params to use for materials
	for _, p := range tr.Spec.Params {
		if p.Name == commitParam {
			commit = p.Value.StringVal
			continue
		}
		if p.Name == urlParam {
			url = p.Value.StringVal
		}
	}

	if tr.Status.TaskSpec != nil {
		for _, p := range tr.Status.TaskSpec.Params {
			if p.Default == nil {
				continue
			}
			if p.Name == commitParam {
				commit = p.Default.StringVal
				continue
			}
			if p.Name == urlParam {
				url = p.Default.StringVal
			}
		}
	}

	for _, r := range tr.Status.TaskRunResults {
		if r.Name == commitParam {
			commit = r.Value
		}
		if r.Name == urlParam {
			url = r.Value
		}
	}

	return
}

// GetSubjectDigests depends on taskResults with names ending with
// _DIGEST.
// To be able to find the resource that matches the digest, it relies on a
// naming schema for an input parameter.
// If the output result from this task is foo_DIGEST, it expects to find an
// parameter named 'foo', and resolves this value to use for the subject with
// for the found digest.
// If no parameter is found, we search the results of this task, which allows
// for a task to create a random resource not known prior to execution that
// gets checksummed and added as the subject.
// Digests can be on two formats: $alg:$digest (commonly used for container
// image hashes), or $alg:$digest $path, which is used when a step is
// calculating a hash of a previous step.
func GetSubjectDigests(tr *v1beta1.TaskRun, logger *zap.SugaredLogger) []in_toto.Subject {
	var subjects []in_toto.Subject

	imgs := artifacts.ExtractOCIImagesFromResults(tr, logger)
	for _, i := range imgs {
		if d, ok := i.(name.Digest); ok {
			subjects = append(subjects, in_toto.Subject{
				Name: d.Repository.Name(),
				Digest: slsa.DigestSet{
					"sha256": strings.TrimPrefix(d.DigestStr(), "sha256:"),
				},
			})
		}
	}

	if tr.Spec.Resources == nil {
		return subjects
	}

	// go through resourcesResult
	for _, output := range tr.Spec.Resources.Outputs {
		name := output.Name
		if output.PipelineResourceBinding.ResourceSpec == nil {
			continue
		}
		// similarly, we could do this for other pipeline resources or whatever thing replaces them
		if output.PipelineResourceBinding.ResourceSpec.Type == v1alpha1.PipelineResourceTypeImage {
			// get the url and digest, and save as a subject
			var url, digest string
			for _, s := range tr.Status.ResourcesResult {
				if s.ResourceName == name {
					if s.Key == "url" {
						url = s.Value
					}
					if s.Key == "digest" {
						digest = s.Value
					}
				}
			}
			subjects = append(subjects, in_toto.Subject{
				Name: url,
				Digest: slsa.DigestSet{
					"sha256": strings.TrimPrefix(digest, "sha256:"),
				},
			})
		}
	}
	sort.Slice(subjects, func(i, j int) bool {
		return subjects[i].Name <= subjects[j].Name
	})
	return subjects
}
