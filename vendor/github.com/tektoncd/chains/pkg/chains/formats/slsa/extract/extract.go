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

package extract

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/artifact"
	"github.com/tektoncd/chains/pkg/chains/formats/slsa/internal/slsaconfig"
	"github.com/tektoncd/chains/pkg/chains/objects"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
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
	case *v1.PipelineRun:
		subjects = subjectsFromPipelineRun(ctx, obj, slsaconfig)
	case *v1.TaskRun:
		subjects = subjectsFromTektonObject(ctx, obj)
	default:
		logger := logging.FromContext(ctx)
		logger.Warnf("object type %T not supported", obj.GetObject())
	}

	return subjects
}

func subjectsFromPipelineRun(ctx context.Context, obj objects.TektonObject, slsaconfig *slsaconfig.SlsaConfig) []*intoto.ResourceDescriptor {
	prSubjects := subjectsFromTektonObject(ctx, obj)

	// If deep inspection is not enabled, just return subjects observed on the pipelinerun level
	if !slsaconfig.DeepInspectionEnabled {
		return prSubjects
	}

	logger := logging.FromContext(ctx)
	// If deep inspection is enabled, collect subjects from child taskruns
	var result []*intoto.ResourceDescriptor

	pro := obj.(*objects.PipelineRunObjectV1)

	pSpec := pro.Status.PipelineSpec
	if pSpec != nil {
		pipelineTasks := pSpec.Tasks
		pipelineTasks = append(pipelineTasks, pSpec.Finally...)
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
				trSubjects := subjectsFromTektonObject(ctx, tr)
				result = artifact.AppendSubjects(result, trSubjects...)
			}
		}
	}

	// also add subjects observed from pipelinerun level with duplication removed
	result = artifact.AppendSubjects(result, prSubjects...)

	return result
}

func subjectsFromTektonObject(ctx context.Context, obj objects.TektonObject) []*intoto.ResourceDescriptor {
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

// SubjectsFromBuildArtifact returns the software artifacts/images produced by the TaskRun/PipelineRun in the form of standard
// subject field of intoto statement. The detection is based on type hinting. To be read as a software artifact the
// type hintint should:
// - use one of the following type-hints:
//   - Use the *ARTIFACT_OUTPUTS object type-hinting suffix. The value associated with the result should be an object
//     with the fields `uri`, `digest`, and `isBuildArtifact` set to true.
//   - Use the IMAGES type-hint
//   - Use the *IMAGE_URL / *IMAGE_DIGEST type-hint suffix
func SubjectsFromBuildArtifact(ctx context.Context, results []objects.Result) []*intoto.ResourceDescriptor {
	var subjects []*intoto.ResourceDescriptor
	logger := logging.FromContext(ctx)
	buildArtifacts := artifacts.ExtractBuildArtifactsFromResults(ctx, results)
	for _, ba := range buildArtifacts {
		splits := strings.Split(ba.Digest, ":")
		if len(splits) != 2 {
			logger.Errorf("Error procesing build artifact %v, digest %v malformed. Build artifact skipped", ba.FullRef(), ba.Digest)
			continue
		}

		alg := splits[0]
		digest := splits[1]
		subjects = artifact.AppendSubjects(subjects, &intoto.ResourceDescriptor{
			Name: ba.URI,
			Digest: common.DigestSet{
				alg: digest,
			},
		})
	}

	imgs := artifacts.ExtractOCIImagesFromResults(ctx, results)
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

	return subjects
}
