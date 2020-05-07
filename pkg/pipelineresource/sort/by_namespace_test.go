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

func Test_PipelineResourcesByNamespace(t *testing.T) {
	pres01 := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "abc",
			Name:      "pres0-1",
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "cloudEvent",
			Params: []v1alpha1.ResourceParam{
				{Name: "targetURI", Value: "http://sink"},
			},
		},
	}

	pres02 := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "abc",
			Name:      "pres0-2",
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "git",
			Params: []v1alpha1.ResourceParam{
				{Name: "le clé", Value: "git@github.com:tektoncd/cli-new.git"},
			},
		},
	}

	pres03 := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "abc",
			Name:      "pres0-3",
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "image",
			Params: []v1alpha1.ResourceParam{
				{Name: "trolla", Value: "quey.io/tekton/controller"},
			},
		},
	}

	pres11 := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "def",
			Name:      "pres1-1",
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "cloudEvent",
			Params: []v1alpha1.ResourceParam{
				{Name: "justUri", Value: "http://sunk"},
			},
		},
	}

	pres12 := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "def",
			Name:      "pres1-2",
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "git",
			Params: []v1alpha1.ResourceParam{
				{Name: "claè nouveau", Value: "git@github.com:tektoncd/cli-new.git"},
			},
		},
	}

	pres13 := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "def",
			Name:      "pres1-3",
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "image",
			Params: []v1alpha1.ResourceParam{
				{Name: "ouarel", Value: "nay.io/tekton/trolley"},
			},
		},
	}

	pres21 := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ghi",
			Name:      "pres2-1",
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "cloudEvent",
			Params: []v1alpha1.ResourceParam{
				{Name: "targetURI", Value: "http://sink"},
			},
		},
	}

	pres22 := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ghi",
			Name:      "pres2-2",
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "git",
			Params: []v1alpha1.ResourceParam{
				{Name: "entrée", Value: "git@github.com:tektoncd/entrypoint.git"},
			},
		},
	}

	pres23 := v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ghi",
			Name:      "pres2-3",
		},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: "image",
			Params: []v1alpha1.ResourceParam{
				{Name: "ay", Value: "docker.io/not-jenkins/holley"},
			},
		},
	}

	pres := []v1alpha1.PipelineResource{
		pres12,
		pres13,
		pres22,
		pres11,
		pres23,
		pres02,
		pres21,
		pres03,
		pres01,
	}

	SortByNamespace(pres)

	element1 := pres[0].Name
	if element1 != "pres0-1" {
		t.Errorf("SortPipelineResourcesByNamespace should be pres0-1 but returned: %s", element1)
	}

	element2 := pres[1].Name
	if element2 != "pres0-2" {
		t.Errorf("SortPipelineResourcesByNamespace should be pres0-2 but returned: %s", element2)
	}

	element3 := pres[2].Name
	if element3 != "pres0-3" {
		t.Errorf("SortPipelineResourcesByNamespace should be pres0-3 but returned: %s", element3)
	}

	element4 := pres[3].Name
	if element4 != "pres1-1" {
		t.Errorf("SortPipelineResourcesByNamespace should be pres1-1 but returned: %s", element4)
	}

	element5 := pres[4].Name
	if element5 != "pres1-2" {
		t.Errorf("SortPipelineResourcesByNamespace should be pres1-2 but returned: %s", element5)
	}

	element6 := pres[5].Name
	if element6 != "pres1-3" {
		t.Errorf("SortPipelineResourcesByNamespace should be pres1-3 but returned: %s", element6)
	}

	element7 := pres[6].Name
	if element7 != "pres2-1" {
		t.Errorf("SortPipelineResourcesByNamespace should be pres2-1 but returned: %s", element7)
	}

	element8 := pres[7].Name
	if element8 != "pres2-2" {
		t.Errorf("SortPipelineResourcesByNamespace should be pres2-2 but returned: %s", element8)
	}

	element9 := pres[8].Name
	if element9 != "pres2-3" {
		t.Errorf("SortPipelineResourcesByNamespace should be pres2-3 but returned: %s", element9)
	}
}
