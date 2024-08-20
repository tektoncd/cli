// Copyright © 2019-2020 The Tekton Authors.
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

package workspaces

import (
	"net/http"
	"testing"

	"github.com/tektoncd/cli/pkg/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMerge(t *testing.T) {
	ws := []v1beta1.WorkspaceBinding{
		{
			Name: "foo",
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "bar"},
			},
		},
		{
			Name: "emptydir-data",
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium: corev1.StorageMediumDefault,
			},
		},
	}

	httpClient := *http.DefaultClient

	optWS := []string{}
	outWS, err := Merge(ws, optWS, httpClient)
	if err != nil {
		t.Errorf("Not expected error: %s", err.Error())
	}
	test.AssertOutput(t, ws, outWS)

	optWS = []string{"test"}
	_, err = Merge(ws, optWS, httpClient)
	if err == nil {
		t.Errorf("Expected error")
	}
	test.AssertOutput(t, "Name not found for workspace", err.Error())

	optWS = []string{"name"}
	_, err = Merge(ws, optWS, httpClient)
	if err == nil {
		t.Errorf("Expected error")
	}
	test.AssertOutput(t, "Name not found for workspace", err.Error())

	optWS = []string{"name=test,configsecret=wrong"}
	_, err = Merge(ws, optWS, httpClient)
	if err == nil {
		t.Errorf("Expected error")
	}
	test.AssertOutput(t, invalidWorkspace+optWS[0], err.Error())

	optWS = []string{"name=emptydir-data-hp,emptyDir=s3"}
	_, err = Merge(ws, optWS, httpClient)
	if err == nil {
		t.Errorf("Expected error")
	}
	test.AssertOutput(t, invalidWorkspace+optWS[0], err.Error())

	optWS = []string{"name=recipe-store,config=sensitive-recipe-storage,item=brownies"}
	_, err = Merge(ws, optWS, httpClient)
	if err == nil {
		t.Errorf("Expected error")
	}
	test.AssertOutput(t, "invalid key value", err.Error())

	optWS = []string{"name=recipe-store,secret=secret-name,item=brownies"}
	_, err = Merge(ws, optWS, httpClient)
	if err == nil {
		t.Errorf("Expected error")
	}
	test.AssertOutput(t, "invalid key value", err.Error())

	optWS = []string{"name=recipe-store,config=sensitive-recipe-storage," +
		"secret=secret-name,item=brownies=recipe.txt"}
	_, err = Merge(ws, optWS, httpClient)
	if err == nil {
		t.Errorf("Expected error")
	}
	test.AssertOutput(t, invalidWorkspace+optWS[0], err.Error())

	optWS = []string{"name=password-vault,secret=secret-name,item=pass=/etc/passwd",
		"name=recipe-store,config=sensitive-recipe-storage,item=brownies=recipe.txt",
		"name=emptydir-data-hp,emptyDir=HugePages",
		"name=emptydir-data-mem,emptyDir=Memory",
		"name=emptydir-data,emptyDir=",
		"name=shared-data-path,claimName=pvc3,subPath=dir",
	}
	outWS, err = Merge(ws, optWS, httpClient)
	if err != nil {
		t.Errorf("Not expected error: %s", err.Error())
	}
	test.AssertOutput(t, 7, len(outWS))

	optWS = []string{"name=volumeclaimtemplatews,volumeClaimTemplateFile=./testdata/pvc.yaml"}
	_, err = Merge(ws, optWS, httpClient)
	if err != nil {
		t.Errorf("Not expected error: %s", err.Error())
	}

	optWS = []string{"name=volumeclaimtemplatews,volumeClaimTemplateFile=./testdata/pvc-typo.yaml"}
	_, err = Merge(ws, optWS, httpClient)
	if err == nil {
		t.Errorf("Expected error")
	}
	test.AssertOutput(t, "error unmarshaling JSON: while decoding JSON: json: unknown field \"storageClassNam\"", err.Error())

	optWS = []string{"name=csiws,csiFile=./testdata/csi.yaml"}
	outWS, err = Merge(ws, optWS, httpClient)
	if err != nil {
		t.Errorf("Not expected error: %s", err.Error())
	}

	optWS = []string{"name=csiws,csiFile=./testdata/csi-typo.yaml"}
	_, err = Merge(ws, optWS, httpClient)
	if err == nil {
		t.Errorf("Expected error")
	}
	test.AssertOutput(t, "error unmarshaling JSON: while decoding JSON: json: unknown field \"drive\"", err.Error())

	storageClassName := "storageclassname"
	isCsiReadOnly := true
	expectedWS := []v1beta1.WorkspaceBinding{
		{Name: "foo", ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "bar"}}},
		{Name: "emptydir-data", EmptyDir: &corev1.EmptyDirVolumeSource{}},
		{
			Name: "password-vault",
			Secret: &corev1.SecretVolumeSource{
				SecretName: "secret-name",
				Items:      []corev1.KeyToPath{{Key: "pass", Path: "/etc/passwd"}}},
		},
		{
			Name: "recipe-store",
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "sensitive-recipe-storage"},
				Items:                []corev1.KeyToPath{{Key: "brownies", Path: "recipe.txt"}}},
		},
		{
			Name:     "emptydir-data-hp",
			EmptyDir: &corev1.EmptyDirVolumeSource{Medium: corev1.StorageMediumHugePages},
		},
		{
			Name:     "emptydir-data-mem",
			EmptyDir: &corev1.EmptyDirVolumeSource{Medium: corev1.StorageMediumMemory},
		},
		{
			Name:                  "shared-data-path",
			SubPath:               "dir",
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "pvc3"},
		},
		{
			Name: "volumeclaimtemplatews",
			VolumeClaimTemplate: &corev1.PersistentVolumeClaim{
				TypeMeta: metav1.TypeMeta{
					Kind:       "PersistentVolumeClaim",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "vctname",
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					StorageClassName: &storageClassName,
					Resources: corev1.VolumeResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("50Mi"),
						},
					},
					AccessModes: []corev1.PersistentVolumeAccessMode{
						corev1.ReadWriteOnce,
					},
				},
			},
		},
		{
			Name: "csiws",
			CSI: &corev1.CSIVolumeSource{
				Driver:           "secrets-store.csi.k8s.io",
				ReadOnly:         &isCsiReadOnly,
				VolumeAttributes: map[string]string{"secretProviderClass": "vault-database"},
			},
		},
	}

	for i := range outWS {
		switch outWS[i].Name {
		case "foo":
			test.AssertOutput(t, expectedWS[0], outWS[i])
		case "emptydir-data":
			test.AssertOutput(t, expectedWS[1], outWS[i])
		case "password-vault":
			test.AssertOutput(t, expectedWS[2], outWS[i])
		case "recipe-store":
			test.AssertOutput(t, expectedWS[3], outWS[i])
		case "emptydir-data-hp":
			test.AssertOutput(t, expectedWS[4], outWS[i])
		case "emptydir-data-mem":
			test.AssertOutput(t, expectedWS[5], outWS[i])
		case "shared-data-path":
			test.AssertOutput(t, expectedWS[6], outWS[i])
		case "volumeclaimtemplatews":
			test.AssertOutput(t, expectedWS[7], outWS[i])
		case "csiws":
			test.AssertOutput(t, expectedWS[8], outWS[i])
		}
	}
}
