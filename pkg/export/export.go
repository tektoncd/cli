package export

import (
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func PipelineToYaml(p *v1beta1.Pipeline) (string, error) {
	p.ObjectMeta.ManagedFields = nil
	p.ObjectMeta.ResourceVersion = ""
	p.ObjectMeta.UID = ""
	p.ObjectMeta.Generation = 0
	p.ObjectMeta.SelfLink = ""
	p.ObjectMeta.Namespace = ""
	p.ObjectMeta.CreationTimestamp = metav1.Time{}

	data, err := yaml.Marshal(p)
	return string(data), err
}
