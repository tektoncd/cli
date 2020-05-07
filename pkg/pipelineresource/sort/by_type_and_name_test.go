// Copyright © 2020 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pipelineresource

import (
	"testing"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_PipelineResourcesByTypeAndName(t *testing.T) {
	pres1 := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "abc",
			Name:      "pres0-1",
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "git",
			Params: []v1alpha1.ResourceParam{
				{Name: "url", Value: "git@github.com:tektoncd/cli-new.git"},
			},
		},
	}

	pres2 := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "abc",
			Name:      "pres1-1",
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "image",
			Params: []v1alpha1.ResourceParam{
				{Name: "URL", Value: "quey.io/tekton/controller"},
			},
		},
	}

	pres3 := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "abc",
			Name:      "pres2-1",
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "cloudEvent",
			Params: []v1alpha1.ResourceParam{
				{Name: "targetURI", Value: "http://sink"},
			},
		},
	}

	pres := []v1alpha1.PipelineResource{
		pres2,
		pres3,
		pres1,
	}

	SortByTypeAndName(pres)

	element1Type := pres[0].Spec.Type
	if element1Type != "cloudEvent" {
		t.Errorf("SortPipelineResourcesByTypeAndName should be cloudEvent but returned: %s", element1Type)
	}

	element2Type := pres[1].Spec.Type
	if element2Type != "git" {
		t.Errorf("SortPipelineResourcesByTypeAndName should be git but returned: %s", element2Type)
	}

	element3Type := pres[2].Spec.Type
	if element3Type != "image" {
		t.Errorf("SortPipelineResourcesByTypeAndName should be image but returned: %s", element3Type)
	}
}
