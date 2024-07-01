// Copyright © 2024 The Tekton Authors.
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

package customrun

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestDeleteSingleCustomRun(t *testing.T) {
	// Create a sample CustomRun
	crs := []*v1beta1.CustomRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "customrun-1",
				Namespace: "foo",
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "customrun-2",
				Namespace: "foo",
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	tdc := testDynamic.Options{}
	dynamicClient, err := tdc.Client(
		cb.UnstructuredV1beta1CustomRun(crs[0], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[1], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	// Create cli.Clients with the dynamic client
	clients := &cli.Clients{
		Dynamic: dynamicClient,
	}

	// Namespace and CustomRun name for the test
	namespace := "foo"
	customRunName := "customrun-1"

	// Check that the CustomRun exists before deletion
	_, err = clients.Dynamic.Resource(customrunGroupResource).Namespace(namespace).Get(context.TODO(), customRunName, metav1.GetOptions{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Call the function
	err = deleteCustomRun(clients, namespace, customRunName)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify that the CustomRun has been deleted
	_, err = clients.Dynamic.Resource(customrunGroupResource).Namespace(namespace).Get(context.TODO(), customRunName, metav1.GetOptions{})
	if err == nil {
		t.Errorf("expected error but got none")
	}

	// Attempt to delete a non-existent CustomRun
	err = deleteCustomRun(clients, namespace, "non-existent-customrun")
	if err == nil {
		t.Errorf("expected error but got none")
	}
}
func TestCustomRunDelete(t *testing.T) {
	now := time.Now()
	// Define the CustomRuns for testing
	crs := []*v1beta1.CustomRun{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "customrun-1",
				Namespace: "ns-1",
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "customrun-2",
				Namespace: "ns-1",
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		// Additional CustomRuns for ns-1 and ns-2
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "customrun-3",
				Namespace: "ns-1",
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "customrun-4",
				Namespace: "ns-1",
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "customrun-5",
				Namespace: "ns-1",
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "customrun-6",
				Namespace: "ns-2",
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "customrun-7",
				Namespace: "ns-2",
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "customrun-8",
				Namespace: "ns-2",
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "customrun-9",
				Namespace: "ns-2",
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "customrun-10",
				Namespace: "ns-2",
			},
			Spec: v1beta1.CustomRunSpec{},
			Status: v1beta1.CustomRunStatus{
				Status: duckv1.Status{
					Conditions: duckv1.Conditions{
						{
							Status: corev1.ConditionTrue,
							Reason: v1beta1.CustomRunReasonSuccessful.String(),
						},
					},
				},
			},
		},
	}

	ns := []*corev1.Namespace{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-1",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-2",
			},
		},
	}

	tdc := testDynamic.Options{}
	dynamicClient, err := tdc.Client(
		cb.UnstructuredV1beta1CustomRun(crs[0], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[1], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[2], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[3], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[4], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[5], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[6], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[7], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[8], versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(crs[9], versionv1beta1),
	)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}

	tests := []struct {
		name      string
		command   *cobra.Command
		args      []string
		wantError bool
		want      string
	}{
		{
			name:      "Delete non-existent customrun",
			command:   commandV1beta1(t, crs, now, ns, dynamicClient),
			args:      []string{"delete", "customrun-xyz", "-n", "ns-1"},
			wantError: true,
			want:      "failed to delete CustomRun customrun-xyz: customruns.tekton.dev customrun-xyz not found\n",
		},
		{
			name:      "Delete one customrun without namespace",
			command:   commandV1beta1(t, crs, now, ns, dynamicClient),
			args:      []string{"delete", "customrun-1"},
			wantError: false,
			want:      "CustomRun customrun-1 not found in namespace \n",
		},
		{
			name:      "Delete multiple customruns without namespace",
			command:   commandV1beta1(t, crs, now, ns, dynamicClient),
			args:      []string{"delete", "customrun-2", "customrun-3"},
			wantError: false,
			want:      "CustomRun customrun-2 not found in namespace \nCustomRun customrun-3 not found in namespace \n",
		},
		{
			name:      "Delete one customrun with namespace",
			command:   commandV1beta1(t, crs, now, ns, dynamicClient),
			args:      []string{"delete", "customrun-4", "-n", "ns-1"},
			wantError: false,
			want:      "CustomRun 'customrun-4' deleted successfully from namespace 'ns-1'\n",
		},
		{
			name:      "Delete multiple customruns with namespace",
			command:   commandV1beta1(t, crs, now, ns, dynamicClient),
			args:      []string{"delete", "customrun-6", "customrun-7", "-n", "ns-2"},
			wantError: false,
			want:      "CustomRun 'customrun-6' deleted successfully from namespace 'ns-2'\nCustomRun 'customrun-7' deleted successfully from namespace 'ns-2'\n",
		},
	}

	for _, td := range tests {
		t.Run(td.name, func(t *testing.T) {
			got, err := test.ExecuteCommand(td.command, td.args...)

			if err != nil && !td.wantError {
				t.Errorf("Unexpected error: %v", err)
			}

			if td.wantError {
				if err != nil {
					test.AssertOutput(t, td.want, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				test.AssertOutput(t, td.want, got)
			}
		})
	}
}
