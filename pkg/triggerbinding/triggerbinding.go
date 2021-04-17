package triggerbinding

import (
	"context"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetAllTriggerBindingNames(client versioned.Interface, namespace string) ([]string, error) {

	ps, err := List(client, namespace)
	if err != nil {
		return nil, err
	}

	ret := []string{}
	for _, item := range ps.Items {
		ret = append(ret, item.ObjectMeta.Name)
	}
	return ret, nil
}

func List(client versioned.Interface, namespace string) (*v1alpha1.TriggerBindingList, error) {
	tbs, err := client.TriggersV1alpha1().TriggerBindings(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// NOTE: this is required for -o json|yaml to work properly since
	// tektoncd go client fails to set these; probably a bug
	tbs.GetObjectKind().SetGroupVersionKind(
		schema.GroupVersionKind{
			Version: "triggers.tekton.dev/v1alpha1",
			Kind:    "TriggerBindingList",
		})

	return tbs, nil
}
