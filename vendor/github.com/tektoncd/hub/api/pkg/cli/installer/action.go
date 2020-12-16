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
	"bytes"
	"errors"
	"fmt"

	"github.com/hashicorp/go-version"
	kErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	decoder "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	catalogLabel = "hub.tekton.dev/catalog"
	versionLabel = "app.kubernetes.io/version"
)

// Errors
var (
	ErrAlreadyExist             = errors.New("resource already exists")
	ErrNotFound                 = errors.New("resource not found")
	ErrVersionAndCatalogMissing = errors.New("version and catalog label missing")
	ErrVersionMissing           = errors.New("version label missing")
	ErrCatalogMissing           = errors.New("catalog label missing")
	ErrSameVersion              = errors.New("resource already exists with the requested verion")
	ErrLowerVersion             = errors.New("cannot upgrade resource as requested version is lower than existing")
	ErrHigherVersion            = errors.New("cannot downgrade resource as requested version is higher than existing")
)

type action string

const (
	update    action = "update"
	upgrade   action = "upgrade"
	downgrade action = " downgrade"
)

// Install a resource
func (i *Installer) Install(data []byte, catalog, namespace string) (*unstructured.Unstructured, error) {

	newRes, err := toUnstructured(data)
	if err != nil {
		return nil, err
	}

	// Check if resource already exists
	existingRes, err := i.get(newRes.GetName(), newRes.GetKind(), namespace, metav1.GetOptions{})
	if err != nil {
		// If error is notFoundError then create the resource
		if kErr.IsNotFound(err) {
			return i.createRes(newRes, catalog, namespace)
		}
		// otherwise return the error
		return nil, err
	}

	return existingRes, ErrAlreadyExist
}

// LookupInstalled checks if a resource is installed
func (i *Installer) LookupInstalled(name, kind, namespace string) (*unstructured.Unstructured, error) {

	var err error
	i.existingRes, err = i.get(name, kind, namespace, metav1.GetOptions{})
	if err != nil {
		if kErr.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if err := checkLabels(i.existingRes); err != nil {
		return i.existingRes, err
	}

	return i.existingRes, nil
}

// Update will updates an existing resource with the passed resource if exist
func (i *Installer) Update(data []byte, catalog, namespace string) (*unstructured.Unstructured, error) {
	return i.updateByAction(data, catalog, namespace, update)
}

// Upgrade an existing resource to a version upper version by passing it
func (i *Installer) Upgrade(data []byte, catalog, namespace string) (*unstructured.Unstructured, error) {
	return i.updateByAction(data, catalog, namespace, upgrade)
}

// Downgrade an existing resource to a version lower version by passing it
func (i *Installer) Downgrade(data []byte, catalog, namespace string) (*unstructured.Unstructured, error) {
	return i.updateByAction(data, catalog, namespace, downgrade)
}

func (i *Installer) updateByAction(data []byte, catalog, namespace string, action action) (*unstructured.Unstructured, error) {

	newRes, err := toUnstructured(data)
	if err != nil {
		return nil, err
	}

	if i.existingRes == nil {
		i.existingRes, err = i.get(newRes.GetName(), newRes.GetKind(), namespace, metav1.GetOptions{})
		if err != nil {
			if kErr.IsNotFound(err) {
				return nil, ErrNotFound
			}
			return nil, err
		}
	}

	existingVersion := i.existingRes.GetLabels()[versionLabel]
	newVersion := newRes.GetLabels()[versionLabel]

	switch action {
	case upgrade:
		if err = isUpgradable(existingVersion, newVersion); err != nil {
			return i.existingRes, err
		}
	case downgrade:
		if err = isDowngradable(existingVersion, newVersion); err != nil {
			return i.existingRes, err
		}
	}

	return i.updateRes(i.existingRes, newRes, catalog, namespace)
}

func isUpgradable(existingVersion, newVersion string) error {

	exVer, _ := version.NewVersion(existingVersion)
	newVer, _ := version.NewVersion(newVersion)

	if newVer.Equal(exVer) {
		return ErrSameVersion
	}
	if newVer.LessThan(exVer) {
		return ErrLowerVersion
	}
	return nil
}

func isDowngradable(existingVersion, newVersion string) error {

	exVer, _ := version.NewVersion(existingVersion)
	newVer, _ := version.NewVersion(newVersion)

	if newVer.Equal(exVer) {
		return ErrSameVersion
	}
	if newVer.GreaterThan(exVer) {
		return ErrHigherVersion
	}
	return nil
}

func checkLabels(res *unstructured.Unstructured) error {

	labels := res.GetLabels()
	if len(labels) == 0 {
		return ErrVersionAndCatalogMissing
	}

	_, versionOk := labels[versionLabel]
	_, catalogOk := labels[catalogLabel]

	// If both label exist then return nil
	if versionOk == catalogOk && versionOk {
		return nil
	}

	if !catalogOk {
		return ErrCatalogMissing
	}

	if !versionOk {
		return ErrVersionMissing
	}

	return nil
}

func (i *Installer) createRes(obj *unstructured.Unstructured, catalog, namespace string) (*unstructured.Unstructured, error) {

	addCatalogLabel(obj, catalog)
	res, err := i.create(obj, namespace, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (i *Installer) updateRes(existing, new *unstructured.Unstructured, catalog, namespace string) (*unstructured.Unstructured, error) {

	addCatalogLabel(new, catalog)
	// replace label, annotation and spec of old resource with new
	existing.SetLabels(new.GetLabels())
	existing.SetAnnotations(new.GetAnnotations())
	existing.Object["spec"] = new.Object["spec"]

	res, err := i.update(existing, namespace, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func toUnstructured(data []byte) (*unstructured.Unstructured, error) {

	r := bytes.NewReader(data)
	decoder := decoder.NewYAMLToJSONDecoder(r)

	res := &unstructured.Unstructured{}
	if err := decoder.Decode(res); err != nil {
		return nil, fmt.Errorf("failed to decode resource: %w", err)
	}
	return res, nil
}

func addCatalogLabel(obj *unstructured.Unstructured, catalog string) {
	labels := obj.GetLabels()
	if len(labels) == 0 {
		labels = make(map[string]string)
	}
	labels[catalogLabel] = catalog
	obj.SetLabels(labels)
}
