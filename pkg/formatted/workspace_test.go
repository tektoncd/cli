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

package formatted

import (
	"testing"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestWorkspace(t *testing.T) {
	workspaceSpec := []v1.WorkspaceBinding{
		{
			Name:     "emptydir-default",
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
		{
			Name: "emptydir-memory",
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumMemory,
			},
		},
		{
			Name: "configmap",
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "bar"},
			},
		},
		{
			Name: "secret",
			Secret: &corev1.SecretVolumeSource{
				SecretName: "foobar",
			},
		},
		{
			Name: "csi",
			CSI: &corev1.CSIVolumeSource{
				Driver: "secrets-store.csi.k8s.io",
			},
		},
	}
	defaultEmptyWorkspaceStr := Workspace(workspaceSpec[0]) // EmptyDir workspace with default storage medium
	assert.Equal(t, defaultEmptyWorkspaceStr, "EmptyDir (emptyDir=)")

	memoryEmptyWorkspaceStr := Workspace(workspaceSpec[1]) // EmptyDir workspace with Memory storage medium
	assert.Equal(t, memoryEmptyWorkspaceStr, "EmptyDir (emptyDir=Memory)")

	configMapWorkspaceStr := Workspace(workspaceSpec[2]) // ConfigMap workspace
	assert.Equal(t, configMapWorkspaceStr, "ConfigMap (config=bar)")

	secretWorkspaceStr := Workspace(workspaceSpec[3]) // Secret Workspace
	assert.Equal(t, secretWorkspaceStr, "Secret (secret=foobar)")

	csiWorkspaceStr := Workspace(workspaceSpec[4]) // CSI Workspace
	assert.Equal(t, csiWorkspaceStr, "CSI (Driver=secrets-store.csi.k8s.io)")
}
