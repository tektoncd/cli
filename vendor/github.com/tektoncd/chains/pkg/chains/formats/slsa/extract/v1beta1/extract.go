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

package v1beta1

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	"github.com/tektoncd/chains/internal/backport"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/artifact"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/logging"
)

// SubjectDigests returns software artifacts produced from the TaskRun/PipelineRun object
// in the form of standard subject field of intoto statement.
// The type hinting fields expected in results help identify the generated software artifacts.
// Valid type hinting fields must:
//   - have suffix `IMAGE_URL` & `IMAGE_DIGEST` or `ARTIFACT_URI` & `ARTIFACT_DIGEST` pair.
//   - the `*_DIGEST` field must be in the format of "<algorithm>:<actual-sha>" where the algorithm must be "sha256" and actual sha must be valid per https://github.com/opencontainers/image-spec/blob/main/descriptor.md#sha-256.
//   - the `*_URL` or `*_URI` fields cannot be empty.
//
//nolint:all
func SubjectDigests(ctx context.Context, obj objects.TektonObject, slsaconfig *slsaconfig.SlsaConfig) []*intoto.ResourceDescriptor {
	var subjects []*intoto.ResourceDescriptor

	switch obj.GetObject().(type) {
	case *v1beta1.PipelineRun:
		subjects = SubjectsFromPipelineRunV1Beta1(ctx, obj, slsaconfig)
	case *v1beta1.TaskRun:
		subjects = SubjectsFromTektonObjectV1Beta1(ctx, obj)
	}

	return subjects
}

// SubjectsFromPipelineRunV1Beta1 returns software artifacts produced from the PipelineRun object.
func SubjectsFromPipelineRunV1Beta1(ctx context.Context, obj objects.TektonObject, slsaconfig *slsaconfig.SlsaConfig) []*intoto.ResourceDescriptor {
	prSubjects := SubjectsFromTektonObjectV1Beta1(ctx, obj)

	// If deep inspection is not enabled, just return subjects observed on the pipelinerun level
	if !slsaconfig.DeepInspectionEnabled {
		return prSubjects
	}

	logger := logging.FromContext(ctx)
	// If deep inspection is enabled, collect subjects from child taskruns
	var result []*intoto.ResourceDescriptor

	pro := obj.(*objects.PipelineRunObjectV1Beta1)

	pSpec := pro.Status.PipelineSpec
	if pSpec != nil {
		pipelineTasks := append(pSpec.Tasks, pSpec.Finally...)
		for _, t := range pipelineTasks {
			taskRuns := pro.GetTaskRunsFromTask(t.Name)
			if len(taskRuns) == 0 {
				logger.Infof("no taskruns found for task %s", t.Name)
				continue
			}
			for _, tr := range taskRuns {
				// Ignore Tasks that did not execute during the PipelineRun.
				if tr == nil || tr.Status.CompletionTime == nil {
					logger.Infof("taskrun status not found for task %s", t.Name)
					continue
				}
				trSubjects := SubjectsFromTektonObjectV1Beta1(ctx, tr)
				result = artifact.AppendSubjects(result, trSubjects...)
			}
		}
	}

	// also add subjects observed from pipelinerun level with duplication removed
	result = artifact.AppendSubjects(result, prSubjects...)

	return result
}

// SubjectsFromTektonObjectV1Beta1 returns software artifacts produced from the Tekton object.
func SubjectsFromTektonObjectV1Beta1(ctx context.Context, obj objects.TektonObject) []*intoto.ResourceDescriptor {
	logger := logging.FromContext(ctx)
	var subjects []*intoto.ResourceDescriptor

	imgs := artifacts.ExtractOCIImagesFromResults(ctx, obj.GetResults())
	for _, i := range imgs {
		if d, ok := i.(name.Digest); ok {
			subjects = artifact.AppendSubjects(subjects, &intoto.ResourceDescriptor{
				Name: d.Repository.Name(),
				Digest: common.DigestSet{
					"sha256": strings.TrimPrefix(d.DigestStr(), "sha256:"),
				},
			})
		}
	}

	sts := artifacts.ExtractSignableTargetFromResults(ctx, obj)
	for _, obj := range sts {
		splits := strings.Split(obj.Digest, ":")
		if len(splits) != 2 {
			logger.Errorf("Digest %s should be in the format of: algorthm:abc", obj.Digest)
			continue
		}
		subjects = artifact.AppendSubjects(subjects, &intoto.ResourceDescriptor{
			Name: obj.URI,
			Digest: common.DigestSet{
				splits[0]: splits[1],
			},
		})
	}

	ssts := artifacts.ExtractStructuredTargetFromResults(ctx, obj.GetResults(), artifacts.ArtifactsOutputsResultName)
	for _, s := range ssts {
		splits := strings.Split(s.Digest, ":")
		alg := splits[0]
		digest := splits[1]
		subjects = artifact.AppendSubjects(subjects, &intoto.ResourceDescriptor{
			Name: s.URI,
			Digest: common.DigestSet{
				alg: digest,
			},
		})
	}

	// Check if object is a Taskrun, if so search for images used in PipelineResources
	// Otherwise object is a PipelineRun, where Pipelineresources are not relevant.
	// PipelineResources have been deprecated so their support has been left out of
	// the POC for TEP-84
	// More info: https://tekton.dev/docs/pipelines/resources/
	tr, ok := obj.GetObject().(*v1beta1.TaskRun) //nolint:staticcheck
	if !ok || tr.Spec.Resources == nil {         //nolint:staticcheck
		return subjects
	}

	// go through resourcesResult
	for _, output := range tr.Spec.Resources.Outputs { //nolint:staticcheck
		name := output.Name
		if output.PipelineResourceBinding.ResourceSpec == nil {
			continue
		}
		// similarly, we could do this for other pipeline resources or whatever thing replaces them
		if output.PipelineResourceBinding.ResourceSpec.Type == backport.PipelineResourceTypeImage {
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
			subjects = artifact.AppendSubjects(subjects, &intoto.ResourceDescriptor{
				Name: url,
				Digest: common.DigestSet{
					"sha256": strings.TrimPrefix(digest, "sha256:"),
				},
			})
		}
	}
	return subjects
}

// RetrieveAllArtifactURIs returns all the URIs of the software artifacts produced from the run object.
// - It first extracts intoto subjects from run object results and converts the subjects
// to a slice of string URIs in the format of "NAME" + "@" + "ALGORITHM" + ":" + "DIGEST".
// - If no subjects could be extracted from results, then an empty slice is returned.
func RetrieveAllArtifactURIs(ctx context.Context, obj objects.TektonObject, deepInspectionEnabled bool) []string {
	result := []string{}
	subjects := SubjectDigests(ctx, obj, &slsaconfig.SlsaConfig{DeepInspectionEnabled: deepInspectionEnabled})

	for _, s := range subjects {
		for algo, digest := range s.Digest {
			result = append(result, fmt.Sprintf("%s@%s:%s", s.Name, algo, digest))
		}
	}
	return result
}
