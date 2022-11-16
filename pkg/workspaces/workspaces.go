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
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/tektoncd/cli/pkg/file"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

var (
	nameParam               = "name"
	claimNameParam          = "claimName"
	subPathParam            = "subPath"
	emptyDirParam           = "emptyDir"
	configParam             = "config"
	secretParam             = "secret"
	configItemParam         = "item"
	volumeClaimTemplateFile = "volumeClaimTemplateFile"
	csiFile                 = "csiFile"
)

const invalidWorkspace = "invalid input format for workspace : "

var errNotFoundParam = errors.New("param not found")

// Merge merges workspacebinding already in pipelineruns with given options
func Merge(ws []v1beta1.WorkspaceBinding, optWS []string, httpClient http.Client) ([]v1beta1.WorkspaceBinding,
	error) {
	workspaces, err := parseWorkspace(optWS, httpClient)
	if err != nil {
		return nil, err
	}

	if len(workspaces) == 0 {
		return ws, nil
	}

	for i := range ws {
		if v, ok := workspaces[ws[i].Name]; ok {
			ws[i] = v
			delete(workspaces, v.Name)
		}
	}

	for _, v := range workspaces {
		ws = append(ws, v)
	}

	return ws, nil
}

func parseWorkspace(w []string, httpClient http.Client) (map[string]v1beta1.WorkspaceBinding, error) {
	ws := map[string]v1beta1.WorkspaceBinding{}
	for _, v := range w {

		r := strings.Split(v, ",")
		name, err := getPar(r, nameParam)
		if err != nil {
			return nil, errors.New("Name not found for workspace")
		}

		wB := v1beta1.WorkspaceBinding{
			Name: name,
		}
		nWB := 0
		subPath, err := getPar(r, subPathParam)
		if err == nil {
			wB.SubPath = subPath
		} else if err != errNotFoundParam {
			return nil, err
		}

		if vctFile, err := getPar(r, volumeClaimTemplateFile); err == nil {
			err = setWorkspaceVCTemplate(r, &wB, vctFile, httpClient)
			if err != nil {
				return nil, err
			}
			ws[name] = wB
			continue
		}

		if csiFile, err := getPar(r, csiFile); err == nil {
			err = setWorkspaceCSITemplate(r, &wB, csiFile, httpClient)
			if err != nil {
				return nil, err
			}
			ws[name] = wB
			continue
		}

		err = setWorkspaceConfig(r, &wB)
		if err == nil {
			ws[name] = wB
			nWB++
		} else if err != errNotFoundParam {
			return nil, err
		}

		err = setWorkspaceSecret(r, &wB)
		if err == nil {
			ws[name] = wB
			nWB++
		} else if err != errNotFoundParam {
			return nil, err
		}

		if err = setWorkspaceEmptyDir(r, &wB); err == nil {
			ws[name] = wB
			nWB++
		}

		if err = setWorkspacePVC(r, &wB); err == nil {
			ws[name] = wB
			nWB++
		}

		if nWB != 1 {
			return nil, errors.New(invalidWorkspace + v)
		}

	}

	return ws, nil
}

func setWorkspaceSecret(r []string, wB *v1beta1.WorkspaceBinding) error {
	secret, err := getPar(r, secretParam)
	if err != nil {
		return err
	}
	items, err := getItems(r)
	if err != nil && err != errNotFoundParam {
		return err
	}
	wB.Secret = &corev1.SecretVolumeSource{
		SecretName: secret,
		Items:      items,
	}
	return nil
}

func setWorkspaceConfig(r []string, wB *v1beta1.WorkspaceBinding) error {
	config, err := getPar(r, configParam)
	if err != nil {
		return err
	}
	items, err := getItems(r)
	if err != nil && err != errNotFoundParam {
		return err
	}

	wB.ConfigMap = &corev1.ConfigMapVolumeSource{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: config},
		Items: items,
	}
	return nil
}

func getItems(r []string) ([]corev1.KeyToPath, error) {
	var kp []corev1.KeyToPath
	for i := range r {
		if !strings.Contains(r[i], configItemParam) {
			continue
		}
		s := strings.SplitN(r[i], "=", 2)
		if s[0] != configItemParam {
			continue
		}
		if len(s) != 2 {
			return nil, errors.New("invalid item")
		}
		key, path, err := getKeyValue(s[1])
		if err != nil {
			return nil, err
		}
		kp = append(kp, corev1.KeyToPath{
			Key:  key,
			Path: path,
		})
	}
	return kp, nil
}

func setWorkspacePVC(r []string, wB *v1beta1.WorkspaceBinding) error {
	claimName, err := getPar(r, claimNameParam)
	if err != nil {
		return err
	}
	wB.PersistentVolumeClaim = &corev1.PersistentVolumeClaimVolumeSource{
		ClaimName: claimName,
	}
	return nil
}

func getKeyValue(s string) (string, string, error) {
	r := strings.SplitN(s, "=", 2)
	if len(r) != 2 {
		return "", "", errors.New("invalid key value")
	}
	return r[0], r[1], nil
}

func setWorkspaceEmptyDir(r []string, wB *v1beta1.WorkspaceBinding) error {
	emptyDir, err := getPar(r, emptyDirParam)
	if err != nil {
		return err
	}

	var sM corev1.StorageMedium
	switch emptyDir {
	case "":
		sM = corev1.StorageMediumDefault
	case "Memory":
		sM = corev1.StorageMediumMemory
	case "HugePages":
		sM = corev1.StorageMediumHugePages
	default:
		return errors.New(invalidWorkspace + emptyDirParam)
	}

	wB.EmptyDir = &corev1.EmptyDirVolumeSource{
		Medium: sM,
	}
	return nil
}

func setWorkspaceVCTemplate(r []string, wB *v1beta1.WorkspaceBinding, vctFile string, httpClient http.Client) error {
	pvc, err := parseVolumeClaimTemplate(vctFile, httpClient)
	if err != nil {
		return err
	}

	wB.VolumeClaimTemplate = pvc
	return nil
}

func setWorkspaceCSITemplate(r []string, wB *v1beta1.WorkspaceBinding, vctFile string, httpClient http.Client) error {
	csi, err := parseCSITemplate(vctFile, httpClient)
	if err != nil {
		return err
	}

	wB.CSI = csi
	return nil
}

func parseVolumeClaimTemplate(filePath string, httpClient http.Client) (*corev1.PersistentVolumeClaim, error) {
	b, err := file.LoadFileContent(httpClient, filePath, file.IsYamlFile(), fmt.Errorf("invalid file format for %s: .yaml or .yml file extension and format required", filePath))
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	err = yaml.UnmarshalStrict(b, &m)
	if err != nil {
		return nil, err
	}

	pvc := corev1.PersistentVolumeClaim{}
	if err := yaml.UnmarshalStrict(b, &pvc); err != nil {
		return nil, err
	}
	return &pvc, nil
}

func parseCSITemplate(filePath string, httpClient http.Client) (*corev1.CSIVolumeSource, error) {
	b, err := file.LoadFileContent(httpClient, filePath, file.IsYamlFile(), fmt.Errorf("invalid file format for %s: .yaml or .yml file extension and format required", filePath))
	if err != nil {
		return nil, err
	}
	m := map[string]interface{}{}
	err = yaml.UnmarshalStrict(b, &m)
	if err != nil {
		return nil, err
	}

	csi := corev1.CSIVolumeSource{}
	if err := yaml.UnmarshalStrict(b, &csi); err != nil {
		return nil, err
	}
	return &csi, nil
}

func getPar(r []string, par string) (string, error) {
	var p string
	for i := range r {
		if !strings.Contains(r[i], par) {
			continue
		}
		s := strings.SplitN(r[i], "=", 2)
		if s[0] != par {
			continue
		}
		if len(s) != 2 {
			return p, errors.New(invalidWorkspace + r[i])
		}
		p = s[1]
		return p, nil
	}
	return p, errNotFoundParam
}
