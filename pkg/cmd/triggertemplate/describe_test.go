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

package triggertemplate

import (
	"fmt"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	triggertest "github.com/tektoncd/triggers/test"
	tt "github.com/tektoncd/triggers/test/builder"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var simpleResourceTemplate = runtime.RawExtension{
	Raw: []byte(`{"kind":"PipelineRun","apiVersion":"tekton.dev/v1alpha1","metadata":{"generateName":"ex2", "creationTimestamp":null},"spec":{},"status":{}}`),
}
var pipelineResources = runtime.RawExtension{
	Raw: []byte(`{"kind":"PipelineResource","apiVersion":"tekton.dev/v1alpha1","metadata":{"name":"ex1", "creationTimestamp":null},"spec":{},"status":{}}`),
}
var v1beta1ResourceTemplate = runtime.RawExtension{
	Raw: []byte(`{"kind":"PipelineRun","apiVersion":"tekton.dev/v1beta1","metadata":{"name":"ex1", "generateName":"ex2", "creationTimestamp":null},"spec":{},"status":{}}`),
}
var paramResourceTemplate = runtime.RawExtension{
	Raw: []byte(`{"kind":"PipelineRun","apiVersion":"tekton.dev/v1alpha1","metadata":{"name":"ex1", "creationTimestamp":null},"spec": "$(params.foo)","status":{}}`),
}
var invalidTemplate = runtime.RawExtension{
	Raw: []byte(`("kind":"InvalidKind")`),
}

func TestTriggerTemplateDescribe_Invalid_Namespace(t *testing.T) {
	cs := test.SeedTestResources(t, triggertest.Resources{})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerTemplate := Command(p)
	out, err := test.ExecuteCommand(triggerTemplate, "desc", "bar", "-n", "invalid")
	if err == nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTriggerTemplateDescribe_NonExistedName(t *testing.T) {
	cs := test.SeedTestResources(t, triggertest.Resources{Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerTemplate := Command(p)
	out, err := test.ExecuteCommand(triggerTemplate, "desc", "bar", "-n", "ns")
	if err == nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTriggerTemplateDescribe_Empty(t *testing.T) {
	cs := test.SeedTestResources(t, triggertest.Resources{Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerTemplate := Command(p)
	out, err := test.ExecuteCommand(triggerTemplate, "desc", "-n", "ns")
	if err == nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTriggerTemplateDescribe_NoParams(t *testing.T) {
	tts := []*v1alpha1.TriggerTemplate{
		tt.TriggerTemplate("tt1", "ns",
			tt.TriggerTemplateSpec(
				tt.TriggerResourceTemplate(simpleResourceTemplate))),
	}
	executeTriggerTemplateCommand(t, tts)
}

func TestTriggerTemplateDescribe_WithOneParam(t *testing.T) {
	tts := []*v1alpha1.TriggerTemplate{
		tt.TriggerTemplate("tt1", "ns",
			tt.TriggerTemplateSpec(
				tt.TriggerTemplateParam("key", "test with one param", "value"),
				tt.TriggerResourceTemplate(simpleResourceTemplate))),
	}
	executeTriggerTemplateCommand(t, tts)
}

func TestTriggerTemplateDescribe_WithOutputName(t *testing.T) {
	tts := []*v1alpha1.TriggerTemplate{
		tt.TriggerTemplate("tt1", "ns",
			tt.TriggerTemplateSpec(
				tt.TriggerTemplateParam("key", "test with one param", "value"),
				tt.TriggerResourceTemplate(simpleResourceTemplate))),
	}
	cs := test.SeedTestResources(t, triggertest.Resources{TriggerTemplates: tts, Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerTemplate := Command(p)
	out, err := test.ExecuteCommand(triggerTemplate, "desc", "tt1", "-o", "name", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTriggerTemplateDescribe_WithOutputYaml(t *testing.T) {
	tts := []*v1alpha1.TriggerTemplate{
		tt.TriggerTemplate("tt1", "ns",
			tt.TriggerTemplateSpec(
				tt.TriggerTemplateParam("key", "test with one param", "value"),
				tt.TriggerResourceTemplate(simpleResourceTemplate))),
	}
	cs := test.SeedTestResources(t, triggertest.Resources{TriggerTemplates: tts, Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerTemplate := Command(p)
	out, err := test.ExecuteCommand(triggerTemplate, "desc", "tt1", "-o", "yaml", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func TestTriggerTemplateDescribe_WithMultipleParams(t *testing.T) {
	tts := []*v1alpha1.TriggerTemplate{
		tt.TriggerTemplate("tt1", "ns",
			tt.TriggerTemplateSpec(
				tt.TriggerTemplateParam("key1", "test with multiple param", "value1"),
				tt.TriggerTemplateParam("key2", "", "value2"),
				tt.TriggerTemplateParam("key3", "", "value3"),
				tt.TriggerTemplateParam("key4", "test with multiple param", "value4"),
				tt.TriggerResourceTemplate(v1beta1ResourceTemplate))),
	}
	executeTriggerTemplateCommand(t, tts)
}

func TestTriggerTemplateDescribe_WithMultipleResourceTemplate(t *testing.T) {
	tts := []*v1alpha1.TriggerTemplate{
		tt.TriggerTemplate("tt1", "ns",
			tt.TriggerTemplateSpec(
				tt.TriggerTemplateParam("key1", "test with multiple param", "value1"),
				tt.TriggerTemplateParam("key2", "", "value2"),
				tt.TriggerTemplateParam("key3", "", "value3"),
				tt.TriggerTemplateParam("key4", "test with multiple param", "value4"),
				tt.TriggerResourceTemplate(v1beta1ResourceTemplate),
				tt.TriggerResourceTemplate(pipelineResources))),
	}
	executeTriggerTemplateCommand(t, tts)
}

func TestTriggerTemplateDescribe_ParamsToResourceTemplate(t *testing.T) {
	tts := []*v1alpha1.TriggerTemplate{
		tt.TriggerTemplate("tt1", "ns",
			tt.TriggerTemplateSpec(
				tt.TriggerTemplateParam("foo", "foo required in resource template", "bar"),
				tt.TriggerResourceTemplate(paramResourceTemplate))),
	}
	executeTriggerTemplateCommand(t, tts)
}

func TestTriggerTemplateDescribe_InvalidResourceTemplate(t *testing.T) {
	tts := []*v1alpha1.TriggerTemplate{
		tt.TriggerTemplate("tt1", "ns",
			tt.TriggerTemplateSpec(
				tt.TriggerTemplateParam("foo", "foo required in resource template", "bar"),
				tt.TriggerResourceTemplate(invalidTemplate))),
	}
	executeTriggerTemplateCommand(t, tts)
}

func TestTriggerTemplateDescribe_AutoSelect(t *testing.T) {

	tts := []*v1alpha1.TriggerTemplate{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "tt1",
				Namespace: "ns",
			},
			Spec: v1alpha1.TriggerTemplateSpec{
				Params: []v1alpha1.ParamSpec{
					{
						Name:        "foo",
						Description: "foo is required in resource template",
					},
				},
				ResourceTemplates: []v1alpha1.TriggerResourceTemplate{
					{
						RawExtension: paramResourceTemplate,
					},
				},
			},
		},
	}

	cs := test.SeedTestResources(t, triggertest.Resources{TriggerTemplates: tts, Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggertemplate := Command(p)
	out, err := test.ExecuteCommand(triggertemplate, "desc", "-n", "ns")
	if err != nil {
		t.Errorf("Error expected here")
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}

func executeTriggerTemplateCommand(t *testing.T, tts []*v1alpha1.TriggerTemplate) {
	cs := test.SeedTestResources(t, triggertest.Resources{TriggerTemplates: tts, Namespaces: []*corev1.Namespace{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}}})
	p := &test.Params{Triggers: cs.Triggers, Kube: cs.Kube}

	triggerTemplate := Command(p)
	out, err := test.ExecuteCommand(triggerTemplate, "desc", "tt1", "-n", "ns")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	golden.Assert(t, out, fmt.Sprintf("%s.golden", t.Name()))
}
