/*
Copyright 2023 The Tekton Authors

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
package artifact

import (
	intoto "github.com/in-toto/attestation/go/v1"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
)

// AppendSubjects adds new subject(s) to the original slice.
// It merges the new item with an existing entry if they are duplicate instead of append.
func AppendSubjects(original []*intoto.ResourceDescriptor, items ...*intoto.ResourceDescriptor) []*intoto.ResourceDescriptor {
	var artifacts []artifact
	for _, s := range original {
		artifacts = append(artifacts, subjectToArtifact(s))
	}

	for _, s := range items {
		artifacts = addArtifact(artifacts, subjectToArtifact(s))
	}

	var result []*intoto.ResourceDescriptor
	for _, a := range artifacts {
		result = append(result, artifactToSubject(a))
	}
	return result
}

// AppendMaterials adds new material(s) to the original slice.
// It merges the new item with an existing entry if they are duplicate instead of append.
func AppendMaterials(original []common.ProvenanceMaterial, items ...common.ProvenanceMaterial) []common.ProvenanceMaterial {
	var artifacts []artifact
	for _, m := range original {
		artifacts = append(artifacts, materialToArtifact(m))
	}

	for _, m := range items {
		artifacts = addArtifact(artifacts, materialToArtifact(m))
	}

	var result []common.ProvenanceMaterial
	for _, a := range artifacts {
		result = append(result, artifactToMaterial(a))
	}
	return result
}

type artifact struct {
	name      string
	digestSet map[string]string
}

// AddArtifact adds a new artifact item to the original slice.
func addArtifact(original []artifact, item artifact) []artifact {

	for i, a := range original {
		// if there is an equivalent entry in the original slice, merge the
		// artifact's DigestSet into the existing entry's DigestSet.
		if artifactEqual(a, item) {
			mergeMaps(original[i].digestSet, item.digestSet)
			return original
		}
	}

	original = append(original, item)
	return original
}

// two artifacts are equal if and only if they have same name and have at least
// one common algorithm and hex value.
func artifactEqual(x, y artifact) bool {
	if x.name != y.name {
		return false
	}
	for algo, hex := range x.digestSet {
		if y.digestSet[algo] == hex {
			return true
		}
	}
	return false
}

func mergeMaps(m1 map[string]string, m2 map[string]string) {
	for k, v := range m2 {
		m1[k] = v
	}
}

func subjectToArtifact(s *intoto.ResourceDescriptor) artifact {
	return artifact{
		name:      s.Name,
		digestSet: s.Digest,
	}
}

func artifactToSubject(a artifact) *intoto.ResourceDescriptor {
	return &intoto.ResourceDescriptor{
		Name:   a.name,
		Digest: a.digestSet,
	}
}

func materialToArtifact(m common.ProvenanceMaterial) artifact {
	return artifact{
		name:      m.URI,
		digestSet: m.Digest,
	}
}

func artifactToMaterial(a artifact) common.ProvenanceMaterial {
	return common.ProvenanceMaterial{
		URI:    a.name,
		Digest: a.digestSet,
	}
}
