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
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	"github.com/google/go-cmp/cmp"
	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/pipeline"
	"github.com/tektoncd/cli/pkg/pipelinerun"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/cli/test/prompt"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	fakepipelineclientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"github.com/tektoncd/pipeline/pkg/reconciler/pipelinerun/resources"
	pipelinev1beta1test "github.com/tektoncd/pipeline/test"
	tb "github.com/tektoncd/pipeline/test/builder"
	pipelinetest "github.com/tektoncd/pipeline/test/v1alpha1"
	"gotest.tools/v3/golden"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	util_runtime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	k8stest "k8s.io/client-go/testing"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
)

func newPipelineClient(version string, objs ...runtime.Object) (*fakepipelineclientset.Clientset, testDynamic.Options) {
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	var localSchemeBuilder runtime.SchemeBuilder
	if version == "v1alpha1" {
		localSchemeBuilder = runtime.SchemeBuilder{v1alpha1.AddToScheme}
	} else {
		localSchemeBuilder = runtime.SchemeBuilder{v1beta1.AddToScheme}
	}

	v1.AddToGroupVersion(scheme, schema.GroupVersion{Version: "v1"})
	util_runtime.Must(localSchemeBuilder.AddToScheme(scheme))
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

					if res.Version == "v1alpha1" {
						v1alpha1PR := obj.(*v1alpha1.PipelineRun)
						unstructuredPR := cb.UnstructuredPR(v1alpha1PR, versionA1)
						return true, unstructuredPR, nil
					}
					v1beta1PR := obj.(*v1beta1.PipelineRun)
					unstructuredPR := cb.UnstructuredV1beta1PR(v1beta1PR, versionB1)
					return true, unstructuredPR, nil
				},
			},
		},
	}
	return nil, dc
}

func TestPipelineStart_ExecuteCommand(t *testing.T) {
	clock := clockwork.NewFakeClock()
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{})
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c1 := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Clock: clock, Resource: cs.Resource}

	pipeline := []*v1alpha1.Pipeline{
		tb.Pipeline("test-pipeline", "ns",
			tb.PipelineSpec(
				tb.PipelineDeclaredResource("git-repo", "git"),
				tb.PipelineDeclaredResource("build-image", "image"),
				tb.PipelineParamSpec("pipeline-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("somethingdifferent")),
				tb.PipelineParamSpec("rev-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("revision")),
				tb.PipelineTask("unit-test-1", "unit-test-task",
					tb.PipelineTaskInputResource("workspace", "git-repo"),
					tb.PipelineTaskOutputResource("image-to-use", "best-image"),
					tb.PipelineTaskOutputResource("workspace", "git-repo"),
				),
			),
		),
	}

	pipeline2 := []*v1alpha1.Pipeline{
		tb.Pipeline("test-pipeline", "ns",
			tb.PipelineSpec(
				tb.PipelineTask("unit-test-1", "unit-test-task",
					tb.PipelineTaskInputResource("workspace", "git-repo"),
					tb.PipelineTaskOutputResource("image-to-use", "best-image"),
					tb.PipelineTaskOutputResource("workspace", "git-repo"),
				),
			),
		),
	}
	cs2, _ := test.SeedTestData(t, pipelinetest.Data{Pipelines: pipeline, Namespaces: namespaces})
	cs2.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredP(pipeline[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c2 := &test.Params{Tekton: cs2.Pipeline, Kube: cs2.Kube, Dynamic: dc2, Clock: clock, Resource: cs2.Resource}

	// With list error mocking
	cs3, _ := test.SeedTestData(t, pipelinetest.Data{Pipelines: pipeline, Namespaces: namespaces})
	cs3.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
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
		cb.UnstructuredP(pipeline[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c3 := &test.Params{Tekton: cs3.Pipeline, Kube: cs3.Kube, Dynamic: dc3, Clock: clock, Resource: cs3.Resource}

	// With create error mocking
	cs4, _ := test.SeedTestData(t, pipelinetest.Data{Pipelines: pipeline2, Namespaces: namespaces})
	cs4.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
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
		cb.UnstructuredP(pipeline2[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c4 := &test.Params{Tekton: cs4.Pipeline, Kube: cs4.Kube, Dynamic: dc4, Clock: clock, Resource: cs4.Resource}

	// Without related pipelinerun
	objs := []runtime.Object{
		pipeline[0],
	}
	seedData, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines:  pipeline,
		Namespaces: namespaces,
	})
	cs5 := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs5.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
	_, tdc5 := newPipelineClient("v1alpha1", objs...)
	dc5, err := tdc5.Client(
		cb.UnstructuredP(pipeline[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c5 := &test.Params{Tekton: cs5.Pipeline, Kube: cs5.Kube, Dynamic: dc5, Clock: clock, Resource: cs5.Resource}

	// pipelineresources data for tests with --filename
	objs2 := []runtime.Object{}
	pres := []*v1alpha1.PipelineResource{
		tb.PipelineResource("scaffold-git", "ns",
			tb.PipelineResourceSpec("git",
				tb.PipelineResourceSpecParam("url", "git@github.com:tektoncd/cli.git"),
			),
		),
		tb.PipelineResource("imageres", "ns",
			tb.PipelineResourceSpec("image",
				tb.PipelineResourceSpecParam("url", "gcr.io/christiewilson-catfactory/leeroy-web"),
			),
		),
	}
	seedData2, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces:        namespaces,
		PipelineResources: pres,
	})
	cs6 := pipelinetest.Clients{
		Pipeline: seedData2.Pipeline,
		Kube:     seedData2.Kube,
		Resource: seedData2.Resource,
	}
	cs6.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
	_, tdc6 := newPipelineClient("v1alpha1", objs2...)
	dc6, err := tdc6.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c6 := &test.Params{Tekton: cs6.Pipeline, Kube: cs6.Kube, Dynamic: dc6, Clock: clock, Resource: cs6.Resource}

	testParams := []struct {
		name       string
		command    []string
		namespace  string
		input      *test.Params
		wantError  bool
		want       string
		goldenFile bool
	}{
		{
			name:      "Invalid namespace",
			command:   []string{"start", "pipeline", "-n", "invalid"},
			namespace: "",
			input:     c1,
			wantError: true,
			want:      "namespaces \"invalid\" not found",
		},
		{
			name:      "Missing pipeline name",
			command:   []string{"start", "-n", "ns"},
			namespace: "",
			input:     c1,
			wantError: true,
			want:      "missing pipeline name",
		},
		{
			name:      "Found no pipelines",
			command:   []string{"start", "test-pipeline-2", "-n", "ns"},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "pipeline name test-pipeline-2 does not exist in namespace ns",
		},
		{
			name: "Start pipeline with showlog flag false",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
				"-p=pipeline-param=value1",
				"-l=jemange=desfrites",
				"-w=name=password-vault,secret=secret-name",
				"-n", "ns",
			},
			namespace: "",
			input:     c2,
			wantError: false,
			want:      "Pipelinerun started: \n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs  -f -n ns\n",
		},
		{
			name: "Start pipeline with showlog flag false",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
				"-p=pipeline-param=value1",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=git-repo=scaffold-git",
				"-p=rev-parm=revision2",
				"-n", "ns",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "param 'rev-parm' not present in spec",
		},
		{
			name: "Invalid resource parameter format",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=git-reposcaffold-git",
				"-p=rev-param=revision2",
				"--task-serviceaccount=task3=task3svc3",
				"--task-serviceaccount=task5=task3svc5",
				"-n", "ns",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "invalid input format for resource parameter: git-reposcaffold-git",
		},
		{
			name: "Invalid parameter format",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=git-repo=scaffold-git",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=git-repo=scaffold-git",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=git-repo=scaffold-git",
				"-p=rev-param=revision2",
				"--task-serviceaccount=task3svc3",
				"-n", "ns",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "invalid service account parameter: task3svc3\nPlease pass task service accounts as --task-serviceaccount TaskName=ServiceAccount",
		},
		{
			name: "List error with last flag",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=git-repo=scaffold-git",
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
			command: []string{"start", "test-pipeline",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=git-repo=scaffold-git",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
				"-p=pipeline-param=value1",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
				"-p=pipeline-param=value1",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
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
			name: "Dry Run with --use-param-defaults, --last and --use-pipelinerun",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
				"--use-param-defaults",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--last",
				"--use-pipelinerun", "dummy-pipelinerun",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "cannot use --last or --use-pipelinerun options with --use-param-defaults option",
		},
		{
			name: "Dry Run with --use-param-defaults and --use-pipelinerun",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
				"-p=pipeline-param=value1",
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
			name: "Dry Run using --filename v1alpha1",
			command: []string{"start", "-f", "./testdata/pipeline-v1alpha1.yaml",
				"-r=source-repo=scaffold-git",
				"-r=web-image=imageres",
				"-n", "ns",
				"--dry-run",
			},
			namespace:  "",
			input:      c6,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with output=json -f v1alpha1",
			command: []string{"start", "-f", "./testdata/pipeline-v1alpha1.yaml",
				"-r=source-repo=scaffold-git",
				"-r=web-image=imageres",
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
			name: "Start pipeline using --filename",
			command: []string{"start", "-f", "./testdata/pipeline-v1alpha1.yaml",
				"-r=source-repo=scaffold-git",
				"-r=web-image=imageres",
				"-n", "ns",
			},
			namespace: "",
			input:     c6,
			wantError: false,
			want:      "Pipelinerun started: random\n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs random -f -n ns\n",
		},
		{
			name: "Error from using --last with --filename",
			command: []string{"start", "-f", "./testdata/pipeline.yaml",
				"-r=source-repo=scaffold-git",
				"-r=web-image=imageres",
				"-n", "ns",
				"--last",
			},
			namespace: "",
			input:     c6,
			wantError: true,
			want:      "cannot use --last option with --filename option",
		},
		{
			name: "Dry Run with --timeout specified",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
				"-p=pipeline-param=value1",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--timeout", "1s",
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
				test.AssertOutput(t, tp.want, err.Error())
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

func TestPipelineV1beta1Start_ExecuteCommand(t *testing.T) {
	clock := clockwork.NewFakeClock()
	namespaces := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{})
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c1 := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Clock: clock, Resource: cs.Resource}

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
						Resources: &v1beta1.PipelineTaskResources{
							Inputs: []v1beta1.PipelineTaskInputResource{
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
							Outputs: []v1beta1.PipelineTaskOutputResource{
								{
									Name:     "image-to-use",
									Resource: "best-image",
								},
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
						},
					},
				},
				Resources: []v1beta1.PipelineDeclaredResource{
					{
						Name: "git-repo",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
					{
						Name: "build-image",
						Type: v1alpha1.PipelineResourceTypeImage,
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
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
						Resources: &v1beta1.PipelineTaskResources{
							Inputs: []v1beta1.PipelineTaskInputResource{
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
							Outputs: []v1beta1.PipelineTaskOutputResource{
								{
									Name:     "image-to-use",
									Resource: "best-image",
								},
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
						},
					},
				},
			},
		},
	}

	cs2, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Pipelines: pipeline, Namespaces: namespaces})
	cs2.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	tdc2 := testDynamic.Options{}
	dc2, err := tdc2.Client(
		cb.UnstructuredV1beta1P(pipeline[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c2 := &test.Params{Tekton: cs2.Pipeline, Kube: cs2.Kube, Dynamic: dc2, Clock: clock, Resource: cs2.Resource}

	// With list error mocking
	cs3, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Pipelines: pipeline, Namespaces: namespaces})
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
		cb.UnstructuredV1beta1P(pipeline[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c3 := &test.Params{Tekton: cs3.Pipeline, Kube: cs3.Kube, Dynamic: dc3, Clock: clock, Resource: cs3.Resource}

	// With create error mocking
	cs4, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Pipelines: pipeline2, Namespaces: namespaces})
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
		cb.UnstructuredV1beta1P(pipeline2[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c4 := &test.Params{Tekton: cs4.Pipeline, Kube: cs4.Kube, Dynamic: dc4, Clock: clock, Resource: cs4.Resource}

	// Without related pipelinerun
	objs := []runtime.Object{
		pipeline[0],
	}
	seedData, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Pipelines:  pipeline,
		Namespaces: namespaces,
	})
	cs5 := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
	}
	cs5.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	_, tdc5 := newPipelineClient("v1beta1", objs...)
	dc5, err := tdc5.Client(
		cb.UnstructuredV1beta1P(pipeline[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c5 := &test.Params{Tekton: cs5.Pipeline, Kube: cs5.Kube, Dynamic: dc5, Clock: clock, Resource: cs5.Resource}

	// pipelineresources data for tests with --filename
	objs2 := []runtime.Object{}
	pres := []*v1alpha1.PipelineResource{
		tb.PipelineResource("scaffold-git", "ns",
			tb.PipelineResourceSpec("git",
				tb.PipelineResourceSpecParam("url", "git@github.com:tektoncd/cli.git"),
			),
		),
		tb.PipelineResource("imageres", "ns",
			tb.PipelineResourceSpec("image",
				tb.PipelineResourceSpecParam("url", "gcr.io/christiewilson-catfactory/leeroy-web"),
			),
		),
	}
	seedData2, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Namespaces:        namespaces,
		PipelineResources: pres,
	})
	cs6 := pipelinetest.Clients{
		Pipeline: seedData2.Pipeline,
		Kube:     seedData2.Kube,
		Resource: seedData2.Resource,
	}
	cs6.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	_, tdc6 := newPipelineClient("v1beta1", objs2...)
	dc6, err := tdc6.Client()
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	c6 := &test.Params{Tekton: cs6.Pipeline, Kube: cs6.Kube, Dynamic: dc6, Clock: clock, Resource: cs6.Resource}

	testParams := []struct {
		name       string
		command    []string
		namespace  string
		input      *test.Params
		wantError  bool
		want       string
		goldenFile bool
	}{
		{
			name:      "Invalid namespace",
			command:   []string{"start", "pipeline", "-n", "invalid"},
			namespace: "",
			input:     c1,
			wantError: true,
			want:      "namespaces \"invalid\" not found",
		},
		{
			name:      "Missing pipeline name",
			command:   []string{"start", "-n", "ns"},
			namespace: "",
			input:     c1,
			wantError: true,
			want:      "missing pipeline name",
		},
		{
			name:      "Found no pipelines",
			command:   []string{"start", "test-pipeline-2", "-n", "ns"},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "pipeline name test-pipeline-2 does not exist in namespace ns",
		},
		{
			name: "Start pipeline with showlog flag false",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
				"-p=pipeline-param=value1",
				"-l=jemange=desfrites",
				"-w=name=password-vault,secret=secret-name",
				"-n", "ns",
			},
			namespace: "",
			input:     c2,
			wantError: false,
			want:      "Pipelinerun started: \n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs  -f -n ns\n",
		},
		{
			name: "Start pipeline with showlog flag false",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
				"-p=pipeline-param=value1",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=git-repo=scaffold-git",
				"-p=rev-parm=revision2",
				"-n", "ns",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "param 'rev-parm' not present in spec",
		},
		{
			name: "Invalid resource parameter format",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=git-reposcaffold-git",
				"-p=rev-param=revision2",
				"--task-serviceaccount=task3=task3svc3",
				"--task-serviceaccount=task5=task3svc5",
				"-n", "ns",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "invalid input format for resource parameter: git-reposcaffold-git",
		},
		{
			name: "Invalid parameter format",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=git-repo=scaffold-git",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=git-repo=scaffold-git",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=git-repo=scaffold-git",
				"-p=rev-param=revision2",
				"--task-serviceaccount=task3svc3",
				"-n", "ns",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "invalid service account parameter: task3svc3\nPlease pass task service accounts as --task-serviceaccount TaskName=ServiceAccount",
		},
		{
			name: "List error with last flag",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=git-repo=scaffold-git",
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
			command: []string{"start", "test-pipeline",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=git-repo=scaffold-git",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
				"-p=pipeline-param=value1",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
				"-p=pipeline-param=value1",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
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
			name: "Dry Run with --use-param-defaults, --last and --use-pipelinerun",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
				"--use-param-defaults",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--last",
				"--use-pipelinerun", "dummy-pipelinerun",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "cannot use --last or --use-pipelinerun options with --use-param-defaults option",
		},
		{
			name: "Dry Run with --use-param-defaults and --use-pipelinerun",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
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
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
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
			name: "Dry Run using --filename v1beta1",
			command: []string{"start", "-f", "./testdata/pipeline-v1beta1.yaml",
				"-r=source-repo=scaffold-git",
				"-r=web-image=imageres",
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
			command: []string{"start", "-f", "./testdata/pipeline-v1beta1.yaml",
				"-r=source-repo=scaffold-git",
				"-r=web-image=imageres",
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
			name: "Start pipeline using --filename",
			command: []string{"start", "-f", "./testdata/pipeline-v1beta1.yaml",
				"-r=source-repo=scaffold-git",
				"-r=web-image=imageres",
				"-n", "ns",
			},
			namespace: "",
			input:     c6,
			wantError: false,
			want:      "Pipelinerun started: random\n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs random -f -n ns\n",
		},
		{
			name: "Error from using --last with --filename",
			command: []string{"start", "-f", "./testdata/pipeline.yaml",
				"-r=source-repo=scaffold-git",
				"-r=web-image=imageres",
				"-n", "ns",
				"--last",
			},
			namespace: "",
			input:     c6,
			wantError: true,
			want:      "cannot use --last option with --filename option",
		},
		{
			name: "Dry Run with --timeout specified",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
				"-p=pipeline-param=value1",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--timeout", "1s",
			},
			namespace:  "",
			input:      c2,
			wantError:  false,
			goldenFile: true,
		},
		{
			name: "Dry Run with invalid --timeout specified",
			command: []string{"start", "test-pipeline",
				"-s=svc1",
				"-r=source=scaffold-git",
				"-p=pipeline-param=value1",
				"-l=jemange=desfrites",
				"-n", "ns",
				"--dry-run",
				"--timeout", "5d",
			},
			namespace: "",
			input:     c2,
			wantError: true,
			want:      "time: unknown unit d in duration 5d",
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
				test.AssertOutput(t, tp.want, err.Error())
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

func TestPipelineStart_Interactive(t *testing.T) {
	t.Skip("Skipping due of flakiness")

	cs, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline("test-pipeline", "ns",
				tb.PipelineSpec(
					tb.PipelineDeclaredResource("git-repo", "git"),
					tb.PipelineParamSpec("pipeline-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("somethingdifferent")),
					tb.PipelineParamSpec("rev-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("revision")),
					tb.PipelineParamSpec("array-param", v1alpha1.ParamTypeArray, tb.ParamSpecDefault("revision1", "revision2")),
					tb.PipelineTask("unit-test-1", "unit-test-task",
						tb.PipelineTaskInputResource("workspace", "git-repo"),
						tb.PipelineTaskOutputResource("image-to-use", "best-image"),
						tb.PipelineTaskOutputResource("workspace", "git-repo"),
					),
				),
			),
		},
		PipelineResources: []*v1alpha1.PipelineResource{
			tb.PipelineResource("scaffold-git", "ns",
				tb.PipelineResourceSpec("git",
					tb.PipelineResourceSpecParam("url", "git@github.com:tektoncd/cli.git"),
				),
			),
		},
		Namespaces: []*corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "ns",
				},
			},
		},
	})

	// Declared single pipeline resource, but has no resource
	cs2, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline("test-pipeline", "ns",
				tb.PipelineSpec(
					tb.PipelineDeclaredResource("git-repo", "git"),
					tb.PipelineTask("first-create-file", "create-file",
						tb.PipelineTaskInputResource("workspace", "git-repo"),
						tb.PipelineTaskOutputResource("workspace", "source-repo"),
					),
					tb.PipelineTask("then-check", "check-stuff-file-exists",
						tb.PipelineTaskInputResource("workspace", "git-repo", tb.From("first-create-file")),
					),
				),
			),
		},

		Tasks: []*v1alpha1.Task{
			tb.Task("check-stuff-file-exists", "ns",
				tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", "git",
							tb.ResourceTargetPath("newworkspace")),
					),
					tb.Step("ubuntu",
						tb.StepName("read"),
						tb.StepCommand("/bin/bash"),
						tb.StepArgs("-c", "cat", "/workspace/newworkspace/stuff"),
					),
				),
			),
			tb.Task("create-file", "ns",
				tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", "git",
							tb.ResourceTargetPath("damnworkspace")),
					),
					tb.TaskOutputs(
						tb.OutputsResource("workspace", "git"),
					),
					tb.Step("ubuntu",
						tb.StepName("read-docs-old"),
						tb.StepCommand("/bin/bash"),
						tb.StepArgs("-c", "ls -la /workspace/damnworkspace/docs/README.md"),
					),
					tb.Step("ubuntu",
						tb.StepName("write-new-stuff"),
						tb.StepCommand("bash"),
						tb.StepArgs("-c", "ln -s /workspace/damnworkspace /workspace/output/workspace && echo some stuff > /workspace/output/workspace/stuff"),
					),
				),
			),
		},
	})

	// Declared multiple pipeline resource, but has no resource
	cs3, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline("test-pipeline", "ns",
				tb.PipelineSpec(
					tb.PipelineDeclaredResource("git-repo", "git"),
					tb.PipelineDeclaredResource("source-repo", "git"),
					tb.PipelineTask("first-create-file", "create-file",
						tb.PipelineTaskInputResource("workspace", "git-repo"),
						tb.PipelineTaskOutputResource("workspace", "source-repo"),
					),
					tb.PipelineTask("then-check", "check-stuff-file-exists",
						tb.PipelineTaskInputResource("workspace", "git-repo", tb.From("first-create-file")),
					),
				),
			),
		},

		Tasks: []*v1alpha1.Task{
			tb.Task("check-stuff-file-exists", "ns",
				tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", "git",
							tb.ResourceTargetPath("newworkspace")),
					),
					tb.Step("ubuntu",
						tb.StepName("read"),
						tb.StepCommand("/bin/bash"),
						tb.StepArgs("-c", "cat", "/workspace/newworkspace/stuff"),
					),
				),
			),
			tb.Task("create-file", "ns",
				tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", "git",
							tb.ResourceTargetPath("damnworkspace")),
					),
					tb.TaskOutputs(
						tb.OutputsResource("workspace", "git"),
					),
					tb.Step("ubuntu",
						tb.StepName("read-docs-old"),
						tb.StepCommand("/bin/bash"),
						tb.StepArgs("-c", "ls -la /workspace/damnworkspace/docs/README.md"),
					),
					tb.Step("ubuntu",
						tb.StepName("write-new-stuff"),
						tb.StepCommand("bash"),
						tb.StepArgs("-c", "ln -s /workspace/damnworkspace /workspace/output/workspace && echo some stuff > /workspace/output/workspace/stuff"),
					),
				),
			),
		},
	})

	// With single pipeline resource
	cs4, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline("test-pipeline", "ns",
				tb.PipelineSpec(
					tb.PipelineDeclaredResource("git-repo", "git"),
					tb.PipelineTask("first-create-file", "create-file",
						tb.PipelineTaskInputResource("workspace", "git-repo"),
						tb.PipelineTaskOutputResource("workspace", "source-repo"),
					),
					tb.PipelineTask("then-check", "check-stuff-file-exists",
						tb.PipelineTaskInputResource("workspace", "git-repo", tb.From("first-create-file")),
					),
				),
			),
		},

		PipelineResources: []*v1alpha1.PipelineResource{
			tb.PipelineResource("gitres", "ns",
				tb.PipelineResourceSpec("git",
					tb.PipelineResourceSpecParam("url", "https://github.com/GoogleContainerTools/skaffold"),
					tb.PipelineResourceSpecParam("version", "master"),
				),
			),
		},

		Tasks: []*v1alpha1.Task{
			tb.Task("check-stuff-file-exists", "ns",
				tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", "git",
							tb.ResourceTargetPath("newworkspace")),
					),
					tb.Step("ubuntu",
						tb.StepName("read"),
						tb.StepCommand("/bin/bash"),
						tb.StepArgs("-c", "cat", "/workspace/newworkspace/stuff"),
					),
				),
			),
			tb.Task("create-file", "ns",
				tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", "git",
							tb.ResourceTargetPath("damnworkspace")),
					),
					tb.TaskOutputs(
						tb.OutputsResource("workspace", "git"),
					),
					tb.Step("ubuntu",
						tb.StepName("read-docs-old"),
						tb.StepCommand("/bin/bash"),
						tb.StepArgs("-c", "ls -la /workspace/damnworkspace/docs/README.md"),
					),
					tb.Step("ubuntu",
						tb.StepName("write-new-stuff"),
						tb.StepCommand("bash"),
						tb.StepArgs("-c", "ln -s /workspace/damnworkspace /workspace/output/workspace && echo some stuff > /workspace/output/workspace/stuff"),
					),
				),
			),
		},
	})

	// With single pipeline resource, another pipeline name
	cs5, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline("gitpipeline", "ns",
				tb.PipelineSpec(
					tb.PipelineDeclaredResource("git-repo", "git"),
					tb.PipelineTask("first-create-file", "create-file",
						tb.PipelineTaskInputResource("workspace", "git-repo"),
						tb.PipelineTaskOutputResource("workspace", "source-repo"),
					),
					tb.PipelineTask("then-check", "check-stuff-file-exists",
						tb.PipelineTaskInputResource("workspace", "git-repo", tb.From("first-create-file")),
					),
				),
			),
		},

		PipelineResources: []*v1alpha1.PipelineResource{
			tb.PipelineResource("gitres", "ns",
				tb.PipelineResourceSpec("git",
					tb.PipelineResourceSpecParam("url", "https://github.com/GoogleContainerTools/skaffold"),
					tb.PipelineResourceSpecParam("version", "master"),
				),
			),
		},

		Tasks: []*v1alpha1.Task{
			tb.Task("check-stuff-file-exists", "ns",
				tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", "git",
							tb.ResourceTargetPath("newworkspace")),
					),
					tb.Step("ubuntu",
						tb.StepName("read"),
						tb.StepCommand("/bin/bash"),
						tb.StepArgs("-c", "cat", "/workspace/newworkspace/stuff"),
					),
				),
			),
			tb.Task("create-file", "ns",
				tb.TaskSpec(
					tb.TaskInputs(
						tb.InputsResource("workspace", "git",
							tb.ResourceTargetPath("damnworkspace")),
					),
					tb.TaskOutputs(
						tb.OutputsResource("workspace", "git"),
					),
					tb.Step("ubuntu",
						tb.StepName("read-docs-old"),
						tb.StepCommand("/bin/bash"),
						tb.StepArgs("-c", "ls -la /workspace/damnworkspace/docs/README.md"),
					),
					tb.Step("ubuntu",
						tb.StepName("write-new-stuff"),
						tb.StepCommand("bash"),
						tb.StepArgs("-c", "ln -s /workspace/damnworkspace /workspace/output/workspace && echo some stuff > /workspace/output/workspace/stuff"),
					),
				),
			),
		},
	})

	// With image resource
	cs6, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline("imagepipeline", "ns",
				tb.PipelineSpec(
					tb.PipelineDeclaredResource("imageres", "image"),
				),
			),
		},

		PipelineResources: []*v1alpha1.PipelineResource{
			tb.PipelineResource("imageres", "ns",
				tb.PipelineResourceSpec("image",
					tb.PipelineResourceSpecParam("url", "gcr.io/christiewilson-catfactory/leeroy-web"),
				),
			),
		},
	})

	// With image resource, another pipeline name
	cs7, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline("imagepipeline2", "ns",
				tb.PipelineSpec(
					tb.PipelineDeclaredResource("imageres", "image"),
				),
			),
		},

		PipelineResources: []*v1alpha1.PipelineResource{
			tb.PipelineResource("imageres", "ns",
				tb.PipelineResourceSpec("image",
					tb.PipelineResourceSpecParam("url", "gcr.io/christiewilson-catfactory/leeroy-web"),
				),
			),
		},
	})

	// With storage resource
	cs8, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline("storagepipeline", "ns",
				tb.PipelineSpec(
					tb.PipelineDeclaredResource("storageres", "storage"),
				),
			),
		},

		PipelineResources: []*v1alpha1.PipelineResource{
			tb.PipelineResource("storageres", "ns",
				tb.PipelineResourceSpec("storage",
					tb.PipelineResourceSpecParam("type", "gcs"),
					tb.PipelineResourceSpecParam("location", "gs://some-bucket"),
				),
			),
		},
	})

	// With pullRequest resource
	cs9, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline("pullrequestpipeline", "ns",
				tb.PipelineSpec(
					tb.PipelineDeclaredResource("pullreqres", "pullRequest"),
					tb.PipelineTask("unit-test-1", "unit-test-task",
						tb.PipelineTaskInputResource("pullres", "pullreqres"),
					),
				),
			),
		},

		PipelineResources: []*v1alpha1.PipelineResource{
			tb.PipelineResource("pullreqres", "ns",
				tb.PipelineResourceSpec("pullRequest",
					tb.PipelineResourceSpecParam("url", "https://github.com/tektoncd/cli/pull/1"),
				),
			),
		},
	})

	// With cluster resource
	cs10, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline("clusterpipeline", "ns",
				tb.PipelineSpec(
					tb.PipelineDeclaredResource("clusterres", "cluster"),
					tb.PipelineTask("unit-test-1", "unit-test-task",
						tb.PipelineTaskInputResource("clusres", "clusterresource"),
					),
				),
			),
		},

		PipelineResources: []*v1alpha1.PipelineResource{
			tb.PipelineResource("clusterresource", "ns",
				tb.PipelineResourceSpec("cluster",
					tb.PipelineResourceSpecParam("name", "abcClus"),
					tb.PipelineResourceSpecParam("url", "https://10.20.30.40/"),
					tb.PipelineResourceSpecParam("username", "thinkpad"),
					tb.PipelineResourceSpecParam("cadata", "ca"),
				),
			),
		},
	})

	// With cloud resource
	cs11, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline("cloudpipeline", "ns",
				tb.PipelineSpec(
					tb.PipelineDeclaredResource("cloudres", "cloudEvent"),
					tb.PipelineTask("unit-test-1", "unit-test-task",
						tb.PipelineTaskInputResource("clusres", "clusterresource"),
					),
				),
			),
		},

		PipelineResources: []*v1alpha1.PipelineResource{
			tb.PipelineResource("cloudresource", "ns",
				tb.PipelineResourceSpec("cloudEvent",
					tb.PipelineResourceSpecParam("targetURI", "https://10.20.30.40/"),
				),
			),
		},
	})

	cs12, _ := test.SeedTestData(t, pipelinetest.Data{
		Pipelines: []*v1alpha1.Pipeline{
			tb.Pipeline("cloudpipeline", "ns",
				tb.PipelineSpec(
					tb.PipelineWorkspaceDeclaration("pvc", "config", "secret", "emtpyDir"),
					tb.PipelineTask("unit-test-1", "unit-test-task",
						tb.PipelineTaskInputResource("clusres", "clusterresource"),
					),
				),
			),
		},
	})

	testParams := []struct {
		name               string
		namespace          string
		input              pipelinetest.Clients
		last               bool
		serviceAccountName string
		serviceAccounts    []string
		prompt             prompt.Prompt
	}{
		{
			name:               "Start pipeline with selecting git resource, pipeline-param, rev-param and array-param from interactive menu",
			namespace:          "ns",
			input:              cs,
			last:               false,
			serviceAccountName: "svc1",
			serviceAccounts:    []string{"task1=svc1"},
			prompt: prompt.Prompt{
				CmdArgs: []string{"test-pipeline"},
				Procedure: func(c *expect.Console) error {
					if _, err := c.ExpectString("Choose the git resource to use for git-repo:"); err != nil {
						return err
					}

					if _, err := c.ExpectString("scaffold-git (git@github.com:tektoncd/cli.git)"); err != nil {
						return err
					}

					if _, err := c.SendLine(string(terminal.KeyEnter)); err != nil {
						return err
					}

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
			name:               "Create new pipeline resource",
			namespace:          "ns",
			input:              cs2,
			last:               false,
			serviceAccountName: "svc1",
			serviceAccounts:    []string{"task1=svc1"},
			prompt: prompt.Prompt{
				CmdArgs: []string{"test-pipeline"},
				Procedure: func(c *expect.Console) error {
					if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
						return err
					}

					if _, err := c.SendLine("newgitres"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for url :"); err != nil {
						return err
					}

					if _, err := c.SendLine("https://github.com/GoogleContainerTools/skaffold"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for revision :"); err != nil {
						return err
					}

					if _, err := c.SendLine("master"); err != nil {
						return err
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					tekton := cs2.Pipeline.TektonV1alpha1()
					runs, err := tekton.PipelineRuns("ns").List(v1.ListOptions{})
					if err != nil {
						return err
					}

					if runs.Items != nil && runs.Items[0].Spec.PipelineRef.Name != "test-pipeline" {
						return errors.New("pipelinerun not found")
					}

					c.Close()
					return nil
				},
			},
		},
		{
			name:               "Create multiple, new pipeline resources",
			namespace:          "ns",
			input:              cs3,
			last:               false,
			serviceAccountName: "svc1",
			serviceAccounts:    []string{"task1=svc1"},
			prompt: prompt.Prompt{
				CmdArgs: []string{"test-pipeline"},
				Procedure: func(c *expect.Console) error {
					if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
						return err
					}

					if _, err := c.SendLine("newgitres"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for url :"); err != nil {
						return err
					}

					if _, err := c.SendLine("https://github.com/GoogleContainerTools/skaffold"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for revision :"); err != nil {
						return err
					}

					if _, err := c.SendLine("master"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
						return err
					}

					if _, err := c.SendLine("newgitres2"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for url :"); err != nil {
						return err
					}

					if _, err := c.SendLine("https://github.com/GoogleContainerTools/skaffold2"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for revision :"); err != nil {
						return err
					}

					if _, err := c.SendLine("master"); err != nil {
						return err
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					tekton := cs3.Pipeline.TektonV1alpha1()
					runs, err := tekton.PipelineRuns("ns").List(v1.ListOptions{})
					if err != nil {
						return err
					}

					if runs.Items != nil && runs.Items[0].Spec.PipelineRef.Name != "test-pipeline" {
						return errors.New("pipelinerun not found")
					}

					c.Close()
					return nil
				},
			},
		},
		{
			name:               "Select existing pipeline resource",
			namespace:          "ns",
			input:              cs4,
			last:               false,
			serviceAccountName: "svc1",
			serviceAccounts:    []string{"task1=svc1"},
			prompt: prompt.Prompt{
				CmdArgs: []string{"test-pipeline"},
				Procedure: func(c *expect.Console) error {
					if _, err := c.ExpectString("Choose the git resource to use for git-repo"); err != nil {
						return err
					}

					if _, err := c.ExpectString("gitres (https://github.com/GoogleContainerTools/skaffold)"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					tekton := cs4.Pipeline.TektonV1alpha1()
					runs, err := tekton.PipelineRuns("ns").List(v1.ListOptions{})
					if err != nil {
						return err
					}

					if runs.Items != nil && runs.Items[0].Spec.PipelineRef.Name != "test-pipeline" {
						return errors.New("pipelinerun not found")
					}

					c.Close()
					return nil
				},
			},
		},
		{
			name:               "Create new pipeline resource with existing resource",
			namespace:          "ns",
			input:              cs5,
			last:               false,
			serviceAccountName: "svc1",
			serviceAccounts:    []string{"task1=svc1"},
			prompt: prompt.Prompt{
				CmdArgs: []string{"gitpipeline"},
				Procedure: func(c *expect.Console) error {
					if _, err := c.ExpectString("Choose the git resource to use for git-repo"); err != nil {
						return err
					}

					if _, err := c.ExpectString("gitres (https://github.com/GoogleContainerTools/skaffold)"); err != nil {
						return err
					}
					if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("create new \"git\" resource"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
						return err
					}

					if _, err := c.SendLine("newgitres"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for url :"); err != nil {
						return err
					}

					if _, err := c.SendLine("https://github.com/GoogleContainerTools/skaffold"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for revision :"); err != nil {
						return err
					}

					if _, err := c.SendLine("master"); err != nil {
						return err
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					tekton := cs5.Pipeline.TektonV1alpha1()
					runs, err := tekton.PipelineRuns("ns").List(v1.ListOptions{})
					if err != nil {
						return err
					}

					if runs.Items != nil && runs.Items[0].Spec.PipelineRef.Name != "gitpipeline" {
						return errors.New("pipelinerun not found")
					}

					c.Close()
					return nil
				},
			},
		},
		{
			name:               "Select existing image resource",
			namespace:          "ns",
			input:              cs6,
			last:               false,
			serviceAccountName: "svc1",
			serviceAccounts:    []string{"task1=svc1"},
			prompt: prompt.Prompt{
				CmdArgs: []string{"imagepipeline"},
				Procedure: func(c *expect.Console) error {
					if _, err := c.ExpectString("Choose the image resource to use for imageres"); err != nil {
						return err
					}

					if _, err := c.ExpectString("imageres (gcr.io/christiewilson-catfactory/leeroy-web"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					tekton := cs6.Pipeline.TektonV1alpha1()
					runs, err := tekton.PipelineRuns("ns").List(v1.ListOptions{})
					if err != nil {
						return err
					}

					if runs.Items != nil && runs.Items[0].Spec.PipelineRef.Name != "imagepipeline" {
						return errors.New("pipelinerun not found")
					}

					c.Close()
					return nil
				},
			},
		},
		{
			name:               "Create new image resource with existing resource",
			namespace:          "ns",
			input:              cs7,
			last:               false,
			serviceAccountName: "svc1",
			serviceAccounts:    []string{"task1=svc1"},
			prompt: prompt.Prompt{
				CmdArgs: []string{"imagepipeline2"},
				Procedure: func(c *expect.Console) error {
					if _, err := c.ExpectString("Choose the image resource to use for imageres"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("create new \"image\" resource"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
						return err
					}

					if _, err := c.SendLine("newimageres"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for url :"); err != nil {
						return err
					}

					if _, err := c.SendLine("gcr.io/christiewilson-catfactory/leeroy-web"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for digest :"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					tekton := cs7.Pipeline.TektonV1alpha1()
					runs, err := tekton.PipelineRuns("ns").List(v1.ListOptions{})
					if err != nil {
						return err
					}

					if runs.Items != nil && runs.Items[0].Spec.PipelineRef.Name != "imagepipeline2" {
						return errors.New("pipelinerun not found")
					}

					c.Close()
					return nil
				},
			},
		},
		{
			name:               "Create new storage resource with existing resource",
			namespace:          "ns",
			input:              cs8,
			last:               false,
			serviceAccountName: "svc1",
			serviceAccounts:    []string{"task1=svc1"},
			prompt: prompt.Prompt{
				CmdArgs: []string{"storagepipeline"},
				Procedure: func(c *expect.Console) error {
					if _, err := c.ExpectString("Choose the storage resource to use for storageres"); err != nil {
						return err
					}

					if _, err := c.ExpectString("storageres (gs://some-bucket)"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("create new \"storage\" resource"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
						return err
					}

					if _, err := c.SendLine("new"); err != nil {
						return err
					}

					if _, err := c.ExpectString("gcs"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for location :"); err != nil {
						return err
					}

					if _, err := c.SendLine("gs://some-bucket"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for dir :"); err != nil {
						return err
					}

					if _, err := c.SendLine("/home"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Secret Key for GOOGLE_APPLICATION_CREDENTIALS :"); err != nil {
						return err
					}

					if _, err := c.SendLine("service_account.json"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Secret Name for GOOGLE_APPLICATION_CREDENTIALS :"); err != nil {
						return err
					}

					if _, err := c.SendLine("bucket-sa"); err != nil {
						return err
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					tekton := cs8.Pipeline.TektonV1alpha1()
					runs, err := tekton.PipelineRuns("ns").List(v1.ListOptions{})
					if err != nil {
						return err
					}

					if runs.Items != nil && runs.Items[0].Spec.PipelineRef.Name != "storagepipeline" {
						return errors.New("pipelinerun not found")
					}

					c.Close()
					return nil
				},
			},
		},
		{
			name:               "Create new pullRequest resource with existing resource",
			namespace:          "ns",
			input:              cs9,
			last:               false,
			serviceAccountName: "svc1",
			serviceAccounts:    []string{"task1=svc1"},
			prompt: prompt.Prompt{
				CmdArgs: []string{"pullrequestpipeline"},
				Procedure: func(c *expect.Console) error {
					if _, err := c.ExpectString("Choose the pullRequest resource to use for pullreqres"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("create new \"pullRequest\" resource"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
						return err
					}

					if _, err := c.SendLine("newpullreq"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for url :"); err != nil {
						return err
					}

					if _, err := c.SendLine("https://github.com/tektoncd/cli/pull/1"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Do you want to set secrets ?"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Yes"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Secret Key for githubToken"); err != nil {
						return err
					}

					if _, err := c.SendLine("githubToken"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Secret Name for githubToken "); err != nil {
						return err
					}

					if _, err := c.SendLine("githubTokenName"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					tekton := cs9.Pipeline.TektonV1alpha1()
					runs, err := tekton.PipelineRuns("ns").List(v1.ListOptions{})
					if err != nil {
						return err
					}

					if runs.Items != nil && runs.Items[0].Spec.PipelineRef.Name != "pullrequestpipeline" {
						return errors.New("pipelinerun not found")
					}

					c.Close()
					return nil
				},
			},
		},
		{
			name:               "Create new cluster resource with existing resource",
			namespace:          "ns",
			input:              cs10,
			last:               false,
			serviceAccountName: "svc1",
			serviceAccounts:    []string{"task1=svc1"},
			prompt: prompt.Prompt{
				CmdArgs: []string{"clusterpipeline"},
				Procedure: func(c *expect.Console) error {
					if _, err := c.ExpectString("Choose the cluster resource to use for clusterres"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("create new \"cluster\" resource"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
						return err
					}

					if _, err := c.SendLine("newclusterresource"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for url :"); err != nil {
						return err
					}

					if _, err := c.SendLine("https://10.10.10.10"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for username :"); err != nil {
						return err
					}

					if _, err := c.SendLine("user"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Is the cluster secure?"); err != nil {
						return err
					}

					if _, err := c.ExpectString("yes"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Which authentication technique you want to use?"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for password :"); err != nil {
						return err
					}

					if _, err := c.SendLine("abcd#@123"); err != nil {
						return err
					}

					if _, err := c.ExpectString("*********"); err != nil {
						return err
					}

					if _, err := c.ExpectString("How do you want to set cadata?"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Passing plain text as parameters"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for cadata :"); err != nil {
						return err
					}

					if _, err := c.SendLine("cadata"); err != nil {
						return err
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					tekton := cs10.Pipeline.TektonV1alpha1()
					runs, err := tekton.PipelineRuns("ns").List(v1.ListOptions{})
					if err != nil {
						return err
					}

					if runs.Items != nil && runs.Items[0].Spec.PipelineRef.Name != "clusterpipeline" {
						return errors.New("pipelinerun not found")
					}

					c.Close()
					return nil
				},
			},
		},
		{
			name:               "Create new cloud resource with existing resource",
			namespace:          "ns",
			input:              cs11,
			last:               false,
			serviceAccountName: "svc1",
			serviceAccounts:    []string{"task1=svc1"},
			prompt: prompt.Prompt{
				CmdArgs: []string{"cloudpipeline"},
				Procedure: func(c *expect.Console) error {
					if _, err := c.ExpectString("Choose the cloudEvent resource to use for cloudres"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyArrowDown)); err != nil {
						return err
					}

					if _, err := c.ExpectString("create new \"cloudEvent\" resource"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a name for a pipeline resource :"); err != nil {
						return err
					}

					if _, err := c.SendLine("newcloudresource"); err != nil {
						return err
					}

					if _, err := c.ExpectString("Enter a value for targetURI :"); err != nil {
						return err
					}

					if _, err := c.SendLine("https://10.10.10.10"); err != nil {
						return err
					}

					if _, err := c.Send(string(terminal.KeyEnter)); err != nil {
						return err
					}

					if _, err := c.ExpectEOF(); err != nil {
						return err
					}

					tekton := cs11.Pipeline.TektonV1alpha1()
					runs, err := tekton.PipelineRuns("ns").List(v1.ListOptions{})
					if err != nil {
						return err
					}

					if runs.Items != nil && runs.Items[0].Spec.PipelineRef.Name != "cloudpipeline" {
						return errors.New("pipelinerun not found")
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

					if _, err := c.SendLine("emtpyDir"); err != nil {
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

					if _, err := c.ExpectString("Type of EmtpyDir :"); err != nil {
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
				Kube:     tp.input.Kube,
				Tekton:   tp.input.Pipeline,
				Resource: tp.input.Resource,
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

func Test_start_pipeline(t *testing.T) {
	pipelineName := "test-pipeline"

	pipeline := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName, "ns",
			tb.PipelineSpec(
				tb.PipelineDeclaredResource("git-repo", "git"),
				tb.PipelineParamSpec("pipeline-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("somethingdifferent")),
				tb.PipelineParamSpec("rev-param", v1alpha1.ParamTypeArray, tb.ParamSpecDefault("booms", "booms", "booms")),
				tb.PipelineTask("unit-test-1", "unit-test-task",
					tb.PipelineTaskInputResource("workspace", "git-repo"),
					tb.PipelineTaskOutputResource("image-to-use", "best-image"),
					tb.PipelineTaskOutputResource("workspace", "git-repo"),
				),
			), // spec
		), // pipeline
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Pipelines: pipeline, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(pipeline[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}
	c := Command(p)

	got, _ := test.ExecuteCommand(c, "start", pipelineName,
		"-r=source=scaffold-git",
		"-p=pipeline-param=value1",
		"-p=rev-param=cat,foo,bar",
		"-l=jemange=desfrites",
		"-s=svc1",
		"-n", "ns")

	expected := "Pipelinerun started: \n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs  -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.List(cl, metav1.ListOptions{}, "ns")
	if err != nil {
		t.Errorf("Error listing pipelineruns %s", err.Error())
	}

	if pr.Items[0].ObjectMeta.GenerateName != (pipelineName + "-run-") {
		t.Errorf("Error pipelinerun generated is different %+v", pr)
	}

	test.AssertOutput(t, 2, len(pr.Items[0].Spec.Params))

	for _, v := range pr.Items[0].Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeArray, ArrayVal: []string{"cat", "foo", "bar"}}, v.Value)
		}
	}

	if d := cmp.Equal(pr.Items[0].ObjectMeta.Labels, map[string]string{"jemange": "desfrites"}); !d {
		t.Errorf("Error labels generated is different Labels Got: %+v", pr.Items[0].ObjectMeta.Labels)
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
						Resources: &v1beta1.PipelineTaskResources{
							Inputs: []v1beta1.PipelineTaskInputResource{
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
							Outputs: []v1beta1.PipelineTaskOutputResource{
								{
									Name:     "image-to-use",
									Resource: "best-image",
								},
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
						},
					},
				},
				Resources: []v1beta1.PipelineDeclaredResource{
					{
						Name: "git-repo",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeArray,
						Default: &v1beta1.ArrayOrString{
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

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Pipelines: pipeline, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(pipeline[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}
	c := Command(p)

	got, _ := test.ExecuteCommand(c, "start", pipelineName,
		"-r=source=scaffold-git",
		"-p=pipeline-param=value1",
		"-p=rev-param=cat,foo,bar",
		"-l=jemange=desfrites",
		"-s=svc1",
		"-n", "ns")

	expected := "Pipelinerun started: \n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs  -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.List(cl, metav1.ListOptions{}, "ns")
	if err != nil {
		t.Errorf("Error listing pipelineruns %s", err.Error())
	}

	if pr.Items[0].ObjectMeta.GenerateName != (pipelineName + "-run-") {
		t.Errorf("Error pipelinerun generated is different %+v", pr)
	}

	test.AssertOutput(t, 2, len(pr.Items[0].Spec.Params))

	for _, v := range pr.Items[0].Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeArray, ArrayVal: []string{"cat", "foo", "bar"}}, v.Value)
		}
	}

	if d := cmp.Equal(pr.Items[0].ObjectMeta.Labels, map[string]string{"jemange": "desfrites"}); !d {
		t.Errorf("Error labels generated is different Labels Got: %+v", pr.Items[0].ObjectMeta.Labels)
	}

}

func Test_start_pipeline_last(t *testing.T) {

	pipelineName := "test-pipeline"

	ps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName, "ns",
			tb.PipelineSpec(
				tb.PipelineDeclaredResource("git-repo", "git"),
				tb.PipelineDeclaredResource("build-image", "image"),
				tb.PipelineWorkspaceDeclaration("test=workspace"),
				tb.PipelineParamSpec("pipeline-param-1", v1alpha1.ParamTypeString, tb.ParamSpecDefault("somethingdifferent-1")),
				tb.PipelineParamSpec("rev-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("revision")),
				tb.PipelineTask("unit-test-1", "unit-test-task",
					tb.PipelineTaskInputResource("workspace", "git-repo"),
					tb.PipelineTaskOutputResource("image-to-use", "best-image"),
					tb.PipelineTaskOutputResource("workspace", "git-repo"),
					tb.PipelineTaskWorkspaceBinding("task-test-workspace", "test-workspace"),
				),
			), // spec
		), // pipeline
	}

	timeoutDuration, _ := time.ParseDuration("10s")

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun("test-pipeline-run-123", "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
			tb.PipelineRunSpec(pipelineName,
				tb.PipelineRunServiceAccountName("test-sa"),
				tb.PipelineRunResourceBinding("git-repo", tb.PipelineResourceBindingRef("some-repo")),
				tb.PipelineRunResourceBinding("build-image", tb.PipelineResourceBindingRef("some-image")),
				tb.PipelineRunParam("pipeline-param-1", "somethingmorefun"),
				tb.PipelineRunParam("rev-param", "revision1"),
				tb.PipelineRunWorkspaceBindingEmptyDir("test-new"),
				tb.PipelineRunTimeout(timeoutDuration),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	//Add namespaces to kube client
	seedData, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newPipelineClient("v1alpha1", objs...)
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], "v1alpha1"),
		cb.UnstructuredPR(prs[0], "v1alpha1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-n", "ns",
	)

	expected := "Pipelinerun started: random\n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.Get(cl, "random", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	for _, v := range pr.Spec.Resources {
		if v.Name == "git-repo" {
			test.AssertOutput(t, "some-repo", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, 2, len(pr.Spec.Params))
	for _, v := range pr.Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeString, StringVal: "revision1"}, v.Value)
		}
	}

	test.AssertOutput(t, "test-sa", pr.Spec.ServiceAccountName)
	test.AssertOutput(t, "test-new", pr.Spec.Workspaces[0].Name)
	test.AssertOutput(t, timeoutDuration, pr.Spec.Timeout.Duration)
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
						Resources: &v1beta1.PipelineTaskResources{
							Inputs: []v1beta1.PipelineTaskInputResource{
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
							Outputs: []v1beta1.PipelineTaskOutputResource{
								{
									Name:     "image-to-use",
									Resource: "best-image",
								},
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
						},
						Workspaces: []v1beta1.WorkspacePipelineTaskBinding{
							{
								Name:      "task-test-workspace",
								Workspace: "test-workspace",
							},
						},
					},
				},
				Resources: []v1beta1.PipelineDeclaredResource{
					{
						Name: "git-repo",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
					{
						Name: "build-image",
						Type: v1alpha1.PipelineResourceTypeImage,
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision",
						},
					},
				},
				Workspaces: []v1beta1.WorkspacePipelineDeclaration{
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
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "git-repo",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-repo",
						},
					},
					{
						Name: "build-image",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-image",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ArrayOrString{
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
				Timeout: &metav1.Duration{Duration: timeoutDuration},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
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

	//Add namespaces to kube client
	seedData, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newPipelineClient("v1beta1", objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-n", "ns",
	)

	expected := "Pipelinerun started: random\n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.Get(cl, "random", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	for _, v := range pr.Spec.Resources {
		if v.Name == "git-repo" {
			test.AssertOutput(t, "some-repo", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, 2, len(pr.Spec.Params))
	for _, v := range pr.Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeString, StringVal: "revision1"}, v.Value)
		}
	}

	test.AssertOutput(t, "test-sa", pr.Spec.ServiceAccountName)
	test.AssertOutput(t, "test-new", pr.Spec.Workspaces[0].Name)
	test.AssertOutput(t, timeoutDuration, pr.Spec.Timeout.Duration)
}

func Test_start_pipeline_last_without_res_param(t *testing.T) {

	pipelineName := "test-pipeline"

	ps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName, "ns",
			tb.PipelineSpec(
				tb.PipelineDeclaredResource("git-repo", "git"),
				tb.PipelineDeclaredResource("build-image", "image"),
				tb.PipelineParamSpec("pipeline-param-1", v1alpha1.ParamTypeString, tb.ParamSpecDefault("somethingdifferent-1")),
				tb.PipelineParamSpec("rev-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("revision")),
				tb.PipelineTask("unit-test-1", "unit-test-task",
					tb.PipelineTaskInputResource("workspace", "git-repo"),
					tb.PipelineTaskOutputResource("image-to-use", "best-image"),
					tb.PipelineTaskOutputResource("workspace", "git-repo"),
				),
			), // spec
		), // pipeline
	}

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun("test-pipeline-run-123", "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
			tb.PipelineRunSpec(pipelineName,
				tb.PipelineRunServiceAccountName("test-sa"),
				tb.PipelineRunResourceBinding("git-repo", tb.PipelineResourceBindingRef("some-repo")),
				tb.PipelineRunResourceBinding("build-image", tb.PipelineResourceBindingRef("some-image")),
				tb.PipelineRunParam("pipeline-param-1", "somethingmorefun"),
				tb.PipelineRunParam("rev-param", "revision1"),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	//Add namespaces to kube client
	seedData, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newPipelineClient("v1alpha1", objs...)
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], "v1alpha1"),
		cb.UnstructuredPR(prs[0], "v1alpha1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-n", "ns")

	expected := "Pipelinerun started: random\n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.Get(cl, "random", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	for _, v := range pr.Spec.Resources {
		if v.Name == "git-repo" {
			test.AssertOutput(t, "some-repo", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, 2, len(pr.Spec.Params))

	for _, v := range pr.Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeString, StringVal: "revision1"}, v.Value)
		}
	}
	test.AssertOutput(t, "test-sa", pr.Spec.ServiceAccountName)
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
						Resources: &v1beta1.PipelineTaskResources{
							Inputs: []v1beta1.PipelineTaskInputResource{
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
							Outputs: []v1beta1.PipelineTaskOutputResource{
								{
									Name:     "image-to-use",
									Resource: "best-image",
								},
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
						},
					},
				},
				Resources: []v1beta1.PipelineDeclaredResource{
					{
						Name: "git-repo",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
					{
						Name: "build-image",
						Type: v1alpha1.PipelineResourceTypeImage,
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
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
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "git-repo",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-repo",
						},
					},
					{
						Name: "build-image",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-image",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
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

	//Add namespaces to kube client
	seedData, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newPipelineClient("v1beta1", objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-n", "ns")

	expected := "Pipelinerun started: random\n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.Get(cl, "random", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	for _, v := range pr.Spec.Resources {
		if v.Name == "git-repo" {
			test.AssertOutput(t, "some-repo", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, 2, len(pr.Spec.Params))

	for _, v := range pr.Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeString, StringVal: "revision1"}, v.Value)
		}
	}
	test.AssertOutput(t, "test-sa", pr.Spec.ServiceAccountName)
}

func Test_start_pipeline_last_merge(t *testing.T) {

	pipelineName := "test-pipeline"

	ps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName, "ns",
			tb.PipelineSpec(
				tb.PipelineDeclaredResource("git-repo", "git"),
				tb.PipelineDeclaredResource("build-image", "image"),
				tb.PipelineParamSpec("pipeline-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("somethingdifferent-1")),
				tb.PipelineParamSpec("rev-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("revision")),
				tb.PipelineTask("unit-test-1", "unit-test-task",
					tb.PipelineTaskInputResource("workspace", "git-repo"),
					tb.PipelineTaskOutputResource("image-to-use", "best-image"),
					tb.PipelineTaskOutputResource("workspace", "git-repo"),
				),
			), // spec
		), // pipeline
	}

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun("test-pipeline-run-123", "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
			tb.PipelineRunSpec(pipelineName,
				tb.PipelineRunServiceAccountName("test-sa"),
				tb.PipelineRunServiceAccountNameTask("task1", "task1svc"),
				tb.PipelineRunServiceAccountNameTask("task3", "task3svc"),
				tb.PipelineRunResourceBinding("git-repo", tb.PipelineResourceBindingRef("some-repo")),
				tb.PipelineRunResourceBinding("build-image", tb.PipelineResourceBindingRef("some-image")),
				tb.PipelineRunParam("pipeline-param-1", "somethingmorefun"),
				tb.PipelineRunParam("rev-param", "revision1"),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	//Add namespaces to kube client
	seedData, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newPipelineClient("v1alpha1", objs...)
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], "v1alpha1"),
		cb.UnstructuredPR(prs[0], "v1alpha1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-s=svc1",
		"-r=git-repo=scaffold-git",
		"-p=rev-param=revision2",
		"--task-serviceaccount=task3=task3svc3",
		"--task-serviceaccount=task5=task3svc5",
		"-n=ns")

	expected := "Pipelinerun started: random\n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.Get(cl, "random", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	for _, v := range pr.Spec.Resources {
		if v.Name == "git-repo" {
			test.AssertOutput(t, "scaffold-git", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, 2, len(pr.Spec.Params))

	for _, v := range pr.Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeString, StringVal: "revision2"}, v.Value)
		}
	}

	for _, v := range pr.Spec.ServiceAccountNames {
		if v.TaskName == "task3" {
			test.AssertOutput(t, "task3svc3", v.ServiceAccountName)
		}
	}

	test.AssertOutput(t, "svc1", pr.Spec.ServiceAccountName)
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
						Resources: &v1beta1.PipelineTaskResources{
							Inputs: []v1beta1.PipelineTaskInputResource{
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
							Outputs: []v1beta1.PipelineTaskOutputResource{
								{
									Name:     "image-to-use",
									Resource: "best-image",
								},
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
						},
					},
				},
				Resources: []v1beta1.PipelineDeclaredResource{
					{
						Name: "git-repo",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
					{
						Name: "build-image",
						Type: v1alpha1.PipelineResourceTypeImage,
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
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
				ServiceAccountNames: []v1beta1.PipelineRunSpecServiceAccountName{
					{
						TaskName:           "task1",
						ServiceAccountName: "task1svc",
					},
					{
						TaskName:           "task3",
						ServiceAccountName: "task3svc",
					},
				},
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "git-repo",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-repo",
						},
					},
					{
						Name: "build-image",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-image",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
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

	//Add namespaces to kube client
	seedData, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newPipelineClient("v1beta1", objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-s=svc1",
		"-r=git-repo=scaffold-git",
		"-p=rev-param=revision2",
		"--task-serviceaccount=task3=task3svc3",
		"--task-serviceaccount=task5=task3svc5",
		"-n=ns")

	expected := "Pipelinerun started: random\n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.Get(cl, "random", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	for _, v := range pr.Spec.Resources {
		if v.Name == "git-repo" {
			test.AssertOutput(t, "scaffold-git", v.ResourceRef.Name)
		}
	}

	test.AssertOutput(t, 2, len(pr.Spec.Params))

	for _, v := range pr.Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeString, StringVal: "revision2"}, v.Value)
		}
	}

	for _, v := range pr.Spec.ServiceAccountNames {
		if v.TaskName == "task3" {
			test.AssertOutput(t, "task3svc3", v.ServiceAccountName)
		}
	}

	test.AssertOutput(t, "svc1", pr.Spec.ServiceAccountName)
}

func Test_start_pipeline_use_pipelinerun(t *testing.T) {

	pipelineName := "test-pipeline"

	ps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName, "ns",
			tb.PipelineSpec(
				tb.PipelineDeclaredResource("git-repo", "git"),
				tb.PipelineDeclaredResource("build-image", "image"),
				tb.PipelineParamSpec("pipeline-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("somethingdifferent-1")),
				tb.PipelineParamSpec("rev-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("revision")),
				tb.PipelineTask("unit-test-1", "unit-test-task",
					tb.PipelineTaskInputResource("workspace", "git-repo"),
					tb.PipelineTaskOutputResource("image-to-use", "best-image"),
					tb.PipelineTaskOutputResource("workspace", "git-repo"),
				),
			), // spec
		), // pipeline
	}

	timeoutDuration, _ := time.ParseDuration("10s")
	theonename := "test-pipeline-run-be-the-one"
	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun("dont-bother-me-trying", "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
			tb.PipelineRunSpec(pipelineName),
		),
		tb.PipelineRun(theonename, "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
			tb.PipelineRunSpec(pipelineName,
				tb.PipelineRunParam("brush", "teeth"),
				tb.PipelineRunTimeout(timeoutDuration),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	//Add namespaces to kube client
	seedData, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0], prs[1]}
	_, tdc := newPipelineClient("v1alpha1", objs...)
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], "v1alpha1"),
		cb.UnstructuredPR(prs[0], "v1alpha1"),
		cb.UnstructuredPR(prs[1], "v1alpha1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}

	pipeline := Command(p)
	// There is no point to checkout otuput since we would be checking if our testdata works!
	_, _ = test.ExecuteCommand(pipeline, "start", pipelineName,
		"--use-pipelinerun="+theonename, "-n", "ns")

	cl, _ := p.Clients()
	pr, err := pipelinerun.Get(cl, "random", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}
	test.AssertOutput(t, pr.Spec.Params[0].Name, "brush")
	test.AssertOutput(t, pr.Spec.Params[0].Value, v1alpha1.ArrayOrString{Type: "string", StringVal: "teeth"})
	test.AssertOutput(t, timeoutDuration, pr.Spec.Timeout.Duration)
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
						Resources: &v1beta1.PipelineTaskResources{
							Inputs: []v1beta1.PipelineTaskInputResource{
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
							Outputs: []v1beta1.PipelineTaskOutputResource{
								{
									Name:     "image-to-use",
									Resource: "best-image",
								},
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
						},
					},
				},
				Resources: []v1beta1.PipelineDeclaredResource{
					{
						Name: "git-repo",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
					{
						Name: "build-image",
						Type: v1alpha1.PipelineResourceTypeImage,
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
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
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
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
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "teeth",
						},
					},
				},
				Timeout: &metav1.Duration{Duration: timeoutDuration},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
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

	//Add namespaces to kube client
	seedData, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0], prs[1]}
	_, tdc := newPipelineClient("v1beta1", objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[1], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}

	pipeline := Command(p)
	// There is no point to checkout otuput since we would be checking if our testdata works!
	_, _ = test.ExecuteCommand(pipeline, "start", pipelineName,
		"--use-pipelinerun="+theonename, "-n", "ns")

	cl, _ := p.Clients()
	pr, err := pipelinerun.Get(cl, "random", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}
	test.AssertOutput(t, pr.Spec.Params[0].Name, "brush")
	test.AssertOutput(t, pr.Spec.Params[0].Value, v1alpha1.ArrayOrString{Type: "string", StringVal: "teeth"})
	test.AssertOutput(t, timeoutDuration, pr.Spec.Timeout.Duration)
}

func Test_start_pipeline_allkindparam(t *testing.T) {
	pipelineName := "test-pipeline"

	ps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName, "ns",
			tb.PipelineSpec(
				tb.PipelineDeclaredResource("git-repo", "git"),
				tb.PipelineParamSpec("pipeline-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("somethingdifferent")),
				tb.PipelineParamSpec("rev-param", v1alpha1.ParamTypeArray, tb.ParamSpecDefault("booms", "booms", "booms")),
				tb.PipelineParamSpec("rev-param-new", v1alpha1.ParamTypeArray, tb.ParamSpecDefault("booms", "booms", "booms")),
				tb.PipelineTask("unit-test-1", "unit-test-task",
					tb.PipelineTaskInputResource("workspace", "git-repo"),
					tb.PipelineTaskOutputResource("image-to-use", "best-image"),
					tb.PipelineTaskOutputResource("workspace", "git-repo"),
				),
			), // spec
		), // pipeline
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{Pipelines: ps, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], versionA1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource, Dynamic: dc}
	pipeline := Command(p)

	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"-r=source=scaffold-git",
		"-p=pipeline-param=value1",
		"-p=rev-param=cat,foo,bar",
		"-p=rev-param-new=help",
		"-l=jemange=desfrites",
		"-s=svc1",
		"-n", "ns")

	expected := "Pipelinerun started: \n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs  -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.List(cl, v1.ListOptions{}, "ns")
	if err != nil {
		t.Errorf("Error listing pipelineruns %s", err.Error())
	}

	if pr.Items[0].ObjectMeta.GenerateName != (pipelineName + "-run-") {
		t.Errorf("Error pipelinerun generated is different %+v", pr)
	}

	test.AssertOutput(t, 3, len(pr.Items[0].Spec.Params))
	for _, v := range pr.Items[0].Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeArray, ArrayVal: []string{"cat", "foo", "bar"}}, v.Value)
		}

		if v.Name == "rev-param-new" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeArray, ArrayVal: []string{"help"}}, v.Value)
		}
	}

	if d := cmp.Equal(pr.Items[0].ObjectMeta.Labels, map[string]string{"jemange": "desfrites"}); !d {
		t.Errorf("Error labels generated is different Labels Got: %+v", pr.Items[0].ObjectMeta.Labels)
	}

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
						Resources: &v1beta1.PipelineTaskResources{
							Inputs: []v1beta1.PipelineTaskInputResource{
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
							Outputs: []v1beta1.PipelineTaskOutputResource{
								{
									Name:     "image-to-use",
									Resource: "best-image",
								},
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
						},
					},
				},
				Resources: []v1beta1.PipelineDeclaredResource{
					{
						Name: "git-repo",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeArray,
						Default: &v1beta1.ArrayOrString{
							Type:     v1beta1.ParamTypeArray,
							ArrayVal: []string{"booms", "booms", "booms"},
						},
					},
					{
						Name: "rev-param-new",
						Type: v1beta1.ParamTypeArray,
						Default: &v1beta1.ArrayOrString{
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

	cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Pipelines: ps, Namespaces: ns})
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	tdc := testDynamic.Options{}
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], versionB1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Resource: cs.Resource, Dynamic: dc}
	pipeline := Command(p)

	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"-r=source=scaffold-git",
		"-p=pipeline-param=value1",
		"-p=rev-param=cat,foo,bar",
		"-p=rev-param-new=help",
		"-l=jemange=desfrites",
		"-s=svc1",
		"-n", "ns")

	expected := "Pipelinerun started: \n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs  -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.List(cl, v1.ListOptions{}, "ns")
	if err != nil {
		t.Errorf("Error listing pipelineruns %s", err.Error())
	}

	if pr.Items[0].ObjectMeta.GenerateName != (pipelineName + "-run-") {
		t.Errorf("Error pipelinerun generated is different %+v", pr)
	}

	test.AssertOutput(t, 3, len(pr.Items[0].Spec.Params))
	for _, v := range pr.Items[0].Spec.Params {
		if v.Name == "rev-param" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeArray, ArrayVal: []string{"cat", "foo", "bar"}}, v.Value)
		}

		if v.Name == "rev-param-new" {
			test.AssertOutput(t, v1alpha1.ArrayOrString{Type: v1alpha1.ParamTypeArray, ArrayVal: []string{"help"}}, v.Value)
		}
	}

	if d := cmp.Equal(pr.Items[0].ObjectMeta.Labels, map[string]string{"jemange": "desfrites"}); !d {
		t.Errorf("Error labels generated is different Labels Got: %+v", pr.Items[0].ObjectMeta.Labels)
	}

}

func Test_start_pipeline_last_generate_name(t *testing.T) {

	pipelineName := "test-pipeline"

	ps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName, "ns",
			tb.PipelineSpec(
				tb.PipelineDeclaredResource("git-repo", "git"),
				tb.PipelineDeclaredResource("build-image", "image"),
				tb.PipelineParamSpec("pipeline-param-1", v1alpha1.ParamTypeString, tb.ParamSpecDefault("somethingdifferent-1")),
				tb.PipelineParamSpec("rev-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("revision")),
				tb.PipelineTask("unit-test-1", "unit-test-task",
					tb.PipelineTaskInputResource("workspace", "git-repo"),
					tb.PipelineTaskOutputResource("image-to-use", "best-image"),
					tb.PipelineTaskOutputResource("workspace", "git-repo"),
				),
			),
		),
	}

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun("test-pipeline-run-123", "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
			tb.PipelineRunSpec(pipelineName,
				tb.PipelineRunServiceAccountName("test-sa"),
				tb.PipelineRunResourceBinding("git-repo", tb.PipelineResourceBindingRef("some-repo")),
				tb.PipelineRunResourceBinding("build-image", tb.PipelineResourceBindingRef("some-image")),
				tb.PipelineRunParam("pipeline-param-1", "somethingmorefun"),
				tb.PipelineRunParam("rev-param", "revision1"),
			),
		),
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

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newPipelineClient("v1alpha1", objs...)
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], "v1alpha1"),
		cb.UnstructuredPR(prs[0], "v1alpha1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-r=git-repo=scaffold-git",
		"-p=rev-param=revision2",
		"-s=svc1",
		"--task-serviceaccount=task3=task3svc3",
		"--task-serviceaccount=task5=task3svc5",
		"-n", "ns")

	expected := "Pipelinerun started: random\n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.Get(cl, "random", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	test.AssertOutput(t, "test-generatename-pipeline-run-", pr.ObjectMeta.GenerateName)
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
						Resources: &v1beta1.PipelineTaskResources{
							Inputs: []v1beta1.PipelineTaskInputResource{
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
							Outputs: []v1beta1.PipelineTaskOutputResource{
								{
									Name:     "image-to-use",
									Resource: "best-image",
								},
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
						},
					},
				},
				Resources: []v1beta1.PipelineDeclaredResource{
					{
						Name: "git-repo",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
					{
						Name: "build-image",
						Type: v1alpha1.PipelineResourceTypeImage,
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
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
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "git-repo",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-repo",
						},
					},
					{
						Name: "build-image",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-image",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
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

	seedData, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newPipelineClient("v1beta1", objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-r=git-repo=scaffold-git",
		"-p=rev-param=revision2",
		"-s=svc1",
		"--task-serviceaccount=task3=task3svc3",
		"--task-serviceaccount=task5=task3svc5",
		"-n", "ns")

	expected := "Pipelinerun started: random\n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.Get(cl, "random", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	test.AssertOutput(t, "test-generatename-pipeline-run-", pr.ObjectMeta.GenerateName)
}

func Test_start_pipeline_last_with_prefix_name(t *testing.T) {
	pipelineName := "test-pipeline"

	ps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName, "ns",
			tb.PipelineSpec(
				tb.PipelineDeclaredResource("git-repo", "git"),
				tb.PipelineDeclaredResource("build-image", "image"),
				tb.PipelineParamSpec("pipeline-param-1", v1alpha1.ParamTypeString, tb.ParamSpecDefault("somethingdifferent-1")),
				tb.PipelineParamSpec("rev-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("revision")),
				tb.PipelineTask("unit-test-1", "unit-test-task",
					tb.PipelineTaskInputResource("workspace", "git-repo"),
					tb.PipelineTaskOutputResource("image-to-use", "best-image"),
					tb.PipelineTaskOutputResource("workspace", "git-repo"),
				),
			),
		),
	}

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun("test-pipeline-run-123", "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
			tb.PipelineRunSpec(pipelineName,
				tb.PipelineRunServiceAccountName("test-sa"),
				tb.PipelineRunResourceBinding("git-repo", tb.PipelineResourceBindingRef("some-repo")),
				tb.PipelineRunResourceBinding("build-image", tb.PipelineResourceBindingRef("some-image")),
				tb.PipelineRunParam("pipeline-param-1", "somethingmorefun"),
				tb.PipelineRunParam("rev-param", "revision1"),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newPipelineClient("v1alpha1", objs...)
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], "v1alpha1"),
		cb.UnstructuredPR(prs[0], "v1alpha1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-r=git-repo=scaffold-git",
		"-p=rev-param=revision2",
		"-s=svc1",
		"--task-serviceaccount=task3=task3svc3",
		"--task-serviceaccount=task5=task3svc5",
		"-n", "ns",
		"--prefix-name", "myprname",
	)

	expected := "Pipelinerun started: random\n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.Get(cl, "random", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	test.AssertOutput(t, "myprname-", pr.ObjectMeta.GenerateName)
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
						Resources: &v1beta1.PipelineTaskResources{
							Inputs: []v1beta1.PipelineTaskInputResource{
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
							Outputs: []v1beta1.PipelineTaskOutputResource{
								{
									Name:     "image-to-use",
									Resource: "best-image",
								},
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
						},
					},
				},
				Resources: []v1beta1.PipelineDeclaredResource{
					{
						Name: "git-repo",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
					{
						Name: "build-image",
						Type: v1alpha1.PipelineResourceTypeImage,
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
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
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "git-repo",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-repo",
						},
					},
					{
						Name: "build-image",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-image",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
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

	seedData, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newPipelineClient("v1beta1", objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"--last",
		"-r=git-repo=scaffold-git",
		"-p=rev-param=revision2",
		"-s=svc1",
		"--task-serviceaccount=task3=task3svc3",
		"--task-serviceaccount=task5=task3svc5",
		"-n", "ns",
		"--prefix-name", "myprname",
	)

	expected := "Pipelinerun started: random\n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.Get(cl, "random", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	test.AssertOutput(t, "myprname-", pr.ObjectMeta.GenerateName)
}

func Test_start_pipeline_with_prefix_name(t *testing.T) {
	pipelineName := "test-pipeline"

	ps := []*v1alpha1.Pipeline{
		tb.Pipeline(pipelineName, "ns",
			tb.PipelineSpec(
				tb.PipelineDeclaredResource("git-repo", "git"),
				tb.PipelineDeclaredResource("build-image", "image"),
				tb.PipelineParamSpec("pipeline-param-1", v1alpha1.ParamTypeString, tb.ParamSpecDefault("somethingdifferent-1")),
				tb.PipelineParamSpec("rev-param", v1alpha1.ParamTypeString, tb.ParamSpecDefault("revision")),
				tb.PipelineTask("unit-test-1", "unit-test-task",
					tb.PipelineTaskInputResource("workspace", "git-repo"),
					tb.PipelineTaskOutputResource("image-to-use", "best-image"),
					tb.PipelineTaskOutputResource("workspace", "git-repo"),
				),
			),
		),
	}

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun("test-pipeline-run-123", "ns",
			tb.PipelineRunLabel("tekton.dev/pipeline", pipelineName),
			tb.PipelineRunSpec(pipelineName,
				tb.PipelineRunServiceAccountName("test-sa"),
				tb.PipelineRunResourceBinding("git-repo", tb.PipelineResourceBindingRef("some-repo")),
				tb.PipelineRunResourceBinding("build-image", tb.PipelineResourceBindingRef("some-image")),
				tb.PipelineRunParam("pipeline-param-1", "somethingmorefun"),
				tb.PipelineRunParam("rev-param", "revision1"),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	seedData, _ := test.SeedTestData(t, pipelinetest.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newPipelineClient("v1alpha1", objs...)
	dc, err := tdc.Client(
		cb.UnstructuredP(ps[0], "v1alpha1"),
		cb.UnstructuredPR(prs[0], "v1alpha1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"-r=git-repo=scaffold-git",
		"-p=rev-param=revision2",
		"-s=svc1",
		"--task-serviceaccount=task3=task3svc3",
		"--task-serviceaccount=task5=task3svc5",
		"-n", "ns",
		"--prefix-name", "myprname",
	)

	expected := "Pipelinerun started: random\n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.Get(cl, "random", v1.GetOptions{}, "ns")
	if err != nil {
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
						Resources: &v1beta1.PipelineTaskResources{
							Inputs: []v1beta1.PipelineTaskInputResource{
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
							Outputs: []v1beta1.PipelineTaskOutputResource{
								{
									Name:     "image-to-use",
									Resource: "best-image",
								},
								{
									Name:     "workspace",
									Resource: "git-repo",
								},
							},
						},
					},
				},
				Resources: []v1beta1.PipelineDeclaredResource{
					{
						Name: "git-repo",
						Type: v1alpha1.PipelineResourceTypeGit,
					},
					{
						Name: "build-image",
						Type: v1alpha1.PipelineResourceTypeImage,
					},
				},
				Params: []v1beta1.ParamSpec{
					{
						Name: "pipeline-param-1",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingdifferent-1",
						},
					},
					{
						Name: "rev-param",
						Type: v1beta1.ParamTypeString,
						Default: &v1beta1.ArrayOrString{
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
				Resources: []v1beta1.PipelineResourceBinding{
					{
						Name: "git-repo",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-repo",
						},
					},
					{
						Name: "build-image",
						ResourceRef: &v1beta1.PipelineResourceRef{
							Name: "some-image",
						},
					},
				},
				Params: []v1beta1.Param{
					{
						Name: "pipeline-param-1",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "somethingmorefun",
						},
					},
					{
						Name: "rev-param",
						Value: v1beta1.ArrayOrString{
							Type:      v1beta1.ParamTypeString,
							StringVal: "revision1",
						},
					},
				},
			},
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
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

	seedData, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{
		Namespaces: ns,
	})
	cs := pipelinetest.Clients{
		Pipeline: seedData.Pipeline,
		Kube:     seedData.Kube,
		Resource: seedData.Resource,
	}
	cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
	objs := []runtime.Object{ps[0], prs[0]}
	_, tdc := newPipelineClient("v1beta1", objs...)
	dc, err := tdc.Client(
		cb.UnstructuredV1beta1P(ps[0], "v1beta1"),
		cb.UnstructuredV1beta1PR(prs[0], "v1beta1"),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}

	pipeline := Command(p)
	got, _ := test.ExecuteCommand(pipeline, "start", pipelineName,
		"-r=git-repo=scaffold-git",
		"-p=rev-param=revision2",
		"-s=svc1",
		"--task-serviceaccount=task3=task3svc3",
		"--task-serviceaccount=task5=task3svc5",
		"-n", "ns",
		"--prefix-name", "myprname",
	)

	expected := "Pipelinerun started: random\n\nIn order to track the pipelinerun progress run:\ntkn pipelinerun logs random -f -n ns\n"
	test.AssertOutput(t, expected, got)

	cl, _ := p.Clients()
	pr, err := pipelinerun.Get(cl, "random", v1.GetOptions{}, "ns")
	if err != nil {
		t.Errorf("Error getting pipelineruns %s", err.Error())
	}

	test.AssertOutput(t, "myprname-", pr.ObjectMeta.GenerateName)
}

func Test_mergeResource(t *testing.T) {
	pr := &v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:    "ns",
			GenerateName: "test-run",
		},
		Spec: v1beta1.PipelineRunSpec{
			PipelineRef: &v1beta1.PipelineRef{Name: "test"},
			Resources: []v1beta1.PipelineResourceBinding{
				{
					Name: "source",
					ResourceRef: &v1beta1.PipelineResourceRef{
						Name: "git",
					},
				},
			},
		},
	}

	err := mergeRes(pr, []string{"test"})
	if err == nil {
		t.Errorf("Expected error")
	}

	err = mergeRes(pr, []string{})
	if err != nil {
		t.Errorf("Did not expect error")
	}
	test.AssertOutput(t, 1, len(pr.Spec.Resources))

	err = mergeRes(pr, []string{"image=test-1"})
	if err != nil {
		t.Errorf("Did not expect error")
	}
	test.AssertOutput(t, 2, len(pr.Spec.Resources))

	err = mergeRes(pr, []string{"image=test-new", "image-2=test-2"})
	if err != nil {
		t.Errorf("Did not expect error")
	}
	test.AssertOutput(t, 3, len(pr.Spec.Resources))
}

func Test_getPipelineResourceByFormat(t *testing.T) {
	pipelineResources := []*v1alpha1.PipelineResource{
		tb.PipelineResource("scaffold-git", "ns",
			tb.PipelineResourceSpec("git",
				tb.PipelineResourceSpecParam("url", "git@github.com:tektoncd/cli.git"),
			),
		),
		tb.PipelineResource("scaffold-git-fork", "ns",
			tb.PipelineResourceSpec("git",
				tb.PipelineResourceSpecParam("url", "git@github.com:tektoncd-fork/cli.git"),
				tb.PipelineResourceSpecParam("revision", "release"),
			),
		),
		tb.PipelineResource("scaffold-image", "ns",
			tb.PipelineResourceSpec("image",
				tb.PipelineResourceSpecParam("url", "docker.io/tektoncd/cli"),
			),
		),
		tb.PipelineResource("scaffold-pull", "ns",
			tb.PipelineResourceSpec("pullRequest",
				tb.PipelineResourceSpecParam("url", "https://github.com/tektoncd/cli/pulls/9"),
			),
		),
		tb.PipelineResource("scaffold-cluster", "ns",
			tb.PipelineResourceSpec("cluster",
				tb.PipelineResourceSpecParam("url", "https://opemshift.com"),
				tb.PipelineResourceSpecParam("user", "tektoncd-developer"),
			),
		),
		tb.PipelineResource("scaffold-storage", "ns",
			tb.PipelineResourceSpec("storage",
				tb.PipelineResourceSpecParam("location", "/home/tektoncd"),
			),
		),
		tb.PipelineResource("scaffold-cloud", "ns",
			tb.PipelineResourceSpec("cloudEvent",
				tb.PipelineResourceSpecParam("targetURI", "http://sink:8080"),
			),
		),
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns",
			},
		},
	}

	cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineResources: pipelineResources, Namespaces: ns})
	res, _ := getPipelineResources(cs.Resource, "ns")
	resFormat := getPipelineResourcesByFormat(res.Items)

	output := getOptionsByType(resFormat, "git")
	expected := []string{"scaffold-git (git@github.com:tektoncd/cli.git)", "scaffold-git-fork (git@github.com:tektoncd-fork/cli.git#release)"}
	if !reflect.DeepEqual(output, expected) {
		t.Errorf("output git = %v, want %v", output, expected)
	}

	output = getOptionsByType(resFormat, "image")
	expected = []string{"scaffold-image (docker.io/tektoncd/cli)"}
	if !reflect.DeepEqual(output, expected) {
		t.Errorf("output image = %v, want %v", output, expected)
	}

	output = getOptionsByType(resFormat, "pullRequest")
	expected = []string{"scaffold-pull (https://github.com/tektoncd/cli/pulls/9)"}
	if !reflect.DeepEqual(output, expected) {
		t.Errorf("output pullRequest = %v, want %v", output, expected)
	}

	output = getOptionsByType(resFormat, "cluster")
	expected = []string{"scaffold-cluster (https://opemshift.com#tektoncd-developer)"}
	if !reflect.DeepEqual(output, expected) {
		t.Errorf("output cluster = %v, want %v", output, expected)
	}

	output = getOptionsByType(resFormat, "storage")
	expected = []string{"scaffold-storage (/home/tektoncd)"}
	if !reflect.DeepEqual(output, expected) {
		t.Errorf("output storage = %v, want %v", output, expected)
	}

	output = getOptionsByType(resFormat, "cloudEvent")
	expected = []string{"scaffold-cloud (http://sink:8080)"}
	if !reflect.DeepEqual(output, expected) {
		t.Errorf("output storage = %v, want %v", output, expected)
	}

	output = getOptionsByType(resFormat, "file")
	expected = []string{}
	if !reflect.DeepEqual(output, expected) {
		t.Errorf("output error = %v, want %v", output, expected)
	}
}

func Test_parseRes(t *testing.T) {
	type args struct {
		res []string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]v1beta1.PipelineResourceBinding
		wantErr bool
	}{{
		name: "Test_parseRes No Err",
		args: args{
			res: []string{"source=git", "image=docker2"},
		},
		want: map[string]v1beta1.PipelineResourceBinding{"source": {
			Name: "source",
			ResourceRef: &v1beta1.PipelineResourceRef{
				Name: "git",
			},
		}, "image": {
			Name: "image",
			ResourceRef: &v1beta1.PipelineResourceRef{
				Name: "docker2",
			},
		}},
		wantErr: false,
	}, {
		name: "Test_parseRes Err",
		args: args{
			res: []string{"value1", "value2"},
		},
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRes(tt.args.res)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseRes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseTaskSvc(t *testing.T) {
	type args struct {
		p []string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]v1beta1.PipelineRunSpecServiceAccountName
		wantErr bool
	}{{
		name: "Test_parseParam No Err",
		args: args{
			p: []string{"key1=value1", "key2=value2"},
		},
		want: map[string]v1beta1.PipelineRunSpecServiceAccountName{
			"key1": {TaskName: "key1", ServiceAccountName: "value1"},
			"key2": {TaskName: "key2", ServiceAccountName: "value2"},
		},
		wantErr: false,
	}, {
		name: "Test_parseParam Err",
		args: args{
			p: []string{"value1", "value2"},
		},
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTaskSvc(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSvc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSvc() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_lastPipelineRun(t *testing.T) {
	clock := clockwork.NewFakeClock()

	pr1Started := clock.Now().Add(10 * time.Second)
	pr2Started := clock.Now().Add(-2 * time.Hour)
	pr3Started := clock.Now().Add(-450 * time.Hour)

	prs := []*v1alpha1.PipelineRun{
		tb.PipelineRun("pr-2", "namespace",
			tb.PipelineRunLabel("tekton.dev/pipeline", "test"),
			cb.PipelineRunCreationTimestamp(pr2Started),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonRunning,
				}),
			),
		),
		tb.PipelineRun("pr-3", "namespace",
			tb.PipelineRunLabel("tekton.dev/pipeline", "test"),
			cb.PipelineRunCreationTimestamp(pr3Started),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionFalse,
					Reason: resources.ReasonFailed,
				}),
			),
		),
		tb.PipelineRun("pr-1", "namespace",
			tb.PipelineRunLabel("tekton.dev/pipeline", "test"),
			cb.PipelineRunCreationTimestamp(pr1Started),
			tb.PipelineRunStatus(
				tb.PipelineRunStatusCondition(apis.Condition{
					Status: corev1.ConditionTrue,
					Reason: resources.ReasonSucceeded,
				}),
			),
		),
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

					cs, _ := test.SeedTestData(t, pipelinetest.Data{PipelineRuns: prs, Namespaces: ns})
					cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
					tdc := testDynamic.Options{}
					dc, err := tdc.Client(
						cb.UnstructuredPR(prs[0], versionA1),
						cb.UnstructuredPR(prs[1], versionA1),
						cb.UnstructuredPR(prs[2], versionA1),
					)
					if err != nil {
						t.Errorf("unable to create dynamic client: %v", err)
					}
					p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Clock: clock, Resource: cs.Resource}
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
					cs, _ := test.SeedTestData(t, pipelinetest.Data{Namespaces: ns})
					cs.Pipeline.Resources = cb.APIResourceList("v1alpha1", []string{"pipeline", "pipelinerun"})
					tdc := testDynamic.Options{}
					dc, err := tdc.Client()
					if err != nil {
						t.Errorf("unable to create dynamic client: %v", err)
					}
					p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}
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

func Test_lastPipelineRun_V1beta1(t *testing.T) {
	clock := clockwork.NewFakeClock()

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
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonRunning,
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
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionFalse,
							Reason: resources.ReasonFailed,
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
				Status: duckv1beta1.Status{
					Conditions: duckv1beta1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: resources.ReasonSucceeded,
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

					cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{PipelineRuns: prs, Namespaces: ns})
					cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
					tdc := testDynamic.Options{}
					dc, err := tdc.Client(
						cb.UnstructuredV1beta1PR(prs[0], versionB1),
						cb.UnstructuredV1beta1PR(prs[1], versionB1),
						cb.UnstructuredV1beta1PR(prs[2], versionB1),
					)
					if err != nil {
						t.Errorf("unable to create dynamic client: %v", err)
					}
					p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Clock: clock, Resource: cs.Resource}
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
					cs, _ := test.SeedV1beta1TestData(t, pipelinev1beta1test.Data{Namespaces: ns})
					cs.Pipeline.Resources = cb.APIResourceList("v1beta1", []string{"pipeline", "pipelinerun"})
					tdc := testDynamic.Options{}
					dc, err := tdc.Client()
					if err != nil {
						t.Errorf("unable to create dynamic client: %v", err)
					}
					p := &test.Params{Tekton: cs.Pipeline, Kube: cs.Kube, Dynamic: dc, Resource: cs.Resource}
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
