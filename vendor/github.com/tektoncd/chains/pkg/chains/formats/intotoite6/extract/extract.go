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
	"sort"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	intoto "github.com/in-toto/in-toto-golang/in_toto"
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/tektoncd/chains/pkg/artifacts"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	"go.uber.org/zap"
)

// GetSubjectDigests extracts OCI images from the TaskRun based on standard hinting set up
// It also goes through looking for any PipelineResources of Image type
func SubjectDigests(obj objects.TektonObject, logger *zap.SugaredLogger) []intoto.Subject {
	var subjects []intoto.Subject

	imgs := artifacts.ExtractOCIImagesFromResults(obj, logger)
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

	sts := artifacts.ExtractSignableTargetFromResults(obj, logger)
	for _, obj := range sts {
		splits := strings.Split(obj.Digest, ":")
		if len(splits) != 2 {
			logger.Errorf("Digest %s should be in the format of: algorthm:abc", obj.Digest)
			continue
		}
		subjects = append(subjects, intoto.Subject{
			Name: obj.URI,
			Digest: slsa.DigestSet{
				splits[0]: splits[1],
			},
		})
	}

	// Check if object is a Taskrun, if so search for images used in PipelineResources
	// Otherwise object is a PipelineRun, where Pipelineresources are not relevant.
	// PipelineResources have been deprecated so their support has been left out of
	// the POC for TEP-84
	// More info: https://tekton.dev/docs/pipelines/resources/
	tr, ok := obj.GetObject().(*v1beta1.TaskRun)
	if !ok || tr.Spec.Resources == nil {
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
			subjects = append(subjects, intoto.Subject{
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
