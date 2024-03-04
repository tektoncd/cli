package bundle

import (
	"strings"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	YAMLSeparator = "\n---\n"
)

func TestDecodeFromRaw(t *testing.T) {
	testcases := []struct {
		name             string
		objects          []runtime.Object
		expectedElements []runtime.Object
		expectedErr      string
	}{
		{
			name: "some-tekton-objects",
			objects: []runtime.Object{
				&v1beta1.Pipeline{
					TypeMeta: metav1.TypeMeta{APIVersion: "tekton.dev/v1beta1", Kind: "Pipeline"},
					Spec: v1beta1.PipelineSpec{
						Tasks: v1beta1.PipelineTaskList{v1beta1.PipelineTask{Name: "foo"}},
					},
				},
				&v1beta1.Task{
					TypeMeta: metav1.TypeMeta{APIVersion: "tekton.dev/v1beta1", Kind: "Task"},
					Spec: v1beta1.TaskSpec{
						Params: []v1beta1.ParamSpec{{Name: "foo"}},
					},
				},
			},
			expectedElements: []runtime.Object{
				&v1beta1.Pipeline{
					TypeMeta: metav1.TypeMeta{APIVersion: "tekton.dev/v1beta1", Kind: "Pipeline"},
					Spec: v1beta1.PipelineSpec{
						Tasks: v1beta1.PipelineTaskList{v1beta1.PipelineTask{Name: "foo"}},
					},
				},
				&v1beta1.Task{
					TypeMeta: metav1.TypeMeta{APIVersion: "tekton.dev/v1beta1", Kind: "Task"},
					Spec: v1beta1.TaskSpec{
						Params: []v1beta1.ParamSpec{{Name: "foo"}},
					},
				},
			},
		},
		{
			name: "does-not-decode-k8s-object",
			objects: []runtime.Object{&corev1.Pod{
				TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Pod"},
				Spec: corev1.PodSpec{
					PriorityClassName: "foo",
				},
			}},
			expectedErr: "failed to parse string as a Tekton object",
		}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			var rawObjects []string
			for _, obj := range tc.objects {
				raw, err := yaml.Marshal(obj)
				if err != nil {
					t.Error(err)
				}
				rawObjects = append(rawObjects, string(raw))
			}

			err := decodeObjects(strings.Join(rawObjects, YAMLSeparator),
				func(_ *schema.GroupVersionKind, element runtime.Object, _ []byte) error {
					test.Contains(t, tc.expectedElements, element)
					return nil
				})
			if err != nil && tc.expectedErr == "" {
				t.Error(err)
			}
			if tc.expectedErr != "" && !strings.Contains(err.Error(), tc.expectedErr) {
				t.Errorf("error %s did not contain %s", err, tc.expectedErr)
			}
		})
	}
}
