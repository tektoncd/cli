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
	slsa "github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/v0.2"
	"github.com/opencontainers/go-digest"
	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/config"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ArtifactsInputsResultName  = "ARTIFACT_INPUTS"
	ArtifactsOutputsResultName = "ARTIFACT_OUTPUTS"
)

type Signable interface {
	ExtractObjects(obj objects.TektonObject) []interface{}
	StorageBackend(cfg config.Config) sets.String
	Signer(cfg config.Config) string
	PayloadFormat(cfg config.Config) config.PayloadType
	// FullKey returns the full identifier for a signable artifact.
	// - For OCI artifact, it is the full representation in the format of `<NAME>@sha256:<DIGEST>`.
	// - For TaskRun/PipelineRun artifact, it is `<GROUP>-<VERSION>-<KIND>-<UID>`
	FullKey(interface{}) string
	// ShortKey returns the short version  of an artifact identifier.
	// - For OCI artifact, it is first 12 chars of the image digest.
	// - For TaskRun/PipelineRun artifact, it is `<KIND>-<UID>`.
	ShortKey(interface{}) string
	Type() string
	Enabled(cfg config.Config) bool
}

type TaskRunArtifact struct {
	Logger *zap.SugaredLogger
}

var _ Signable = &TaskRunArtifact{}

func (ta *TaskRunArtifact) ShortKey(obj interface{}) string {
	tro := obj.(*objects.TaskRunObject)
	return "taskrun-" + string(tro.UID)
}

func (ta *TaskRunArtifact) FullKey(obj interface{}) string {
	tro := obj.(*objects.TaskRunObject)
	gvk := tro.GetGroupVersionKind()
	return fmt.Sprintf("%s-%s-%s-%s", gvk.Group, gvk.Version, gvk.Kind, tro.UID)
}

func (ta *TaskRunArtifact) ExtractObjects(obj objects.TektonObject) []interface{} {
	return []interface{}{obj}
}

func (ta *TaskRunArtifact) Type() string {
	return "tekton"
}

func (ta *TaskRunArtifact) StorageBackend(cfg config.Config) sets.String {
	return cfg.Artifacts.TaskRuns.StorageBackend
}

func (ta *TaskRunArtifact) PayloadFormat(cfg config.Config) config.PayloadType {
	return config.PayloadType(cfg.Artifacts.TaskRuns.Format)
}

func (ta *TaskRunArtifact) Signer(cfg config.Config) string {
	return cfg.Artifacts.TaskRuns.Signer
}

func (ta *TaskRunArtifact) Enabled(cfg config.Config) bool {
	return cfg.Artifacts.TaskRuns.Enabled()
}

type PipelineRunArtifact struct {
	Logger *zap.SugaredLogger
}

var _ Signable = &PipelineRunArtifact{}

func (pa *PipelineRunArtifact) ShortKey(obj interface{}) string {
	pro := obj.(*objects.PipelineRunObject)
	return "pipelinerun-" + string(pro.UID)
}

func (pa *PipelineRunArtifact) FullKey(obj interface{}) string {
	pro := obj.(*objects.PipelineRunObject)
	gvk := pro.GetGroupVersionKind()
	return fmt.Sprintf("%s-%s-%s-%s", gvk.Group, gvk.Version, gvk.Kind, pro.UID)
}

func (pa *PipelineRunArtifact) ExtractObjects(obj objects.TektonObject) []interface{} {
	return []interface{}{obj}
}

func (pa *PipelineRunArtifact) Type() string {
	// TODO: Is this right?
	return "tekton-pipeline-run"
}

func (pa *PipelineRunArtifact) StorageBackend(cfg config.Config) sets.String {
	return cfg.Artifacts.PipelineRuns.StorageBackend
}

func (pa *PipelineRunArtifact) PayloadFormat(cfg config.Config) config.PayloadType {
	return config.PayloadType(cfg.Artifacts.PipelineRuns.Format)
}

func (pa *PipelineRunArtifact) Signer(cfg config.Config) string {
	return cfg.Artifacts.PipelineRuns.Signer
}

func (pa *PipelineRunArtifact) Enabled(cfg config.Config) bool {
	return cfg.Artifacts.PipelineRuns.Enabled()
}

type OCIArtifact struct {
	Logger *zap.SugaredLogger
}

var _ Signable = &OCIArtifact{}

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

func (oa *OCIArtifact) ExtractObjects(obj objects.TektonObject) []interface{} {
	objs := []interface{}{}

	// TODO: Not applicable to PipelineRuns, should look into a better way to separate this out
	if tr, ok := obj.GetObject().(*v1beta1.TaskRun); ok {
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

		for _, image := range imageResourceNames {
			dgst, err := name.NewDigest(fmt.Sprintf("%s@%s", image.url, image.digest))
			if err != nil {
				oa.Logger.Error(err)
				continue
			}
			objs = append(objs, dgst)
		}
	}

	// Now check TaskResults
	resultImages := ExtractOCIImagesFromResults(obj, oa.Logger)
	objs = append(objs, resultImages...)

	return objs
}

func ExtractOCIImagesFromResults(obj objects.TektonObject, logger *zap.SugaredLogger) []interface{} {
	objs := []interface{}{}
	ss := extractTargetFromResults(obj, "IMAGE_URL", "IMAGE_DIGEST", logger)
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
	for _, key := range obj.GetResults() {
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
func ExtractSignableTargetFromResults(obj objects.TektonObject, logger *zap.SugaredLogger) []*StructuredSignable {
	objs := []*StructuredSignable{}
	ss := extractTargetFromResults(obj, "ARTIFACT_URI", "ARTIFACT_DIGEST", logger)
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

// FullRef returns the full reference of the signable artifact in the format of URI@DIGEST
func (s *StructuredSignable) FullRef() string {
	return fmt.Sprintf("%s@%s", s.URI, s.Digest)
}

func extractTargetFromResults(obj objects.TektonObject, identifierSuffix string, digestSuffix string, logger *zap.SugaredLogger) map[string]*StructuredSignable {
	ss := map[string]*StructuredSignable{}

	for _, res := range obj.GetResults() {
		if strings.HasSuffix(res.Name, identifierSuffix) {
			if res.Value.StringVal == "" {
				logger.Debugf("error getting string value for %s", res.Name)
				continue
			}
			marker := strings.TrimSuffix(res.Name, identifierSuffix)
			if v, ok := ss[marker]; ok {
				v.URI = strings.TrimSpace(res.Value.StringVal)

			} else {
				ss[marker] = &StructuredSignable{URI: strings.TrimSpace(res.Value.StringVal)}
			}
			// TODO: add logic for Intoto signable target as input.
		}
		if strings.HasSuffix(res.Name, digestSuffix) {
			if res.Value.StringVal == "" {
				logger.Debugf("error getting string value for %s", res.Name)
				continue
			}
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

// RetrieveMaterialsFromStructuredResults retrieves structured results from Tekton Object, and convert them into materials.
func RetrieveMaterialsFromStructuredResults(obj objects.TektonObject, categoryMarker string, logger *zap.SugaredLogger) []slsa.ProvenanceMaterial {
	// Retrieve structured provenance for inputs.
	mats := []slsa.ProvenanceMaterial{}
	ssts := ExtractStructuredTargetFromResults(obj, ArtifactsInputsResultName, logger)
	for _, s := range ssts {
		if err := checkDigest(s.Digest); err != nil {
			logger.Debugf("Digest for %s not in the right format: %s, %v", s.URI, s.Digest, err)
			continue
		}
		splits := strings.Split(s.Digest, ":")
		alg := splits[0]
		digest := splits[1]
		mats = append(mats, slsa.ProvenanceMaterial{
			URI:    s.URI,
			Digest: map[string]string{alg: digest},
		})
	}
	return mats
}

// ExtractStructuredTargetFromResults extracts structured signable targets aim to generate intoto provenance as materials within TaskRun results and store them as StructuredSignable.
// categoryMarker categorizes signable targets into inputs and outputs.
func ExtractStructuredTargetFromResults(obj objects.TektonObject, categoryMarker string, logger *zap.SugaredLogger) []*StructuredSignable {
	objs := []*StructuredSignable{}
	if categoryMarker != ArtifactsInputsResultName && categoryMarker != ArtifactsOutputsResultName {
		return objs
	}

	// TODO(#592): support structured results using Run
	results := []objects.Result{}
	for _, res := range obj.GetResults() {
		results = append(results, objects.Result{
			Name:  res.Name,
			Value: res.Value,
		})
	}
	for _, res := range results {
		if strings.HasSuffix(res.Name, categoryMarker) {
			valid, err := isStructuredResult(res, categoryMarker)
			if err != nil {
				logger.Debugf("ExtractStructuredTargetFromResults: %v", err)
			}
			if valid {
				objs = append(objs, &StructuredSignable{URI: res.Value.ObjectVal["uri"], Digest: res.Value.ObjectVal["digest"]})
			}
		}
	}
	return objs
}

func isStructuredResult(res objects.Result, categoryMarker string) (bool, error) {
	if !strings.HasSuffix(res.Name, categoryMarker) {
		return false, nil
	}
	if res.Value.ObjectVal == nil {
		return false, fmt.Errorf("%s should be an object: %v", res.Name, res.Value.ObjectVal)
	}
	if res.Value.ObjectVal["uri"] == "" {
		return false, fmt.Errorf("%s should have uri field: %v", res.Name, res.Value.ObjectVal)
	}
	if res.Value.ObjectVal["digest"] == "" {
		return false, fmt.Errorf("%s should have digest field: %v", res.Name, res.Value.ObjectVal)
	}
	if err := checkDigest(res.Value.ObjectVal["digest"]); err != nil {
		return false, fmt.Errorf("error getting digest %s: %v", res.Value.ObjectVal["digest"], err)
	}
	return true, nil
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

func (oa *OCIArtifact) PayloadFormat(cfg config.Config) config.PayloadType {
	return config.PayloadType(cfg.Artifacts.OCI.Format)
}

func (oa *OCIArtifact) Signer(cfg config.Config) string {
	return cfg.Artifacts.OCI.Signer
}

func (oa *OCIArtifact) ShortKey(obj interface{}) string {
	v := obj.(name.Digest)
	return strings.TrimPrefix(v.DigestStr(), "sha256:")[:12]
}

func (oa *OCIArtifact) FullKey(obj interface{}) string {
	v := obj.(name.Digest)
	return v.Name()
}

func (oa *OCIArtifact) Enabled(cfg config.Config) bool {
	return cfg.Artifacts.OCI.Enabled()
}
