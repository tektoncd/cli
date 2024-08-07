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

	"github.com/tektoncd/cli/pkg/cli"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

func TestDeleteMultipleCustomRuns(t *testing.T) {
	// Create sample CustomRuns
	cr1 := &v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "customrun-1",
			Namespace: "foo",
		},
	}
	cr2 := &v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "customrun-2",
			Namespace: "foo",
		},
	}

	tdc := testDynamic.Options{}
	dynamicClient, err := tdc.Client(
		cb.UnstructuredV1beta1CustomRun(cr1, versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(cr2, versionv1beta1),
	)
	if err != nil {
		t.Fatalf("unable to create dynamic client: %v", err)
	}

	// Create cli.Clients with the dynamic client
	clients := &cli.Clients{
		Dynamic: dynamicClient,
	}

	// Namespace and CustomRun names for the test
	namespace := "foo"
	customRunNames := []string{"customrun-1", "customrun-2"}

	// Call the function to delete multiple CustomRuns
	s := &cli.Stream{}
	err = deleteCustomRuns(s, clients, namespace, customRunNames)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that the CustomRuns have been deleted
	for _, customRunName := range customRunNames {
		_, err = clients.Dynamic.Resource(customrunGroupResource).Namespace(namespace).Get(context.TODO(), customRunName, metav1.GetOptions{})
		if !apierrors.IsNotFound(err) {
			t.Errorf("expected not found error for %s but got: %v", customRunName, err)
		}
	}
}
func TestNamespaceSpecificDeletion(t *testing.T) {
	// Create CustomRuns in different namespaces
	cr1 := &v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "customrun-1",
			Namespace: "foo",
		},
	}
	cr2 := &v1beta1.CustomRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "customrun-2",
			Namespace: "bar",
		},
	}

	tdc := testDynamic.Options{}
	dynamicClient, err := tdc.Client(
		cb.UnstructuredV1beta1CustomRun(cr1, versionv1beta1),
		cb.UnstructuredV1beta1CustomRun(cr2, versionv1beta1),
	)
	if err != nil {
		t.Fatalf("unable to create dynamic client: %v", err)
	}

	// Create cli.Clients with the dynamic client
	clients := &cli.Clients{
		Dynamic: dynamicClient,
	}

	// Namespace and CustomRun names for the test
	namespace := "foo"
	customRunNames := []string{"customrun-1"}

	// Call the function to delete CustomRuns in a specific namespace
	s := &cli.Stream{}
	err = deleteCustomRuns(s, clients, namespace, customRunNames)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that the CustomRun in the "foo" namespace has been deleted
	_, err = clients.Dynamic.Resource(customrunGroupResource).Namespace(namespace).Get(context.TODO(), "customrun-1", metav1.GetOptions{})
	if !apierrors.IsNotFound(err) {
		t.Errorf("expected not found error but got: %v", err)
	}

	// Verify that the CustomRun in the "bar" namespace still exists
	_, err = clients.Dynamic.Resource(customrunGroupResource).Namespace("bar").Get(context.TODO(), "customrun-2", metav1.GetOptions{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
