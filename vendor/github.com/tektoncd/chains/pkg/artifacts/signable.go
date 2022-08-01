/*
Copyright 2020 The Tekton Authors
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

package artifacts

import (
	_ "crypto/sha256" // Recommended by go-digest.
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/opencontainers/go-digest"
	"github.com/tektoncd/chains/pkg/chains/formats"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/sets"
)

type Signable interface {
	ExtractObjects(tr *v1beta1.TaskRun) []interface{}
	StorageBackend(cfg config.Config) sets.String
	Signer(cfg config.Config) string
	PayloadFormat(cfg config.Config) formats.PayloadType
	Key(interface{}) string
	Type() string
	Enabled(cfg config.Config) bool
}

type TaskRunArtifact struct {
	Logger *zap.SugaredLogger
}

func (ta *TaskRunArtifact) Key(obj interface{}) string {
	tr := obj.(*v1beta1.TaskRun)
	return "taskrun-" + string(tr.UID)
}

func (ta *TaskRunArtifact) ExtractObjects(tr *v1beta1.TaskRun) []interface{} {
	return []interface{}{tr}
}
func (ta *TaskRunArtifact) Type() string {
	return "tekton"
}

func (ta *TaskRunArtifact) StorageBackend(cfg config.Config) sets.String {
	return cfg.Artifacts.TaskRuns.StorageBackend
}

func (ta *TaskRunArtifact) PayloadFormat(cfg config.Config) formats.PayloadType {
	return formats.PayloadType(cfg.Artifacts.TaskRuns.Format)
}

func (ta *TaskRunArtifact) Signer(cfg config.Config) string {
	return cfg.Artifacts.TaskRuns.Signer
}

func (ta *TaskRunArtifact) Enabled(cfg config.Config) bool {
	return cfg.Artifacts.TaskRuns.Enabled()
}

type OCIArtifact struct {
	Logger *zap.SugaredLogger
}

type image struct {
	url    string
	digest string
}

// StructuredSignable contains info for signable targets to become either subjects or materials in intoto Statements.
// URI is the resource uri for the target needed iff the target is a material.
// Digest is the target's SHA digest.
type StructuredSignable struct {
	URI    string
	Digest string
}

func (oa *OCIArtifact) ExtractObjects(tr *v1beta1.TaskRun) []interface{} {
	imageResourceNames := map[string]*image{}
	if tr.Status.TaskSpec != nil && tr.Status.TaskSpec.Resources != nil {
		for _, output := range tr.Status.TaskSpec.Resources.Outputs {
			if output.Type == v1beta1.PipelineResourceTypeImage {
				imageResourceNames[output.Name] = &image{}
			}
		}
	}

	for _, rr := range tr.Status.ResourcesResult {
		img, ok := imageResourceNames[rr.ResourceName]
		if !ok {
			continue
		}
		// We have a result for an image!
		if rr.Key == "url" {
			img.url = rr.Value
		} else if rr.Key == "digest" {
			img.digest = rr.Value
		}
	}

	objs := []interface{}{}
	for _, image := range imageResourceNames {
		dgst, err := name.NewDigest(fmt.Sprintf("%s@%s", image.url, image.digest))
		if err != nil {
			oa.Logger.Error(err)
			continue
		}
		objs = append(objs, dgst)
	}

	// Now check TaskResults
	resultImages := ExtractOCIImagesFromResults(tr, oa.Logger)
	objs = append(objs, resultImages...)

	return objs
}

func ExtractOCIImagesFromResults(tr *v1beta1.TaskRun, logger *zap.SugaredLogger) []interface{} {
	ss := extractTargetFromResults(tr, "IMAGE_URL", "IMAGE_DIGEST", logger)
	objs := []interface{}{}
	for _, s := range ss {
		if s == nil || s.Digest == "" || s.URI == "" {
			continue
		}
		dgst, err := name.NewDigest(fmt.Sprintf("%s@%s", s.URI, s.Digest))
		if err != nil {
			logger.Errorf("error getting digest: %v", err)
			continue
		}

		objs = append(objs, dgst)
	}
	// look for a comma separated list of images
	for _, key := range tr.Status.TaskRunResults {
		if key.Name != "IMAGES" {
			continue
		}
		imgs := strings.FieldsFunc(key.Value.StringVal, split)

		for _, img := range imgs {
			trimmed := strings.TrimSpace(img)
			if trimmed == "" {
				continue
			}
			dgst, err := name.NewDigest(trimmed)
			if err != nil {
				logger.Errorf("error getting digest for img %s: %v", trimmed, err)
				continue
			}
			objs = append(objs, dgst)
		}
	}

	return objs
}

// ExtractSignableTargetFromResults extracts signable targets that aim to generate intoto provenance as materials within TaskRun results and store them as StructuredSignable.
func ExtractSignableTargetFromResults(tr *v1beta1.TaskRun, logger *zap.SugaredLogger) []*StructuredSignable {
	objs := []*StructuredSignable{}
	ss := extractTargetFromResults(tr, "ARTIFACT_URI", "ARTIFACT_DIGEST", logger)
	// Only add it if we got both the signable URI and digest.
	for _, s := range ss {
		if s == nil || s.Digest == "" || s.URI == "" {
			continue
		}
		if err := checkDigest(s.Digest); err != nil {
			logger.Errorf("error getting digest %s: %v", s.Digest, err)
			continue
		}

		objs = append(objs, s)
	}

	return objs
}

func extractTargetFromResults(tr *v1beta1.TaskRun, identifierSuffix string, digestSuffix string, logger *zap.SugaredLogger) map[string]*StructuredSignable {
	ss := map[string]*StructuredSignable{}

	for _, res := range tr.Status.TaskRunResults {
		if strings.HasSuffix(res.Name, identifierSuffix) {
			marker := strings.TrimSuffix(res.Name, identifierSuffix)
			if v, ok := ss[marker]; ok {
				v.URI = strings.TrimSpace(res.Value.StringVal)

			} else {
				ss[marker] = &StructuredSignable{URI: strings.TrimSpace(res.Value.StringVal)}
			}
			// TODO: add logic for Intoto signable target as input.
		}
		if strings.HasSuffix(res.Name, digestSuffix) {
			marker := strings.TrimSuffix(res.Name, digestSuffix)
			if v, ok := ss[marker]; ok {
				v.Digest = strings.TrimSpace(res.Value.StringVal)
			} else {
				ss[marker] = &StructuredSignable{Digest: strings.TrimSpace(res.Value.StringVal)}
			}
		}

	}
	return ss
}

func checkDigest(dig string) error {
	prefix := digest.Canonical.String() + ":"
	if !strings.HasPrefix(dig, prefix) {
		return fmt.Errorf("unsupported digest algorithm: %s", dig)
	}
	hex := strings.TrimPrefix(dig, prefix)
	if err := digest.Canonical.Validate(hex); err != nil {
		return err
	}
	return nil
}

// split allows IMAGES to be separated either by commas (for backwards compatibility)
// or by newlines
func split(r rune) bool {
	return r == '\n' || r == ','
}

func (oa *OCIArtifact) Type() string {
	return "oci"
}

func (oa *OCIArtifact) StorageBackend(cfg config.Config) sets.String {
	return cfg.Artifacts.OCI.StorageBackend
}

func (oa *OCIArtifact) PayloadFormat(cfg config.Config) formats.PayloadType {
	return formats.PayloadType(cfg.Artifacts.OCI.Format)
}

func (oa *OCIArtifact) Signer(cfg config.Config) string {
	return cfg.Artifacts.OCI.Signer
}

func (oa *OCIArtifact) Key(obj interface{}) string {
	v := obj.(name.Digest)
	return strings.TrimPrefix(v.DigestStr(), "sha256:")[:12]
}

func (oa *OCIArtifact) Enabled(cfg config.Config) bool {
	return cfg.Artifacts.OCI.Enabled()
}
