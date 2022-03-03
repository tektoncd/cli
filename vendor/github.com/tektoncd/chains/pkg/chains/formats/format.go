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

package formats

// Payloader is an interface to generate a chains Payload from a TaskRun
type Payloader interface {
	CreatePayload(obj interface{}) (interface{}, error)
	Type() PayloadType
	Wrap() bool
}

type PayloadType string

// If you update this, remember to update AllFormatters
const (
	PayloadTypeTekton        PayloadType = "tekton"
	PayloadTypeSimpleSigning PayloadType = "simplesigning"
	PayloadTypeInTotoIte6    PayloadType = "in-toto"
	PayloadTypeProvenance    PayloadType = "tekton-provenance"
)

var AllFormatters = []PayloadType{PayloadTypeTekton, PayloadTypeSimpleSigning, PayloadTypeInTotoIte6, PayloadTypeProvenance}
