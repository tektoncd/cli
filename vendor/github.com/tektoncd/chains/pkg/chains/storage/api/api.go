// Copyright 2023 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"context"

	"github.com/tektoncd/chains/pkg/chains/objects"
	"github.com/tektoncd/chains/pkg/chains/signing"
)

// StoreRequest contains the information needed to store a signature/attestation object.
type StoreRequest[Input any, Output any] struct {
	// Object is the original Tekton object as received by the controller.
	Object objects.TektonObject
	// Artifact is the artifact that was extracted from the Object (e.g. build config, image, etc.)
	Artifact Input
	// Payload is the formatted payload that was generated for the Artifact (e.g. simplesigning, in-toto attestation)
	Payload Output
	// Bundle contains the signing output details.
	Bundle *signing.Bundle
}

// StoreResponse contains metadata for the result of the store operation.
type StoreResponse struct {
	// currently empty, but may contain data in the future.
	// present to allow for backwards compatible changes to the Storer interface in the future.
}

type Storer[Input, Output any] interface {
	Store(context.Context, *StoreRequest[Input, Output]) (*StoreResponse, error)
}
