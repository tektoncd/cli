// Copyright © 2019 The Tekton Authors.
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

package image

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/file"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	cliopts "k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes/scheme"
)

// parsedTektonResource represents the contents of a parsed file that contains a known Tekton resource like a task.
type parsedTektonResource struct {
	Name     string
	Kind     *schema.GroupVersionKind
	Contents string
}

type pushOptions struct {
	image  name.Reference
	files  []string
	stream *cli.Stream
}

func pushCommand(p cli.Params) *cobra.Command {
	opts := &pushOptions{files: []string{}}
	eg := `Push an OCI image composed of Tekton resources. Each file must contain a single Tekton resource:

    tkn image push [IMAGE REFERENCE] -f task.yaml -f pipeline.yaml
`

	c := &cobra.Command{
		Use:          "push",
		Short:        "Build and push an OCI image",
		Example:      eg,
		SilenceUsage: true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("no image specified")
			}

			var err error
			opts.image, err = name.ParseReference(args[0])
			return err
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.stream = &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			return opts.run(p)
		},
	}
	f := cliopts.NewPrintFlags("push")
	f.AddFlags(c)
	c.Flags().StringSliceVarP(&opts.files, "file", "f", []string{}, "local or remote filename to add to the image")
	return c
}

func (po *pushOptions) run(p cli.Params) error {
	if err := registerSchemes(); err != nil {
		return err
	}

	resources, err := po.parseTektonResources(p)
	if err != nil {
		return err
	}
	if len(resources) == 0 {
		return errors.New("no tekton objects found in input")
	}

	img, err := po.pushImage(resources)
	if err != nil {
		return err
	}

	d, err := img.Digest()
	if err != nil {
		return err
	}

	fmt.Fprintf(po.stream.Out, "Pushed %s", po.image.Context().Digest(d.String()))
	return nil
}

// pushImage will add each resource as a layer on a new image and push it to the remote repository with the provided
// name using the user's default repository credentials.
func (po *pushOptions) pushImage(resources []parsedTektonResource) (v1.Image, error) {
	img := empty.Image
	for _, r := range resources {
		fmt.Fprintf(po.stream.Out, "[%s]: adding %s:%s\n", po.image.Name(), r.Kind, r.Name)
		l, err := tarball.LayerFromReader(strings.NewReader(r.Contents))
		if err != nil {
			return img, fmt.Errorf("Error creating layer for resource %s/%s: %w", r.Kind, r.Name, err)
		}
		img, err = mutate.Append(img, mutate.Addendum{
			Layer: l,
			Annotations: map[string]string{
				"org.opencontainers.image.title": getLayerName(r.Kind.Kind, r.Name),
			},
		})
		if err != nil {
			return img, fmt.Errorf("Error appending resource %q: %w", r.Name, err)
		}
	}

	return img, remote.Write(po.image, img, remote.WithAuthFromKeychain(authn.DefaultKeychain))
}

// parseTektonResources will read all of the provided files and attempt to unmarshal the contents as Tekton objects. If
// something is unmarshable, it will be skipped.
func (po *pushOptions) parseTektonResources(p cli.Params) ([]parsedTektonResource, error) {
	// Read each file and attempt to marshal it to a runtime object.
	parsedResources := make([]parsedTektonResource, 0, len(po.files))

	for _, path := range po.files {
		content, err := file.LoadFileContent(p, path, file.IsYamlFile(), fmt.Errorf("invalid file format for %s: .yaml or .yml file extension and format required", path))
		if err != nil {
			return nil, err
		}

		// Attempt to unmarshal the content and ensure the Group + Version is known to Tekton.
		obj, kind, err := scheme.Codecs.UniversalDeserializer().Decode(content, nil, nil)
		if err != nil {
			// Failed to decode which indicates this object is likely not a Tekton object. Ignore instead of failing.
			fmt.Fprintf(po.stream.Out, "Skipping %s due to parsing error: %s\n", path, err.Error())
			continue
		}

		// Convert the structured data into yaml to get a "clean" copy of the resource (will actually 'omitempty').
		rawContents, err := yaml.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("could not marshal %+v to yaml: %w", obj, err)
		}

		parsedResources = append(parsedResources, parsedTektonResource{
			Name:     getResourceName(obj, kind.Kind),
			Kind:     kind,
			Contents: string(rawContents),
		})
	}

	return parsedResources, nil
}

// getResourceName will reflexively read out the ObjectMeta.Name field from the Tekton resource since all known Tekton
// CRDs use the K8s ObjectMeta field.
func getResourceName(object runtime.Object, kind string) string {
	return reflect.Indirect(reflect.ValueOf(object)).FieldByName("ObjectMeta").FieldByName("Name").String()
}

func getLayerName(kind, name string) string {
	resourceType := strings.ToLower(kind)
	if resourceType == "clustertask" {
		// Override this type because our "pulling" logic treats clustertasks as namespaced tasks.
		resourceType = "task"
	}
	return fmt.Sprintf("%s/%s", resourceType, name)
}

func registerSchemes() error {
	// Because we are using the K8s deserializer, we need to add Tekton's types to it across versions.
	schemeBuilder := runtime.NewSchemeBuilder(v1alpha1.AddToScheme)
	if err := schemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		return err
	}
	schemeBuilder = runtime.NewSchemeBuilder(v1beta1.AddToScheme)
	if err := schemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		return err
	}
	return nil
}
