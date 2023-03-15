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
	"fmt"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
)

func Workspace(ws v1.WorkspaceBinding) string {
	if ws.VolumeClaimTemplate != nil {
		return "VolumeClaimTemplate"
	}
	if ws.PersistentVolumeClaim != nil {
		claimName := ws.PersistentVolumeClaim.ClaimName
		return fmt.Sprintf("PersistentVolumeClaim (claimName=%s)", claimName)
	}
	if ws.EmptyDir != nil {
		dirType := getWorkspaceEmptyDir(ws.EmptyDir)
		return fmt.Sprintf("EmptyDir (emptyDir=%s)", dirType)
	}
	if ws.ConfigMap != nil {
		cm := getWorkspaceConfig(ws.ConfigMap)
		return fmt.Sprintf("ConfigMap (%s)", cm)
	}
	if ws.Secret != nil {
		secret := getWorkspaceSecret(ws.Secret)
		return fmt.Sprintf("Secret (%s)", secret)
	}
	if ws.CSI != nil {
		return fmt.Sprintf("CSI (Driver=%s)", ws.CSI.Driver)
	}
	return ""
}

func getWorkspaceEmptyDir(ed *corev1.EmptyDirVolumeSource) string {
	sM := ed.Medium
	var dirType string
	if sM == corev1.StorageMediumDefault {
		dirType = ""
	}
	if sM == corev1.StorageMediumMemory {
		dirType = "Memory"
	}
	if sM == corev1.StorageMediumHugePages {
		dirType = "HugePages"
	}
	return dirType
}

func getWorkspaceConfig(cm *corev1.ConfigMapVolumeSource) string {
	cmName := cm.LocalObjectReference.Name
	cmItems := cm.Items
	str := fmt.Sprintf("config=%s", cmName)
	if len(cmItems) != 0 {
		str = fmt.Sprintf("%s%s", str, getItems(cmItems))
	}
	return str
}

func getWorkspaceSecret(secret *corev1.SecretVolumeSource) string {
	secretName := secret.SecretName
	secretItems := secret.Items
	str := fmt.Sprintf("secret=%s", secretName)
	if len(secretItems) != 0 {
		str = fmt.Sprintf("%s%s", str, getItems(secretItems))
	}
	return str
}

func getItems(items []corev1.KeyToPath) string {
	str := ""
	for i := range items {
		kp := items[i]
		key := kp.Key
		value := kp.Path
		str = fmt.Sprintf("%s,item=%s=%s", str, key, value)
	}
	return str
}
