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

package installer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/cli/pkg/cmd/hub/hub"
	"github.com/tektoncd/cli/pkg/cmd/hub/test"
	cb "github.com/tektoncd/cli/pkg/cmd/hub/test/builder"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const res = `---
apiVersion: tekton.dev/v1beta1
kind: Task
metadata:
  name: gvr
  labels:
    app.kubernetes.io/version: '0.3'
  annotations:
    tekton.dev/pipelines.minVersion: '0.13.1'
    tekton.dev/tags: cli
    tekton.dev/displayName: 'gvr-bar'
spec:
  description: >-
    v0.3 Task to run gvr

`

func TestToUnstructuredAndAddLabel(t *testing.T) {
	testCases := []struct {
		name            string
		data            string
		hubType         string
		org             string
		catalog         string
		wantSupportTier string
		wantCatalog     string
		wantOrg         string
	}{
		{
			name:        "Install From Tekton Hub",
			data:        res,
			hubType:     hub.TektonHubType,
			org:         "",
			catalog:     "tekton",
			wantCatalog: "tekton",
		},
		{
			name:            "Install Verified Catalog From Artifact Hub",
			data:            res,
			hubType:         hub.ArtifactHubType,
			org:             verifiedCatOrg,
			catalog:         "golang-build",
			wantSupportTier: verifiedSupportTier,
			wantCatalog:     "golang-build",
			wantOrg:         "tektoncd",
		},
		{
			name:            "Install Community Catalog From Artifact Hub",
			data:            res,
			hubType:         hub.ArtifactHubType,
			org:             "tekton-legacy",
			catalog:         "tekton-catalog-tasks",
			wantSupportTier: communitySupportTier,
			wantCatalog:     "tekton-catalog-tasks",
			wantOrg:         "tekton-legacy",
		},
	}

	for _, tc := range testCases {
		obj, err := toUnstructured([]byte(res))
		assert.NoError(t, err)
		assert.Equal(t, "gvr", obj.GetName())

		if err := addCatalogLabel(obj, tc.hubType, tc.org, tc.catalog); err != nil {
			t.Errorf("%s", err.Error())
		}

		if tc.hubType == hub.ArtifactHubType {
			assert.Equal(t, tc.wantSupportTier, obj.GetLabels()[artifactHubSupportTierLabel])
			assert.Equal(t, tc.wantCatalog, obj.GetLabels()[artifactHubCatalogLabel])
			assert.Equal(t, tc.wantOrg, obj.GetLabels()[artifactHubOrgLabel])
			assert.Equal(t, "", obj.GetLabels()[tektonHubCatalogLabel])
		} else {
			assert.Equal(t, "tekton", obj.GetLabels()[tektonHubCatalogLabel])
		}
	}
}

func TestListInstalled(t *testing.T) {
	existingTask := &v1beta1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gvr",
			Namespace: "hub",
			Labels: map[string]string{
				"hub.tekton.dev/catalog":    "tekton",
				"app.kubernetes.io/version": "0.1",
			}},
	}

	version := "v1beta1"
	dynamic := test.DynamicClient(cb.UnstructuredV1beta1T(existingTask, version))

	cs, _ := test.SeedV1beta1TestData(t, test.Data{Tasks: []*v1beta1.Task{existingTask}})
	cs.Pipeline.Resources = cb.APIResourceList(version, []string{"task"})

	clientSet := test.FakeClientSet(cs.Pipeline, dynamic, "hub")

	installer := New(clientSet)
	list, _ := installer.ListInstalled("task", "hub")

	assert.Equal(t, len(list), 1)
}
