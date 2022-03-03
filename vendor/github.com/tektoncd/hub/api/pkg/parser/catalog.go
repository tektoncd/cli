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

package parser

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tektoncd/hub/api/pkg/git"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	decoder "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
	"knative.dev/pkg/apis"
)

const (
	VersionLabel                  = "app.kubernetes.io/version"
	DisplayNameAnnotation         = "tekton.dev/displayName"
	MinPipelinesVersionAnnotation = "tekton.dev/pipelines.minVersion"
	TagsAnnotation                = "tekton.dev/tags"
	CategoryAnnotation            = "tekton.dev/categories"
	PlatformsAnnotation           = "tekton.dev/platforms"
	DefaultPlatform               = "linux/amd64"
	DeprecatedAnnotation          = "tekton.dev/deprecated"
)

type (
	Resource struct {
		Name       string
		Kind       string
		Tags       []string
		Platforms  []string
		Versions   []VersionInfo
		Categories []string
	}

	VersionInfo struct {
		Version             string
		DisplayName         string
		Deprecated          bool
		MinPipelinesVersion string
		Description         string
		Path                string
		ModifiedAt          time.Time
		Platforms           []string
	}
)

type CatalogParser struct {
	logger      *zap.SugaredLogger
	repo        git.Repo
	contextPath string
}

func (c *CatalogParser) Parse() ([]Resource, Result) {
	resources := []Resource{}
	result := Result{}

	for _, k := range kinds {
		res, r := c.findResourcesByKind(k)
		resources = append(resources, res...)
		result.Combine(r)
	}
	if len(resources) == 0 {
		result.AddError(fmt.Errorf("no resources found in repo"))
	}
	return resources, result
}

func ignoreNotExists(err error) error {
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (c CatalogParser) findResourcesByKind(kind string) ([]Resource, Result) {
	log := c.logger.With("kind", kind)
	log.Info("looking for resources")

	found := []Resource{}
	result := Result{}

	// search for resources under catalog/<contextPath>/<kind>
	kindPath := filepath.Join(c.repo.Path(), c.contextPath, strings.ToLower(kind))

	resourceDirs, err := ioutil.ReadDir(kindPath)
	if err != nil && ignoreNotExists(err) != nil {
		log.Warnf("failed to find %s: %s", kind, err)
		// NOTE: returns empty task list; upto caller to check for error
		result.AddError(err)
		return []Resource{}, result
	}

	for _, res := range resourceDirs {
		if !res.IsDir() {
			log.Infof("ignoring %q not a directory for %s", res.Name(), kind)
			continue
		}

		res, r := c.parseResource(kind, kindPath, res)
		result.Combine(r)
		if r.Errors != nil {
			log.Warn(r.Error())
			continue
		}

		// NOTE: res can be nil if no files exists for resource (invalid resource)
		if res != nil {
			found = append(found, *res)
		}
	}

	log.Infof("found %d resources of kind %s", len(found), kind)
	return found, result

}

func dirCount(path string) int {
	count := 0
	dirs, _ := ioutil.ReadDir(path)
	for _, d := range dirs {
		if d.IsDir() {
			count++
		}
	}
	return count
}

func (c CatalogParser) parseResource(kind, kindPath string, f os.FileInfo) (*Resource, Result) {
	log := c.logger.With("kind", kind)
	name := f.Name()
	log.Info("checking path", kindPath, " resource: ", name)

	res := Resource{
		Name:     name,
		Kind:     kind,
		Versions: []VersionInfo{},
	}
	result := Result{}

	// search for catalog/<contextPath>/<kind>/<name>/<version>/<name.yaml>
	pattern := filepath.Join(kindPath, name, "*", name+".yaml")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Warn(err, "failed to glob %s", pattern)
		result.AddError(err)
		return nil, result
	}
	if len(matches) == 0 {
		log.Warn("failed to find resources in path %s", kindPath)
		result.Critical("failed to find any resource matching %s", pattern)
		return nil, result
	}

	if exp, got := dirCount(filepath.Join(kindPath, name)), len(matches); got != exp {
		log.Warn("expected to find %d versions for %s/%s but found only", exp, kind, name, got)
		result.Critical("expected to find %d versions but found only %d for %s ", exp, got, pattern)
	}

	for _, m := range matches {
		log.Info(" found file: ", m)

		r := c.appendVersion(&res, m)
		result.Combine(r)
		if r.Errors != nil {
			log.Warn(result.Error())
			continue
		}
	}

	log.Infof("found %d versions of resource %s/%s", len(res.Versions), kind, name)
	if len(res.Versions) == 0 {
		return nil, result
	}

	return &res, result
}

// appendVersion reads the contents of the file at filePath and use the K8s deserializer
// to marshal the contents into a Tekton struct.
// This fails if the resource is unparseable or is not a Tekton resource.
func (c CatalogParser) appendVersion(res *Resource, filePath string) Result {

	result := Result{}

	kind := res.Kind
	log := c.logger.With("kind", kind)

	relPath, err := c.repo.RelPath(filePath)
	if err != nil {
		result.AddError(err)
		log.Error("unexpected error %q when computing path in repo of %q", filePath)
		return result
	}

	modified, err := c.repo.ModifiedTime(filePath)
	if err != nil {
		issue := fmt.Errorf("internal error computing modified time for %q: %s", relPath, err)
		result.AddError(err)
		log.Warn(issue)
		return result
	}

	f, err := os.Open(filePath)
	if err != nil {
		result.AddError(err)
		return result
	}

	tkn, err := decodeResource(f, kind)
	if err != nil {
		log.Warn(err)
		result.AddError(err)
		return result
	}

	log = log.With("kind", kind, "name", tkn.Name)

	u := tkn.Unstructured

	// mandatory checks
	labels := u.GetLabels()
	version, ok := labels[VersionLabel]
	if !ok {
		issue := fmt.Sprintf("Resource %s - %s is missing mandatory version label", tkn.GVK, tkn.Name)
		result.Critical(issue)
		log.With("action", "error").Warn(issue)
		return result
	}

	annotations := u.GetAnnotations()
	MinPipelinesVersion, ok := annotations[MinPipelinesVersionAnnotation]
	if !ok {
		issue := fmt.Sprintf("Resource %s - %s is missing mandatory minimum pipeline version annotation", tkn.GVK, tkn.Name)
		log.With("action", "error").Warn(issue)
		result.Critical(issue)
		return result
	}

	// optionals checks
	displayName, ok := annotations[DisplayNameAnnotation]
	if !ok {
		issue := fmt.Sprintf("Resource %s - %s has no display name", tkn.GVK, tkn.Name)
		log.With("action", "ignore").Info(issue)
		result.Info(issue)
	}

	// optional check
	isDeprecated, ok := annotations[DeprecatedAnnotation]
	if ok {
		issue := fmt.Sprintf("Resource %s - %s version of %s has been deprecated", tkn.GVK, version, tkn.Name)
		log.With("action", "ignore").Info(issue)
	}

	var deprecated bool
	if isDeprecated == "true" {
		deprecated = true
	}

	description, found, err := unstructured.NestedString(u.Object, "spec", "description")
	if !found || err != nil {
		issue := fmt.Sprintf("Resource %s - %s has no description", tkn.GVK, tkn.Name)
		log.With("action", "ignore").Info(issue)
		return result
	}

	tags := annotations[TagsAnnotation]
	tagList := strings.FieldsFunc(tags, func(c rune) bool { return c == ',' || c == ' ' })
	res.Tags = append(res.Tags, tagList...)

	categories := annotations[CategoryAnnotation]
	categoryList := strings.FieldsFunc(categories, func(c rune) bool { return c == ',' })
	for c := range categoryList {
		categoryList[c] = strings.TrimSpace(categoryList[c])
	}
	res.Categories = append(res.Categories, categoryList...)

	versionPlatforms := []string{}
	platforms := annotations[PlatformsAnnotation]
	platformList := strings.FieldsFunc(platforms, func(c rune) bool { return c == ',' || c == ' ' })
	versionPlatforms = append(versionPlatforms, platformList...)
	// add default platform value in case if platform is not specified
	if len(versionPlatforms) == 0 {
		versionPlatforms = append(versionPlatforms, DefaultPlatform)
	}

	res.Versions = append(res.Versions,
		VersionInfo{
			Version:             version,
			DisplayName:         displayName,
			Deprecated:          deprecated,
			MinPipelinesVersion: MinPipelinesVersion,
			Description:         description,
			Path:                relPath,
			ModifiedAt:          modified,
			Platforms:           versionPlatforms,
		},
	)

	return result
}

// decode consumes the given reader and parses its contents as YAML.
func decodeResource(reader io.Reader, kind string) (*TektonResource, error) {

	// create a duplicate for  UniversalDeserializer and NewYAMLToJSONDecoder
	// to read from readers
	var dup bytes.Buffer
	r := io.TeeReader(reader, &dup)
	contents, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	object, gvk, err := scheme.Codecs.UniversalDeserializer().Decode(contents, nil, nil)
	if err != nil || !isTektonKind(gvk) {
		return nil, fmt.Errorf("parse error: invalid resource %+v:\n%s", err, contents)
	}

	decoder := decoder.NewYAMLToJSONDecoder(&dup)

	var res *unstructured.Unstructured
	for {
		res = &unstructured.Unstructured{}
		if err := decoder.Decode(res); err != nil {
			return nil, fmt.Errorf("failed to decode: %w", err)
		}

		if len(res.Object) == 0 {
			continue
		}

		// break at the first object that has a kind and Catalog TEP expects
		// that the files have only one resource
		if res.GetKind() != "" {
			break
		}
	}

	if k := res.GetKind(); k != kind {
		return nil, fmt.Errorf("expected kind to be %s but got %s", kind, k)
	}

	if _, err := convertToTyped(res); err != nil {
		return nil, err
	}

	return &TektonResource{
		Name:         res.GetName(),
		Kind:         gvk.Kind,
		GVK:          res.GroupVersionKind(),
		Unstructured: res,
		Object:       object,
	}, nil
}

func convertToTyped(u *unstructured.Unstructured) (interface{}, error) {
	t, err := typeForKind(u.GetKind())
	if err != nil {
		return nil, err
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, t); err != nil {
		return nil, err
	}

	t.SetDefaults(context.Background())
	if fe := t.Validate(context.Background()); !isEmptyFieldError(fe) {
		return nil, fe
	}

	return t, nil
}

func isEmptyFieldError(fe *apis.FieldError) bool {
	if fe == nil {
		return true
	}
	return fe.Message == "" && fe.Details == "" && len(fe.Paths) == 0
}

type tektonKind interface {
	apis.Validatable
	apis.Defaultable
}

func typeForKind(kind string) (tektonKind, error) {
	switch kind {
	case "Task":
		return &v1beta1.Task{}, nil
	case "ClusterTask":
		return &v1beta1.ClusterTask{}, nil
	case "Pipeline":
		return &v1beta1.Pipeline{}, nil
	}

	return nil, fmt.Errorf("unknown kind %s", kind)
}

func isTektonKind(gvk *schema.GroupVersionKind) bool {
	id := gvk.GroupVersion().Identifier()
	return id == v1beta1.SchemeGroupVersion.Identifier()
}
