package watch

import (
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

func Watch(gr schema.GroupVersionResource, clients *cli.Clients, ns string, op metav1.ListOptions) (watch.Interface, error) {
	gvr, err := actions.GetGroupVersionResource(gr, clients.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	watch, err := clients.Dynamic.Resource(*gvr).Namespace(ns).Watch(op)
	if err != nil {
		return nil, err
	}

	return watch, nil
}
