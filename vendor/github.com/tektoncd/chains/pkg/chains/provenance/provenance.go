/*
Copyright 2021` The Tekton Authors
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

import "time"

/*
DigestSet contains a set of digests. It is represented as a map from
algorithm name to lowercase hex-encoded value.
*/
type DigestSet map[string]string

// ProvenancePredicate is the provenance predicate definition.
type ProvenancePredicate struct {
	// removed: ProvenanceBuilder
	Invocation Invocation           `json:"invocation"`
	Recipe     ProvenanceRecipe     `json:"recipe"`
	Metadata   ProvenanceMetadata   `json:"metadata"`
	Materials  []ProvenanceMaterial `json:"materials,omitempty"`
}

// Invocation describes how the Taskrun was created
type Invocation struct {
	Parameters interface{} `json:"parameters"`
	// This would be nil for an "inline task", a URI for the Task definition itself
	// if it was created in-cluster, or an OCI uri if it came from OCI
	// Something else if it came from a pipeline
	RecipeURI string `json:"recipe_uri"`
	EventID   string `json:"event_id"`
	ID        string `json:"builder.id"`
}

// ProvenanceRecipe describes the actions performed by the builder.
type ProvenanceRecipe struct {
	Steps []RecipeStep `json:"steps,omitempty"`
}

type RecipeStep struct {
	EntryPoint  string            `json:"entryPoint"`
	Arguments   interface{}       `json:"arguments,omitempty"`
	Environment interface{}       `json:"environment,omitempty"`
	Annotations map[string]string `json:"annotations"`
}

// ProvenanceMetadata contains metadata for the built artifact.
type ProvenanceMetadata struct {
	// Use pointer to make sure that the abscense of a time is not
	// encoded as the Epoch time.
	BuildStartedOn  *time.Time `json:"buildStartedOn,omitempty"`
	BuildFinishedOn *time.Time `json:"buildFinishedOn,omitempty"`
	// removed: Completeness
	Reproducible bool `json:"reproducible,omitempty"`
}

// ProvenanceMaterial defines the materials used to build an artifact.
type ProvenanceMaterial struct {
	URI    string    `json:"uri"`
	Digest DigestSet `json:"digest,omitempty"`
}
