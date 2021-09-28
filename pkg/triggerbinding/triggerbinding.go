package triggerbinding

import (
	"fmt"
	"os"

	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var triggerbindingGroupResource = schema.GroupVersionResource{Group: "triggers.tekton.dev", Resource: "triggerbindings"}

func GetAllTriggerBindingNames(client *cli.Clients, namespace string) ([]string, error) {
	ps, err := List(client, metav1.ListOptions{}, namespace)
	if err != nil {
		return nil, err
	}

	ret := []string{}
	for _, item := range ps.Items {
		ret = append(ret, item.ObjectMeta.Name)
	}
	return ret, nil
}

func List(c *cli.Clients, opts metav1.ListOptions, ns string) (*v1beta1.TriggerBindingList, error) {
	unstructuredTB, err := actions.List(triggerbindingGroupResource, c.Dynamic, c.Triggers.Discovery(), ns, opts)
	if err != nil {
		return nil, err
	}

	var triggerbindings *v1beta1.TriggerBindingList
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredTB.UnstructuredContent(), &triggerbindings); err != nil {
		return nil, err
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list triggerbindings from %s namespace \n", ns)
		return nil, err
	}

	return triggerbindings, nil
}

func Get(c *cli.Clients, tbname string, opts metav1.GetOptions, ns string) (*v1beta1.TriggerBinding, error) {
	unstructuredTB, err := actions.Get(triggerbindingGroupResource, c.Dynamic, c.Triggers.Discovery(), tbname, ns, opts)
	if err != nil {
		return nil, err
	}

	var tb *v1beta1.TriggerBinding
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredTB.UnstructuredContent(), &tb); err != nil {
		fmt.Fprintf(os.Stderr, "failed to get triggerbinding from %s namespace \n", ns)
		return nil, err
	}
	return tb, nil
}
