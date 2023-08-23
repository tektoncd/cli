// Copyright © 2019 The Tekton Authors.
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

package pipeline

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/pipeline"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/cli/test/prompt"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	fakepipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	pipelinetest "github.com/tektoncd/pipeline/test"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stest "k8s.io/client-go/testing"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func newV1beta1PipelineClient(objs ...runtime.Object) (*fakepipelineclientset.Clientset, testDynamic.Options) {
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	localSchemeBuilder := runtime.SchemeBuilder{v1beta1.AddToScheme}

	metav1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	util.Must(localSchemeBuilder.AddToScheme(scheme))
	o := k8stest.NewObjectTracker(scheme, codecs.UniversalDecoder())
	for _, obj := range objs {
		if err := o.Add(obj); err != nil {
			panic(err)
		}
	}

	dc := testDynamic.Options{
		AddReactorRes:  "*",
		AddReactorVerb: "*",
		AddReactorFun:  k8stest.ObjectReaction(o),
		WatchResource:  "*",
		WatchReactionFun: func(action k8stest.Action) (handled bool, ret watch.Interface, err error) {
			gvr := action.GetResource()
			ns := action.GetNamespace()
			watch, err := o.Watch(gvr, ns)
			if err != nil {
				return false, nil, err
			}
			return true, watch, nil
		},
		PrependReactors: []testDynamic.PrependOpt{
			{
				Resource: "pipelineruns",
				Verb:     "create",
				Action: func(action k8stest.Action) (bool, runtime.Object, error) {
					create := action.(k8stest.CreateActionImpl)
					unstructuredPR := create.GetObject().(*unstructured.Unstructured)
					unstructuredPR.SetName("random")
					rFunc := k8stest.ObjectReaction(o)
					_, o, err := rFunc(action)
					return true, o, err
				},
			},
			{
				Resource: "pipelineruns",
				Verb:     "get",
				Action: func(action k8stest.Action) (bool, runtime.Object, error) {
					getAction, _ := action.(k8stest.GetActionImpl)
					res := getAction.GetResource()
					ns := getAction.GetNamespace()
					name := getAction.GetName()
					obj, err := o.Get(res, ns, name)
					if err != nil {
						return false, nil, err
					}
					if reflect.TypeOf(obj).String() == "*unstructured.Unstructured" {
						return true, obj, nil
					}

					v1beta1PR := obj.(*v1beta1.PipelineRun)
					unstructuredPR := cb.UnstructuredV1beta1PR(v1beta1PR, versionv1beta1)
					return true, unstructuredPR, nil
				},
			},
		},
	}
	return nil, dc
}

func TestPipelineStart_ExecuteCommand_v1beta1(t *testing.T) {
	clock := test.FakeClock()
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{})
	cs.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c1 := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Clock: clock}

	pipeline := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pipeline",
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "unit-test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "unit-test-task",
						},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision",
						},
					},
				},
			},
		},
	}

	pipeline2 := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pipeline",
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "unit-test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "unit-test-task",
						},
					},
				},
			},
		},
	}

	cs2, _ := test.SeedV1beta1TestData(t, test.Data{Pipelines: pipeline, Namespaces: namespaces})
	cs2.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1P(pipeline[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c2 := &test.Params{Tekton: cs2.Pipeline, Kube: cs2.Kube, Dynamic: dc2, Clock: clock}

	// With list error mocking
	cs3, _ := test.SeedV1beta1TestData(t, test.Data{Pipelines: pipeline, Namespaces: namespaces})
	cs3.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	tdc3 := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{
			{
				Resource: "pipelineruns",
				Verb:     "list",
				Action: func(_ k8stest.Action) (bool, runtime.Object, error) {
					return true, nil, fmt.Errorf("test generated error")
				},
			},
		},
	}
	dc3, err := tdc3.Client(
		cb.UnstructuredV1beta1P(pipeline[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c3 := &test.Params{Tekton: cs3.Pipeline, Kube: cs3.Kube, Dynamic: dc3, Clock: clock}

	// With create error mocking
	cs4, _ := test.SeedV1beta1TestData(t, test.Data{Pipelines: pipeline2, Namespaces: namespaces})
	cs4.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	tdc4 := testDynamic.Options{
		PrependReactors: []testDynamic.PrependOpt{
			{
				Resource: "pipelineruns",
				Verb:     "create",
				Action: func(_ k8stest.Action) (bool, runtime.Object, error) {
					return true, nil, fmt.Errorf("mock error")
				},
			},
		},
	}
	dc4, err := tdc4.Client(
		cb.UnstructuredV1beta1P(pipeline2[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c4 := &test.Params{Tekton: cs4.Pipeline, Kube: cs4.Kube, Dynamic: dc4, Clock: clock}

	// Without related pipelinerun
	objs := []runtime.Object{
		pipeline[0],
	}
	seedData, _ := test.SeedV1beta1TestData(t, test.Data{
		Pipelines:  pipeline,
		Namespaces: namespaces,
	})
	cs5 := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs5.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	_, tdc5 := newV1beta1PipelineClient(objs...)
	dc5, err := tdc5.Client(
		cb.UnstructuredV1beta1P(pipeline[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c5 := &test.Params{Tekton: cs5.Pipeline, Kube: cs5.Kube, Dynamic: dc5, Clock: clock}

	// pipelineresources data for tests with --filename
	objs2 := []runtime.Object{}
	seedData2, _ := test.SeedV1beta1TestData(t, test.Data{
		Namespaces: namespaces,
	})
	cs6 := pipelinetest.Clients{
		Pipeline: seedData2.Pipeline,
		Kube:     seedData2.Kube,
	}
	cs6.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	_, tdc6 := newV1beta1PipelineClient(objs2...)
	dc6, err := tdc6.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c6 := &test.Params{Tekton: cs6.Pipeline, Kube: cs6.Kube, Dynamic: dc6, Clock: clock}

	cs7, _ := test.SeedV1beta1TestData(t, test.Data{Pipelines: pipeline, Namespaces: namespaces})
	cs7.Pipeline.Resources = cb.APIResourceList(versionv1beta1, []string{"pipeline", "pipelinerun"})
	tdc7 := testDynamic.Options{}
	dc7, err := tdc7.Client(
		cb.UnstructuredV1beta1P(pipeline[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c7 := &test.Params{Tekton: cs7.Pipeline, Kube: cs7.Kube, Dynamic: dc7, Clock: clock}

	testParams := []struct {
		name       string
		command    []string
		namespace  string
		input      *test.Params
		wantError  bool
		hasPrefix  bool
		want       string
		goldenFile bool
	}{
		{
			name:      "Invalid namespace",
			command:   []string{"start", "pipeline", "-n", "invalid"},
			namespace: "",
			input:     c1,
			wantError: true,
			want:      "Pipeline name pipeline does not exist in namespace invalid",
		},
		{
			name:      "Missing pipeline name",
			command:   []string{"start", "-n", "ns"},
			namespace: "",
			input:     c1,
			wantError: true,
			want:      "missing Pipeline name",
		},
		{
			name:      "Found no pipelines",
			command:   []string{"start", "test-pipeline-2", "-n", "ns"},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "Pipeline name test-pipeline-2 does not exist in namespace ns",
		},
		{
			name: "Start pipeline with showlog flag false",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-w=name=password-vault,secret=secret-name",
				"-n", "ns",
			},
			namespace: "",
			input:     c2,
			wantError: false,
			want:      "PipelineRun started: \n\nIn order to track the PipelineRun progress run:\ntkn pipelinerun logs  -f -n ns\n",
		},
		{
			name: "Start pipeline with different context",
			command: []string{
				"start", "test-pipeline",
				"--context=GummyBear",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-w=name=password-vault,secret=secret-name",
				"-n", "ns",
			},
			namespace: "",
			input:     c7,
			wantError: false,
			want:      "PipelineRun started: \n\nIn order to track the PipelineRun progress run:\ntkn pipelinerun --context=GummyBear logs  -f -n ns\n",
		},
		{
			name: "Start pipeline with invalid workspace name",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-w=name=password-vault,secret=secret-name",
				"-w=claimName=pvc3",
				"-n", "ns",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "Name not found for workspace",
		},
		{
			name: "Wrong parameter name",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-p=rev-parm=revision2",
				"-n", "ns",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "param 'rev-parm' not present in spec",
		},
		{
			name: "Invalid parameter format",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=rev-paramrevision2",
				"--task-serviceaccount=task3=task3svc3",
				"--task-serviceaccount=task5=task3svc5",
				"-n", "ns",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "invalid input format for param parameter: rev-paramrevision2",
		},
		{
			name: "Invalid label parameter format",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value",
				"-p=rev-param=revision2",
				"-l=keyvalue",
				"--task-serviceaccount=task3=task3svc3",
				"--task-serviceaccount=task5=task3svc5",
				"-n", "ns",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "invalid input format for label parameter: keyvalue",
		},
		{
			name: "Invalid service account parameter format",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value",
				"-p=rev-param=revision2",
				"--task-serviceaccount=task3svc3",
				"-n", "ns",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "invalid service account parameter: task3svc3\nPlease pass Task service accounts as --task-serviceaccount TaskName=ServiceAccount",
		},
		{
			name: "List error with last flag",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=rev-param=revision2",
				"--task-serviceaccount=task3=task3svc3",
				"--last",
				"-n", "ns",
			},
			namespace: "",
			input:     c3,
			wantError: true,
			want:      "test generated error",
		},
		{
			name: "Create error",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-n", "ns",
			},
			namespace: "",
			input:     c4,
			wantError: true,
			want:      "mock error",
		},
		{
			name: "No pipelineruns with last flag",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=rev-param=revision2",
				"--task-serviceaccount=task3=task3svc3",
				"--task-serviceaccount=task5=task3svc5",
				"--last",
				"-n", "ns",
			},
			namespace: "",
			input:     c5,
			wantError: true,
			want:      "no pipelineruns related to pipeline test-pipeline found in namespace ns",
		},
		{
			name: "Dry Run with invalid output",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--output", "invalid",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "output format specified is invalid but must be yaml or json",
		},
		{
			name: "Dry Run with only --dry-run specified",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
			},
			namespace:  "",
			input:      c2,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with --use-param-defaults and specified params",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--use-param-defaults",
			},
			namespace:  "",
			input:      c2,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with --use-param-defaults and no specified params",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"--use-param-defaults",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
			},
			namespace:  "",
			input:      c2,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with --last and --use-pipelinerun",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--last",
				"--use-pipelinerun", "dummy-pipelinerun",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "option --last and option --use-pipelinerun can't be specify together",
		},
		{
			name: "Dry Run with --use-param-defaults and --use-pipelinerun",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"--use-param-defaults",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--use-pipelinerun", "dummy-pipelinerun",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "cannot use --last or --use-pipelinerun options with --use-param-defaults option",
		},
		{
			name: "Dry Run with --use-param-defaults and --last",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"--use-param-defaults",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--last",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "cannot use --last or --use-pipelinerun options with --use-param-defaults option",
		},
		{
			name: "Dry Run with output=json",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--output", "json",
			},
			namespace:  "",
			input:      c2,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with output=name",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--output=name",
			},
			namespace:  "",
			input:      c2,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run using --filename v1beta1",
			command: []string{
				"start", "-f", "./testdata/pipeline.yaml",
				"-n", "ns",
				"--dry-run",
			},
			namespace:  "",
			input:      c6,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with output=json -f v1beta1",
			command: []string{
				"start", "-f", "./testdata/pipeline.yaml",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-n", "ns",
				"--dry-run",
				"--output", "json",
			},
			namespace:  "",
			input:      c6,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with output=json -f v1beta1 parameter without type",
			command: []string{
				"start", "-f", "./testdata/pipeline-parameter-with-invalid-type.yaml",
				"-n", "ns",
				"--dry-run",
				"--output", "json",
			},
			namespace: "",
			input:     c6,
			wantError: true,
			want:      "params does not have a valid type - 'test-param'",
		},
		{
			name: "Start pipeline using --filename v1beta1",
			command: []string{
				"start", "-f", "./testdata/pipeline.yaml",
				"-n", "ns",
			},
			namespace: "",
			input:     c6,
			wantError: false,
			want:      "PipelineRun started: random\n\nIn order to track the PipelineRun progress run:\ntkn pipelinerun logs random -f -n ns\n",
		},
		{
			name: "Start pipeline using invalid --filename v1beta1",
			command: []string{
				"start", "-f", "./testdata/pipeline-invalid.yaml",
				"-n", "ns",
			},
			namespace: "",
			input:     c6,
			wantError: true,
			want:      `error unmarshaling JSON: while decoding JSON: json: unknown field "conditons"`,
		},
		{
			name: "Error from using --last with --filename",
			command: []string{
				"start", "-f", "./testdata/pipeline-not.yaml",
				"-n", "ns",
				"--last",
			},
			namespace: "",
			input:     c6,
			wantError: true,
			want:      "cannot use --last option with --filename option",
		},
		{
			name: "Dry Run with --pipeline-timeout specified",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--pipeline-timeout", "1s",
			},
			namespace:  "",
			input:      c2,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with --pipeline-timeout, --tasks-timeout specified",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--pipeline-timeout", "2s",
				"--tasks-timeout", "1s",
			},
			namespace:  "",
			input:      c2,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with --pipeline-timeout, --tasks-timeout, --finally-timeout specified",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--pipeline-timeout", "3s",
				"--tasks-timeout", "1s",
				"--finally-timeout", "1s",
			},
			namespace:  "",
			input:      c2,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with invalid --timeout specified (deprecated)",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--timeout", "5d",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			hasPrefix: true,
			want:      `time: unknown unit`,
		},
		{
			name: "Dry Run with invalid --pipeline-timeout specified",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--pipeline-timeout", "5d",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			hasPrefix: true,
			want:      `time: unknown unit`,
		},
		{
			name: "Dry Run with invalid --tasks-timeout specified",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--tasks-timeout", "5d",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			hasPrefix: true,
			want:      `time: unknown unit`,
		},
		{
			name: "Dry Run with --finally-timeout specified",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--finally-timeout", "5d",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			hasPrefix: true,
			want:      `time: unknown unit`,
		},

		{
			name: "Dry Run with PodTemplate",
			command: []string{
				"start", "test-pipeline",
				"-s=svc1",
				"-p=pipeline-param=value1",
				"-p=rev-param=value2",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--pod-template", "./testdata/podtemplate.yaml",
			},
			namespace:  "",
			input:      c2,
			wantError:  false,
			goldenFile: true,
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			if tp.namespace != "" {
				tp.input.SetNamespace(tp.namespace)
			}
			c := Command(tp.input)

			got, err := test.ExecuteCommand(c, tp.command...)
			if tp.wantError {
				if err == nil {
					t.Errorf("error expected here")
				}

				if tp.hasPrefix {
					test.AssertOutputPrefix(t, tp.want, err.Error())
				} else {
					test.AssertOutput(t, tp.want, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected Error")
				}
				if tp.goldenFile {
					golden.Assert(t, got, strings.ReplaceAll(fmt.Sprintf("%s.golden", t.Name()), "/", "-"))
				} else {
					test.AssertOutput(t, tp.want, got)
				}
			}
		})
	}
}

func TestPipelineStart_Interactive_v1beta1(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedV1beta1TestData(t, test.Data{
		Pipelines: []*v1beta1.Pipeline{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pipeline",
					Namespace: "ns",
				},
				Spec: v1beta1.PipelineSpec{
					Tasks: []v1beta1.PipelineTask{
						{
							Name: "unit-test-1",
							TaskRef: &v1beta1.TaskRef{
								Name: "unit-test-task",
							},
						},
					},
					Params: []v1beta1.ParamSpec{
						{
							Name: "pipeline-param",
							Type: v1beta1.ParamTypeString,
							Default: &v1beta1.ParamValue{
								Type:      v1beta1.ParamTypeString,
								StringVal: "somethingdifferent",
							},
						},
						{
							Name: "rev-param",
							Type: v1beta1.ParamTypeString,
							Default: &v1beta1.ParamValue{
								Type:      v1beta1.ParamTypeString,
								StringVal: "revision",
							},
						},
						{
							Name: "array-param",
							Type: v1beta1.ParamTypeArray,
							Default: &v1beta1.ParamValue{
								Type:     v1beta1.ParamTypeString,
								ArrayVal: []string{"revision1", "revision2"},
							},
						},
					},
				},
			},
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	cs12, _ := test.SeedV1beta1TestData(t, test.Data{
		Pipelines: []*v1beta1.Pipeline{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloudpipeline",
					Namespace: "ns",
				},
				Spec: v1beta1.PipelineSpec{
					Tasks: []v1beta1.PipelineTask{
						{
							Name: "unit-test-1",
							TaskRef: &v1beta1.TaskRef{
								Name: "unit-test-task",
							},
						},
					},
					Workspaces: []v1beta1.PipelineWorkspaceDeclaration{
						{
							Name:        "pvc",
							Description: "config",
						},
					},
				},
			},
		},
	})

	testParams := []struct {
		name               string
		namespace          string
		input              test.Clients
		last               bool
		serviceAccountName string
		serviceAccounts    []string
		prompt             prompt.Prompt
	}{
		{
			name:               "Start pipeline with pipeline-param, rev-param and array-param from interactive menu",
			namespace:          "ns",
			input:              cs,
			last:               false,
			serviceAccountName: "svc1",
			serviceAccounts:    []string{"task1=svc1"},
			prompt: prompt.Prompt{
				CmdArgs: []string{"test-pipeline"},
				Procedure: func(c *expect.Console) error {
					if _, err := c.ExpectString("Value for param `pipeline-param` of type `string`? (Default is `somethingdifferent`)"); err != nil {
						return err
					}

					if _, err := c.SendLine("test"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Value for param `rev-param` of type `string`? (Default is `revision`)"); err != nil {
						return err
					}

					if _, err := c.SendLine("test1"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Value for param `array-param` of type `array`? (Default is `revision1,revision2`)"); err != nil {
						return err
					}

					if _, err := c.SendLine("test2, test3"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Pipelinerun started:"); err != nil {
						return err
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					c.Close()
					return nil
				},
			},
		},
		{
			name:               "Pipeline with workspace",
			namespace:          "ns",
			input:              cs12,
			last:               false,
			serviceAccountName: "svc1",
			serviceAccounts:    []string{"task1=svc1"},
			prompt: prompt.Prompt{
				CmdArgs: []string{"cloudpipeline"},
				Procedure: func(c *expect.Console) error {
					if _, err := c.ExpectString("Name for the workspace :"); err != nil {
						return err
					}

					if _, err := c.SendLine("pvc1"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Value of the Sub Path :"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Type of the Workspace :"); err != nil {
						return err
					}

					if _, err := c.SendLine("pvc"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Value of Claim :"); err != nil {
						return err
					}

					if _, err := c.SendLine("pvc1"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Name for the workspace :"); err != nil {
						return err
					}

					if _, err := c.SendLine("config"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Value of the Sub Path :"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Type of the Workspace :"); err != nil {
						return err
					}

					if _, err := c.SendLine("config"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Name of the configmap :"); err != nil {
						return err
					}

					if _, err := c.SendLine("cmpap"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Item Value :"); err != nil {
						return err
					}

					if _, err := c.SendLine("key=value"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Item Value :"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Name for the workspace :"); err != nil {
						return err
					}

					if _, err := c.SendLine("secret"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Value of the Sub Path :"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Type of the Workspace :"); err != nil {
						return err
					}

					if _, err := c.SendLine("secret"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Name of the secret :"); err != nil {
						return err
					}

					if _, err := c.SendLine("secretname"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Item Value :"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Name for the workspace :"); err != nil {
						return err
					}

					if _, err := c.SendLine("emptyDir"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Value of the Sub Path :"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Type of the Workspace :"); err != nil {
						return err
					}

					if _, err := c.SendLine("emptyDir"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Type of EmptyDir :"); err != nil {
						return err
					}

					if _, err := c.SendLine(""); err != nil {
						return err
					}

					c.Close()
					return nil
				},
			},
		},
	}

	for _, tp := range testParams {
		t.Run(tp.name, func(t *testing.T) {
			p := test.Params{
				Kube:   tp.input.Kube,
				Tekton: tp.input.Pipeline,
			}
			p.SetNamespace(tp.namespace)

			opts := startOptions{
				cliparams:          &p,
				Last:               tp.last,
				ServiceAccountName: tp.serviceAccountName,
				ServiceAccounts:    tp.serviceAccounts,
			}

			tp.prompt.RunTest(t, tp.prompt.Procedure, func(stdio terminal.Stdio) error {
				opts.askOpts = prompt.WithStdio(stdio)
				opts.stream = &cli.Stream{Out: stdio.Out, Err: stdio.Err}
				pipelineObj := &v1beta1.Pipeline{}
				pipelineObj.ObjectMeta.Name = tp.prompt.CmdArgs[0]
				return opts.run(pipelineObj)
			})
		})
	}
}

func Test_start_pipeline_v1beta1(t *testing.T) {
	pipelineName := "test-pipeline"

	pipeline := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "unit-test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "unit-test-task",
						},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeArray,
						Default: &v1beta1.ParamValue{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Pipelines: pipeline, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(pipeline[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	c := Command(p)

	got, _ := test.ExecuteCommand(c, "start", pipelineName,
		"-p=pipeline-param=value1",
		"-p=rev-param=cat,foo,bar",
		"-l=jemange=desfrites",
		"-s=svc1",
		"-n", "ns")

	expected := "PipelineRun started: \n\nIn order to track the PipelineRun progress run:\ntkn pipelinerun logs  -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	var pr *v1.PipelineRunList
	err = actions.ListV1(pipelineRunGroupResource, cl, metav1.ListOptions{}, "ns", &pr)
	if err != nil {
		t.Errorf("Error listing pipelineruns %s", err.Error())
	}

	if pr.Items[0].ObjectMeta.GenerateName != (pipelineName + "-run-") {
		t.Errorf("Error pipelinerun generated is different %+v", pr)
	}

	test.AssertOutput(t, 2, len(pr.Items[0].Spec.Params))

	for _, v := range pr.Items[0].Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1.ParamValue{Type: v1.ParamTypeArray, ArrayVal: []string{"cat", "foo", "bar"}}, v.Value)
		}
	}

	if d := cmp.Equal(pr.Items[0].ObjectMeta.Labels, map[string]string{"jemange": "desfrites"}); !d {
		t.Errorf("Error labels generated is different Labels Got: %+v", pr.Items[0].ObjectMeta.Labels)
	}
}

func Test_start_pipeline_last_v1beta1(t *testing.T) {
	pipelineName := "test-pipeline"
	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "unit-test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "unit-test-task",
						},
						Workspaces: []v1beta1.WorkspacePipelineTaskBinding{
							{
								Name:      "task-test-workspace",
								Workspace: "test-workspace",
							},
						},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision",
						},
					},
				},
				Workspaces: []v1beta1.PipelineWorkspaceDeclaration{
					{
						Name: "test-workspace",
					},
				},
			},
		},
	}

	timeoutDuration, _ := time.ParseDuration("10s")
	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pipeline-run-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
				Workspaces: []v1beta1.WorkspaceBinding{
					{
						Name: "test-new",
					},
				},
				Timeouts: &v1beta1.TimeoutFields{Pipeline: &metav1.Duration{Duration: timeoutDuration}},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedV1beta1TestData(t, test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newV1beta1PipelineClient(objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-n", "ns",
	)

	expected := "PipelineRun started: random\n\nIn order to track the PipelineRun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	var pr *v1beta1.PipelineRun
	if err = actions.GetV1(pipelineRunGroupResource, cl, "random", "ns", metav1.GetOptions{}, &pr); err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	test.AssertOutput(t, 2, len(pr.Spec.Params))
	for _, v := range pr.Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "revision1"}, v.Value)
		}
	}

	test.AssertOutput(t, "test-sa", pr.Spec.ServiceAccountName)
	test.AssertOutput(t, "test-new", pr.Spec.Workspaces[0].Name)
	test.AssertOutput(t, timeoutDuration, pr.Spec.Timeouts.Pipeline.Duration)
}

func Test_start_pipeline_last_override_timeout_deprecated_v1beta1(t *testing.T) {
	pipelineName := "test-pipeline"
	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "unit-test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "unit-test-task",
						},
						Workspaces: []v1beta1.WorkspacePipelineTaskBinding{
							{
								Name:      "task-test-workspace",
								Workspace: "test-workspace",
							},
						},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision",
						},
					},
				},
				Workspaces: []v1beta1.PipelineWorkspaceDeclaration{
					{
						Name: "test-workspace",
					},
				},
			},
		},
	}

	// Add timeout to last PipelineRun for Pipeline
	timeoutDuration, _ := time.ParseDuration("10s")
	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pipeline-run-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
				Workspaces: []v1beta1.WorkspaceBinding{
					{
						Name: "test-new",
					},
				},
				Timeouts: &v1beta1.TimeoutFields{Pipeline: &metav1.Duration{Duration: timeoutDuration}},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedV1beta1TestData(t, test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newV1beta1PipelineClient(objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pipeline := Command(p)
	// Specify new timeout value to override previous value
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"--timeout", "1s",
		"-n", "ns",
	)

	expected := "Flag --timeout has been deprecated, please use --pipeline-timeout flag instead\nPipelineRun started: random\n\nIn order to track the PipelineRun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	var pr *v1beta1.PipelineRun
	if err = actions.GetV1(pipelineRunGroupResource, cl, "random", "ns", metav1.GetOptions{}, &pr); err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	// Assert newly started PipelineRun has new timeout value
	timeoutDuration, _ = time.ParseDuration("1s")
	test.AssertOutput(t, timeoutDuration, pr.Spec.Timeouts.Pipeline.Duration)
}

func Test_start_pipeline_last_without_res_param_v1beta1(t *testing.T) {
	pipelineName := "test-pipeline"

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "unit-test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "unit-test-task",
						},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision",
						},
					},
				},
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pipeline-run-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedV1beta1TestData(t, test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newV1beta1PipelineClient(objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-n", "ns")

	expected := "PipelineRun started: random\n\nIn order to track the PipelineRun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	var pr *v1beta1.PipelineRun
	if err = actions.GetV1(pipelineRunGroupResource, cl, "random", "ns", metav1.GetOptions{}, &pr); err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	test.AssertOutput(t, 2, len(pr.Spec.Params))

	for _, v := range pr.Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "revision1"}, v.Value)
		}
	}
	test.AssertOutput(t, "test-sa", pr.Spec.ServiceAccountName)
}

func Test_start_pipeline_last_merge_v1beta1(t *testing.T) {
	pipelineName := "test-pipeline"

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "unit-test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "unit-test-task",
						},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision",
						},
					},
				},
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pipeline-run-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
				ServiceAccountName: "test-sa",
				TaskRunSpecs: []v1beta1.PipelineTaskRunSpec{
					{
						PipelineTaskName:       "task1",
						TaskServiceAccountName: "task1svc",
					},
					{
						PipelineTaskName:       "task3",
						TaskServiceAccountName: "task3svc",
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedV1beta1TestData(t, test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newV1beta1PipelineClient(objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-s=svc1",
		"-p=rev-param=revision2",
		"--task-serviceaccount=task3=task3svc3",
		"--task-serviceaccount=task5=task3svc5",
		"-n=ns")

	expected := "PipelineRun started: random\n\nIn order to track the PipelineRun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	var pr *v1beta1.PipelineRun
	if err = actions.GetV1(pipelineRunGroupResource, cl, "random", "ns", metav1.GetOptions{}, &pr); err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	test.AssertOutput(t, 2, len(pr.Spec.Params))

	for _, v := range pr.Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1beta1.ParamValue{Type: v1beta1.ParamTypeString, StringVal: "revision2"}, v.Value)
		}
	}

	for _, v := range pr.Spec.TaskRunSpecs {
		if v.PipelineTaskName == "task3" {
			test.AssertOutput(t, "task3svc3", v.TaskServiceAccountName)
		}
	}

	test.AssertOutput(t, "svc1", pr.Spec.ServiceAccountName)
}

func Test_start_pipeline_use_pipelinerun_v1beta1(t *testing.T) {
	pipelineName := "test-pipeline"
	theonename := "test-pipeline-run-be-the-one"
	timeoutDuration, _ := time.ParseDuration("10s")

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "unit-test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "unit-test-task",
						},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision",
						},
					},
				},
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "dont-bother-me-trying",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      theonename,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
				Params: []v1beta1.Param{
					{
						Name: "brush",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "teeth",
						},
					},
				},
				Timeouts: &v1beta1.TimeoutFields{Pipeline: &metav1.Duration{Duration: timeoutDuration}},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedV1beta1TestData(t, test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0], prs[1]}
	_, tdc := newV1beta1PipelineClient(objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[1], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pipeline := Command(p)
	// There is no point to checkout otuput since we would be checking if our testdata works!
	_, _ = test.ExecuteCommand(pipeline, "start", pipelineName,
		"--use-pipelinerun="+theonename, "-n", "ns")

	cl, _ := p.Clients()
	var pr *v1beta1.PipelineRun
	if err = actions.GetV1(pipelineRunGroupResource, cl, "random", "ns", metav1.GetOptions{}, &pr); err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}
	test.AssertOutput(t, pr.Spec.Params[0].Name, "brush")
	test.AssertOutput(t, pr.Spec.Params[0].Value, v1beta1.ParamValue{Type: "string", StringVal: "teeth"})
	test.AssertOutput(t, timeoutDuration, pr.Spec.Timeouts.Pipeline.Duration)
}

func Test_start_pipeline_use_pipelinerun_cancelled_status_v1beta1(t *testing.T) {
	pipelineName := "test-pipeline"
	theonename := "test-pipeline-run-be-the-one"
	timeoutDuration, _ := time.ParseDuration("10s")

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "unit-test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "unit-test-task",
						},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision",
						},
					},
				},
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      theonename,
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
				Params: []v1beta1.Param{
					{
						Name: "brush",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "teeth",
						},
					},
				},
				Timeouts: &v1beta1.TimeoutFields{Pipeline: &metav1.Duration{Duration: timeoutDuration}},
				// nolint: staticcheck
				Status: v1beta1.PipelineRunSpecStatus(v1beta1.PipelineRunSpecStatusCancelled),
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.PipelineRunReasonCancelled.String(),
						},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedV1beta1TestData(t, test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newV1beta1PipelineClient(objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pipeline := Command(p)
	_, _ = test.ExecuteCommand(pipeline, "start", pipelineName,
		"--use-pipelinerun="+theonename, "-n", "ns")

	cl, _ := p.Clients()
	var pr *v1beta1.PipelineRun
	if err = actions.GetV1(pipelineRunGroupResource, cl, "random", "ns", metav1.GetOptions{}, &pr); err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}
	test.AssertOutput(t, pr.Spec.Params[0].Name, "brush")
	test.AssertOutput(t, pr.Spec.Params[0].Value, v1beta1.ParamValue{Type: "string", StringVal: "teeth"})
	test.AssertOutput(t, timeoutDuration, pr.Spec.Timeouts.Pipeline.Duration)
	// Assert that new PipelineRun does not contain cancelled status of previous run
	test.AssertOutput(t, v1beta1.PipelineRunSpecStatus(""), pr.Spec.Status)
}

func Test_start_pipeline_allkindparam_v1beta1(t *testing.T) {
	pipelineName := "test-pipeline"
	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "unit-test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "unit-test-task",
						},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeArray,
						Default: &v1beta1.ParamValue{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
					{
						Name: "rev-param-new",
						Type: v1beta1.ParamTypeArray,
						Default: &v1beta1.ParamValue{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Pipelines: ps, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
	pipeline := Command(p)

	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"-p=pipeline-param=value1",
		"-p=rev-param=cat,foo,bar",
		"-p=rev-param-new=help",
		"-l=jemange=desfrites",
		"-s=svc1",
		"-n", "ns")

	expected := "PipelineRun started: \n\nIn order to track the PipelineRun progress run:\ntkn pipelinerun logs  -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	var pr *v1.PipelineRunList
	err = actions.ListV1(pipelineRunGroupResource, cl, metav1.ListOptions{}, "ns", &pr)
	if err != nil {
		t.Errorf("Error listing pipelineruns %s", err.Error())
	}

	if pr.Items[0].ObjectMeta.GenerateName != (pipelineName + "-run-") {
		t.Errorf("Error pipelinerun generated is different %+v", pr)
	}

	test.AssertOutput(t, 3, len(pr.Items[0].Spec.Params))
	for _, v := range pr.Items[0].Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1.ParamValue{Type: v1.ParamTypeArray, ArrayVal: []string{"cat", "foo", "bar"}}, v.Value)
		}

		if v.Name == "rev-param-new" {
			test.AssertOutput(t, v1.ParamValue{Type: v1.ParamTypeArray, ArrayVal: []string{"help"}}, v.Value)
		}
	}

	if d := cmp.Equal(pr.Items[0].ObjectMeta.Labels, map[string]string{"jemange": "desfrites"}); !d {
		t.Errorf("Error labels generated is different Labels Got: %+v", pr.Items[0].ObjectMeta.Labels)
	}
}

func Test_start_pipeline_last_generate_name_v1beta1(t *testing.T) {
	pipelineName := "test-pipeline"

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "unit-test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "unit-test-task",
						},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision",
						},
					},
				},
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pipeline-run-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	// Setting GenerateName for test
	prs[0].ObjectMeta.GenerateName = "test-generatename-pipeline-run-"

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedV1beta1TestData(t, test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newV1beta1PipelineClient(objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-p=rev-param=revision2",
		"-s=svc1",
		"--task-serviceaccount=task3=task3svc3",
		"--task-serviceaccount=task5=task3svc5",
		"-n", "ns")

	expected := "PipelineRun started: random\n\nIn order to track the PipelineRun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	var pr *v1beta1.PipelineRun
	if err = actions.GetV1(pipelineRunGroupResource, cl, "random", "ns", metav1.GetOptions{}, &pr); err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	test.AssertOutput(t, "test-generatename-pipeline-run-", pr.ObjectMeta.GenerateName)
}

func Test_start_pipeline_last_with_prefix_name_v1beta1(t *testing.T) {
	pipelineName := "test-pipeline"

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "unit-test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "unit-test-task",
						},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision",
						},
					},
				},
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pipeline-run-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedV1beta1TestData(t, test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newV1beta1PipelineClient(objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-p=rev-param=revision2",
		"-s=svc1",
		"--task-serviceaccount=task3=task3svc3",
		"--task-serviceaccount=task5=task3svc5",
		"-n", "ns",
		"--prefix-name", "myprname",
	)

	expected := "PipelineRun started: random\n\nIn order to track the PipelineRun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	var pr *v1beta1.PipelineRun
	if err = actions.GetV1(pipelineRunGroupResource, cl, "random", "ns", metav1.GetOptions{}, &pr); err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	test.AssertOutput(t, "myprname-", pr.ObjectMeta.GenerateName)
}

func Test_start_pipeline_with_prefix_name_v1beta1(t *testing.T) {
	pipelineName := "test-pipeline"

	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "unit-test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "unit-test-task",
						},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision",
						},
					},
				},
			},
		},
	}

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pipeline-run-123",
				Namespace: "ns",
				Labels:    map[string]string{"tekton.dev/pipeline": pipelineName},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: pipelineName,
				},
				ServiceAccountName: "test-sa",
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedV1beta1TestData(t, test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newV1beta1PipelineClient(objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"-p=pipeline-param-1=value1",
		"-p=rev-param=revision2",
		"-s=svc1",
		"--task-serviceaccount=task3=task3svc3",
		"--task-serviceaccount=task5=task3svc5",
		"-n", "ns",
		"--prefix-name", "myprname",
	)

	expected := "PipelineRun started: random\n\nIn order to track the PipelineRun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	var pr *v1beta1.PipelineRun
	if err = actions.GetV1(pipelineRunGroupResource, cl, "random", "ns", metav1.GetOptions{}, &pr); err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	test.AssertOutput(t, "myprname-", pr.ObjectMeta.GenerateName)
}

func Test_lastPipelineRun_v1beta1(t *testing.T) {
	clock := test.FakeClock()

	pr1Started := clock.Now().Add(10 * time.Second)
	pr2Started := clock.Now().Add(-2 * time.Hour)
	pr3Started := clock.Now().Add(-450 * time.Hour)

	prs := []*v1beta1.PipelineRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pr-2",
				Namespace:         "namespace",
				Labels:            map[string]string{"tekton.dev/pipeline": "test"},
				CreationTimestamp: metav1.Time{Time: pr2Started},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "test",
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonRunning.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pr-3",
				Namespace:         "namespace",
				Labels:            map[string]string{"tekton.dev/pipeline": "test"},
				CreationTimestamp: metav1.Time{Time: pr3Started},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "test",
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: v1beta1.PipelineRunReasonFailed.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "pr-1",
				Namespace:         "namespace",
				Labels:            map[string]string{"tekton.dev/pipeline": "test"},
				CreationTimestamp: metav1.Time{Time: pr1Started},
			},
			Spec: v1beta1.PipelineRunSpec{
				PipelineRef: &v1beta1.PipelineRef{
					Name: "test",
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.PipelineRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace",
			},
		},
	}

	type args struct {
		p        cli.Params
		pipeline string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "lastPipelineRun Test No Err",
			args: args{
				pipeline: "test",
				p: func() *test.Params {
					clock.Advance(time.Duration(60) * time.Minute)

					cs, _ := test.SeedV1beta1TestData(t, test.Data{PipelineRuns: prs, Namespaces: ns})
					cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
					tdc := testDynamic.Options{}
					dc, err := tdc.Client(
						cb.UnstructuredV1beta1PR(prs[0], versionv1beta1),
						cb.UnstructuredV1beta1PR(prs[1], versionv1beta1),
						cb.UnstructuredV1beta1PR(prs[2], versionv1beta1),
					)
					if err != nil {
						t.Errorf("unable to create dynamic client: %v", err)
					}
					p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Clock: clock}
					p.SetNamespace("namespace")
					return p
				}(),
			},
			want:    "pr-1",
			wantErr: false,
		},
		{
			name: "lastPipelineRun Test Err",
			args: args{
				pipeline: "test",
				p: func() *test.Params {
					cs, _ := test.SeedV1beta1TestData(t, test.Data{Namespaces: ns})
					cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
					tdc := testDynamic.Options{}
					dc, err := tdc.Client()
					if err != nil {
						t.Errorf("unable to create dynamic client: %v", err)
					}
					p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}
					p.SetNamespace("namespace")
					return p
				}(),
			},

			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs, _ := tt.args.p.Clients()
			got, err := pipeline.LastRun(cs, tt.args.pipeline, tt.args.p.Namespace())
			if (err != nil) != tt.wantErr {
				t.Errorf("lastPipelineRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			} else if err == nil {
				test.AssertOutput(t, tt.want, got.Name)
			}
		})
	}
}

func Test_start_pipeline_with_skip_optional_workspace_flag_v1beta1(t *testing.T) {
	pipelineName := "test-pipeline"
	ps := []*v1beta1.Pipeline{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pipelineName,
				Namespace: "ns",
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{
					{
						Name: "unit-test-1",
						TaskRef: &v1beta1.TaskRef{
							Name: "unit-test-task",
						},
						Workspaces: []v1beta1.WorkspacePipelineTaskBinding{
							{
								Name:      "task-test-workspace",
								Workspace: "test-workspace",
							},
						},
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ParamValue{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision",
						},
					},
				},
				Workspaces: []v1beta1.PipelineWorkspaceDeclaration{
					{
						Name:     "test-workspace",
						Optional: true,
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedV1beta1TestData(t, test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0]}
	_, tdc := newV1beta1PipelineClient(objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--skip-optional-workspace",
		"-n", "ns",
		"-p=pipeline-param-1=value1",
		"-p=rev-param=value2",
	)

	expected := "PipelineRun started: random\n\nIn order to track the PipelineRun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)
}
