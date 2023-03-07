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

	"github.com/tektoncd/hub/api/pkg/cli/hub"
	tknVer "github.com/tektoncd/hub/api/pkg/cli/version"
	kErr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	decoder "k8s.io/apimachinery/pkg/util/yaml"
)

const (
	versionLabel                = "app.kubernetes.io/version"
	tektonHubCatalogLabel       = "hub.tekton.dev/catalog"
	artifactHubCatalogLabel     = "artifacthub.io/catalog"
	artifactHubOrgLabel         = "artifacthub.io/org"
	artifactHubSupportTierLabel = "artifacthub.io/support-tier"
	communitySupportTier        = "Community"
	verifiedSupportTier         = "Verified"
	verifiedCatOrg              = "tektoncd"
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
	ErrVersionIncompatible      = errors.New("requires compatible version")
	ErrWarnVersionNotFound      = errors.New("pipeline version unknown")
)

type action string

const (
	update             action = "update"
	upgrade            action = "upgrade"
	downgrade          action = " downgrade"
	ResourceMinVersion string = "tekton.dev/pipelines.minVersion"
)

func (i *Installer) TektonPipelinesVersion() {

	var err error
	i.pipelineVersion, err = tknVer.GetPipelineVersion(i.cs.Dynamic())
	if err != nil {
		i.pipelineVersion = ""
	}
}

func (i *Installer) GetPipelineVersion() string {
	return i.pipelineVersion
}

func (i *Installer) checkVersion(resPipMinVersion string) error {
	i.TektonPipelinesVersion()

	if i.GetPipelineVersion() == "" {
		return ErrWarnVersionNotFound
	}

	if i.GetPipelineVersion() < resPipMinVersion && i.GetPipelineVersion() != "devel" {
		return ErrVersionIncompatible
	}

	return nil
}

// Install a resource
func (i *Installer) Install(data []byte, hubType, org, catalog, namespace string) (*unstructured.Unstructured, []error) {

	errors := make([]error, 0)

	newRes, err := toUnstructured(data)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	newResPipMinVersion := newRes.GetAnnotations()[ResourceMinVersion]
	err = i.checkVersion("v" + newResPipMinVersion)
	if err != nil {
		if err == ErrWarnVersionNotFound {
			errors = append(errors, err)
		}

		if err == ErrVersionIncompatible {
			errors = append(errors, err)
			return nil, errors
		}
	}

	// Check if resource already exists
	existingRes, err := i.get(newRes.GetName(), newRes.GetKind(), namespace, metav1.GetOptions{})
	if err != nil {
		// If error is notFoundError then create the resource
		if kErr.IsNotFound(err) {
			resp, err := i.createRes(newRes, hubType, org, catalog, namespace)
			if err != nil {
				errors = append(errors, err)
			}
			return resp, errors
		}
		errors = append(errors, err)
		// otherwise return the error
		return nil, errors
	}

	errors = append(errors, ErrAlreadyExist)

	return existingRes, errors
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

func (i *Installer) ListInstalled(kind, namespace string) ([]unstructured.Unstructured, error) {
	i.TektonPipelinesVersion()
	listResources, err := i.list(kind, namespace, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return listResources.Items, nil
}

// Update will updates an existing resource with the passed resource if exist
func (i *Installer) Update(data []byte, catalog, namespace string) (*unstructured.Unstructured, []error) {
	return i.updateByAction(data, catalog, namespace, update)
}

// Upgrade an existing resource to a version upper version by passing it
func (i *Installer) Upgrade(data []byte, catalog, namespace string) (*unstructured.Unstructured, []error) {
	return i.updateByAction(data, catalog, namespace, upgrade)
}

// Downgrade an existing resource to a version lower version by passing it
func (i *Installer) Downgrade(data []byte, catalog, namespace string) (*unstructured.Unstructured, []error) {
	return i.updateByAction(data, catalog, namespace, downgrade)
}

func (i *Installer) updateByAction(data []byte, catalog, namespace string, action action) (*unstructured.Unstructured, []error) {

	var errors []error

	newRes, err := toUnstructured(data)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	newResPipMinVersion := newRes.GetAnnotations()[ResourceMinVersion]

	err = i.checkVersion("v" + newResPipMinVersion)
	if err != nil {
		if err == ErrWarnVersionNotFound {
			errors = append(errors, err)
		}

		if err == ErrVersionIncompatible {
			errors = append(errors, err)
			return newRes, errors
		}
	}

	if i.existingRes == nil {
		i.existingRes, err = i.get(newRes.GetName(), newRes.GetKind(), namespace, metav1.GetOptions{})
		if err != nil {
			if kErr.IsNotFound(err) {
				errors = append(errors, ErrNotFound)
				return nil, errors
			}
			errors = append(errors, err)
			return nil, errors
		}
	}

	existingVersion := i.existingRes.GetLabels()[versionLabel]
	newVersion := newRes.GetLabels()[versionLabel]

	switch action {
	case upgrade:
		if err = isUpgradable(existingVersion, newVersion); err != nil {
			errors = append(errors, err)
			return i.existingRes, errors
		}
	case downgrade:
		if err = isDowngradable(existingVersion, newVersion); err != nil {
			errors = append(errors, err)
			return i.existingRes, errors
		}
	}

	res, err := i.updateRes(i.existingRes, newRes, catalog, namespace)
	if err != nil {
		errors = append(errors, err)
	}
	return res, errors
}

func isUpgradable(existingVersion, newVersion string) error {

	if newVersion == existingVersion {
		return ErrSameVersion
	}
	if newVersion < existingVersion {
		return ErrLowerVersion
	}
	return nil
}

func isDowngradable(existingVersion, newVersion string) error {

	if newVersion == existingVersion {
		return ErrSameVersion
	}
	if newVersion > existingVersion {
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
	_, catalogOk := labels[tektonHubCatalogLabel]

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

func (i *Installer) createRes(obj *unstructured.Unstructured, hubType, org, catalog, namespace string) (*unstructured.Unstructured, error) {

	if err := addCatalogLabel(obj, hubType, org, catalog); err != nil {
		return nil, err
	}
	res, err := i.create(obj, namespace, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (i *Installer) updateRes(existing, new *unstructured.Unstructured, catalog, namespace string) (*unstructured.Unstructured, error) {

	// TODO: update addCatalogLabel() params when supporting upgrade/downgrade command for artifact type
	if err := addCatalogLabel(new, hub.TektonHubType, "", catalog); err != nil {
		return nil, err
	}

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

func addCatalogLabel(obj *unstructured.Unstructured, hubType, org, catalog string) error {
	labels := obj.GetLabels()
	if len(labels) == 0 {
		labels = make(map[string]string)
	}

	switch hubType {
	case hub.TektonHubType:
		labels[tektonHubCatalogLabel] = catalog
	case hub.ArtifactHubType:
		labels[artifactHubCatalogLabel] = catalog
		if org == verifiedCatOrg {
			labels[artifactHubSupportTierLabel] = verifiedSupportTier
		} else {
			labels[artifactHubSupportTierLabel] = communitySupportTier
		}
		labels[artifactHubOrgLabel] = org
	default:
		return fmt.Errorf("hub type: %s not supported in addCatalogLabel", hubType)
	}

	obj.SetLabels(labels)
	return nil
}
