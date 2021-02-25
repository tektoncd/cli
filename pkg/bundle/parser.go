package bundle

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// decodedElementHandler is a function type that processes a single decoded resource from #decodeObjects. Errors are
// returned as is. Each invocation is given the GVR of the element, the parsed struct, and the raw JSON bytes.
type decodedElementHandler func(gvr *schema.GroupVersionKind, element runtime.Object, raw []byte) error

func decodeObjects(contents string, handler decodedElementHandler) error {
	yamlDecoder := yaml.NewYAMLOrJSONDecoder(strings.NewReader(contents), 4096)
	tektonDecoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDeserializer()

	// Scratch object to decode raw manifest into.
	spec := runtime.RawExtension{}
	for {
		// Note that the code below mostly mirrors
		// https://github.com/kubernetes/cli-runtime/blob/b4586cbefd3668543b8b2b56845419e39ad1792f/pkg/resource/visitor.go#L572
		if err := yamlDecoder.Decode(&spec); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("found a spec that isn't YAML or JSON parseable: %q", contents)
		}

		spec.Raw = bytes.TrimSpace(spec.Raw)
		if len(spec.Raw) == 0 || bytes.Equal(spec.Raw, []byte("null")) {
			continue
		}

		specInJSON := string(spec.Raw)
		obj, gvr, err := tektonDecoder.Decode(spec.Raw, nil, nil)
		if err != nil {
			return fmt.Errorf("failed to parse string as a Tekton object: %q", specInJSON)
		}

		if gvr == nil {
			return fmt.Errorf("failed to parse raw tekton object with no kind or apiVersion: %s", specInJSON)
		}

		if err = handler(gvr, obj, spec.Raw); err != nil {
			return err
		}
	}
	return nil
}
