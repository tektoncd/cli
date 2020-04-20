package taskrun

import (
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func Watch(c *cli.Clients, opts metav1.ListOptions, ns string) (watch.Interface, error) {
	watch, err := actions.Watch(trGroupResource, c, ns, opts)
	if err != nil {
		return nil, err
	}

	return watch, nil
}
