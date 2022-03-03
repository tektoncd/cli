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
	"sort"
	"strings"

	"github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"

	"github.com/google/go-containerregistry/pkg/name"
)

const (
	tektonID                     = "https://tekton.dev/attestations/chains@v2"
	commitParam                  = "CHAINS-GIT_COMMIT"
	urlParam                     = "CHAINS-GIT_URL"
	ociDigestResult              = "IMAGE_DIGEST"
	chainsDigestSuffix           = "_DIGEST"
	ChainsReproducibleAnnotation = "chains.tekton.dev/reproducible"
)

type InTotoIte6 struct {
	builderID string
	logger    *zap.SugaredLogger
}

func NewFormatter(cfg config.Config, logger *zap.SugaredLogger) (formats.Payloader, error) {
	return &InTotoIte6{
		builderID: cfg.Builder.ID,
		logger:    logger,
	}, nil
}

func (i *InTotoIte6) Wrap() bool {
	return true
}

func (i *InTotoIte6) CreatePayload(obj interface{}) (interface{}, error) {
	var tr *v1beta1.TaskRun
	switch v := obj.(type) {
	case *v1beta1.TaskRun:
		tr = v
	default:
		return nil, fmt.Errorf("intoto does not support type: %s", v)
	}
	return i.generateAttestationFromTaskRun(tr)
}

// generateAttestationFromTaskRun translates a Tekton TaskRun into an in-toto attestation
// with the slsa-provenance predicate type
func (i *InTotoIte6) generateAttestationFromTaskRun(tr *v1beta1.TaskRun) (interface{}, error) {
	subjects := GetSubjectDigests(tr, i.logger)

	att := intoto.ProvenanceStatement{
		StatementHeader: intoto.StatementHeader{
			Type:          intoto.StatementInTotoV01,
			PredicateType: slsa.PredicateSLSAProvenance,
			Subject:       subjects,
		},
		Predicate: slsa.ProvenancePredicate{
			Builder: slsa.ProvenanceBuilder{
				ID: i.builderID,
			},
			BuildType:   tektonID,
			Invocation:  invocation(tr),
			BuildConfig: buildConfig(tr),
			Metadata:    metadata(tr),
			Materials:   materials(tr),
		},
	}
	return att, nil
}

func metadata(tr *v1beta1.TaskRun) *slsa.ProvenanceMetadata {
	m := &slsa.ProvenanceMetadata{}
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

// invocation describes the event that kicked off the build
// we currently don't set ConfigSource because we don't know
// which material the Task definition came from
func invocation(tr *v1beta1.TaskRun) slsa.ProvenanceInvocation {
	i := slsa.ProvenanceInvocation{}
	// get parameters
	params := make(map[string]string)
	for _, p := range tr.Spec.Params {
		params[p.Name] = fmt.Sprintf("%v", p.Value)
	}
	// add params
	if ts := tr.Status.TaskSpec; ts != nil {
		for _, p := range ts.Params {
			if p.Default != nil {
				v := p.Default.StringVal
				if v == "" {
					v = fmt.Sprintf("%v", p.Default.ArrayVal)
				}
				params[p.Name] = fmt.Sprintf("%s", v)
			}
		}
	}
	i.Parameters = params
	return i
}

// GetSubjectDigests extracts OCI images from the TaskRun based on standard hinting set up
// It also goes through looking for any PipelineResources of Image type
func GetSubjectDigests(tr *v1beta1.TaskRun, logger *zap.SugaredLogger) []intoto.Subject {
	var subjects []intoto.Subject

	imgs := artifacts.ExtractOCIImagesFromResults(tr, logger)
	for _, i := range imgs {
		if d, ok := i.(name.Digest); ok {
			subjects = append(subjects, intoto.Subject{
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

// add any Git specification to materials
func materials(tr *v1beta1.TaskRun) []slsa.ProvenanceMaterial {
	var mats []slsa.ProvenanceMaterial
	gitCommit, gitURL := gitInfo(tr)

	// Store git rev as Materials and Recipe.Material
	if gitCommit != "" && gitURL != "" {
		mats = append(mats, slsa.ProvenanceMaterial{
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

		m := slsa.ProvenanceMaterial{
			Digest: slsa.DigestSet{},
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

func (i *InTotoIte6) Type() formats.PayloadType {
	return formats.PayloadTypeInTotoIte6
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
