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

package tekton

import (
	"fmt"

	"github.com/tektoncd/chains/pkg/chains/formats"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// Tekton is a formatter that just captures the TaskRun Status with no modifications.
type Tekton struct {
}

func NewFormatter() (formats.Payloader, error) {
	return &Tekton{}, nil
}

// CreatePayload implements the Payloader interface.
func (i *Tekton) CreatePayload(obj interface{}) (interface{}, error) {

	switch v := obj.(type) {
	case *v1beta1.TaskRun:
		return v.Status, nil
	default:
		return nil, fmt.Errorf("unsupported type %s", v)
	}

}

func (i *Tekton) Type() formats.PayloadType {
	return formats.PayloadTypeTekton
}

func (i *Tekton) Wrap() bool {
	return false
}
