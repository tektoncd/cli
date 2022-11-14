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

package taskrun

import (
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats/intotoite6/attest"
	"github.com/tektoncd/chains/pkg/chains/formats/intotoite6/extract"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	"go.uber.org/zap"
)

func GenerateAttestation(builderID string, tro *objects.TaskRunObject, logger *zap.SugaredLogger) (interface{}, error) {
	subjects := extract.SubjectDigests(tro, logger)

	att := intoto.ProvenanceStatement{
		StatementHeader: intoto.StatementHeader{
			Type:          intoto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject:       subjects,
		},
		Predicate: slsa.ProvenancePredicate{
			Builder: slsa.ProvenanceBuilder{
				ID: builderID,
			},
			BuildType:   tro.GetGVK(),
			Invocation:  invocation(tro),
			BuildConfig: buildConfig(tro),
			Metadata:    metadata(tro),
			Materials:   materials(tro, logger),
		},
	}
	return att, nil
}

// invocation describes the event that kicked off the build
// we currently don't set ConfigSource because we don't know
// which material the Task definition came from
func invocation(tro *objects.TaskRunObject) slsa.ProvenanceInvocation {
	var paramSpecs []v1beta1.ParamSpec
	if ts := tro.Status.TaskSpec; ts != nil {
		paramSpecs = ts.Params
	}
	return attest.Invocation(tro.Spec.Params, paramSpecs)
}

func metadata(tro *objects.TaskRunObject) *slsa.ProvenanceMetadata {
	m := &slsa.ProvenanceMetadata{}
	if tro.Status.StartTime != nil {
		m.BuildStartedOn = &tro.Status.StartTime.Time
	}
	if tro.Status.CompletionTime != nil {
		m.BuildFinishedOn = &tro.Status.CompletionTime.Time
	}
	for label, value := range tro.Labels {
		if label == attest.ChainsReproducibleAnnotation && value == "true" {
			m.Reproducible = true
		}
	}
	return m
}

// add any Git specification to materials
func materials(tro *objects.TaskRunObject, logger *zap.SugaredLogger) []slsa.ProvenanceMaterial {
	var mats []slsa.ProvenanceMaterial
	gitCommit, gitURL := gitInfo(tro)

	// Store git rev as Materials and Recipe.Material
	if gitCommit != "" && gitURL != "" {
		mats = append(mats, slsa.ProvenanceMaterial{
			URI:    gitURL,
			Digest: map[string]string{"sha1": gitCommit},
		})
		return mats
	}

	sms := artifacts.RetrieveMaterialsFromStructuredResults(tro, artifacts.ArtifactsInputsResultName, logger)
	mats = append(mats, sms...)

	if tro.Spec.Resources == nil {
		return mats
	}

	// check for a Git PipelineResource
	for _, input := range tro.Spec.Resources.Inputs {
		if input.ResourceSpec == nil || input.ResourceSpec.Type != v1alpha1.PipelineResourceTypeGit {
			continue
		}

		m := slsa.ProvenanceMaterial{
			Digest: slsa.DigestSet{},
		}

		for _, rr := range tro.Status.ResourcesResult {
			if rr.ResourceName != input.Name {
				continue
			}
			if rr.Key == "url" {
				m.URI = attest.SPDXGit(rr.Value, "")
			} else if rr.Key == "commit" {
				m.Digest["sha1"] = rr.Value
			}
		}

		var url string
		var revision string
		for _, param := range input.ResourceSpec.Params {
			if param.Name == "url" {
				url = param.Value
			}
			if param.Name == "revision" {
				revision = param.Value
			}
		}
		m.URI = attest.SPDXGit(url, revision)
		mats = append(mats, m)
	}
	return mats
}

// gitInfo scans over the input parameters and looks for parameters
// with specified names.
func gitInfo(tro *objects.TaskRunObject) (commit string, url string) {
	// Scan for git params to use for materials
	if tro.Status.TaskSpec != nil {
		for _, p := range tro.Status.TaskSpec.Params {
			if p.Default == nil {
				continue
			}
			if p.Name == attest.CommitParam {
				commit = p.Default.StringVal
				continue
			}
			if p.Name == attest.URLParam {
				url = p.Default.StringVal
			}
		}
	}

	for _, p := range tro.Spec.Params {
		if p.Name == attest.CommitParam {
			commit = p.Value.StringVal
			continue
		}
		if p.Name == attest.URLParam {
			url = p.Value.StringVal
		}
	}

	for _, r := range tro.Status.TaskRunResults {
		if r.Name == attest.CommitParam {
			commit = r.Value.StringVal
		}
		if r.Name == attest.URLParam {
			url = r.Value.StringVal
		}
	}

	url = attest.SPDXGit(url, "")
	return
}
