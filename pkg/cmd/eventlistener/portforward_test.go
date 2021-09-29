package eventlistener

import (
	"testing"
	"time"

	"github.com/jonboulle/clockwork"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/test"
	cb "github.com/tektoncd/cli/pkg/test/builder"
	testDynamic "github.com/tektoncd/cli/pkg/test/dynamic"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggertest "github.com/tektoncd/triggers/test"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestEventListenerPortForward(t *testing.T) {
	now := time.Now()

	els := []*v1beta1.EventListener{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "eventlistener-no-deploy",
			},
			Status: v1beta1.EventListenerStatus{
				Configuration: v1beta1.EventListenerConfig{
					GeneratedResourceName: "eventlistener-no-deploy",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "eventlistener-no-pods",
			},
			Status: v1beta1.EventListenerStatus{
				Configuration: v1beta1.EventListenerConfig{
					GeneratedResourceName: "eventlistener-no-pods",
				},
			},
		},
	}

	deploys := []*appsv1.Deployment{{
		ObjectMeta: metav1.ObjectMeta{
			Name: "eventlistener-no-pods",
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"some": "label"},
			},
		},
	}}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "No arguments passed",
			args: []string{"port-forward"},
			want: "accepts 1 arg(s), received 0",
		},
		{
			name: "No EventListener found",
			args: []string{"port-forward", "notFound"},
			want: "failed to get EventListener notFound: eventlisteners.triggers.tekton.dev \"notFound\" not found",
		},
		{
			name: "No EventListener deployment",
			args: []string{"port-forward", "eventlistener-no-deploy"},
			want: "failed to get Deployment eventlistener-no-deploy: deployments.apps \"eventlistener-no-deploy\" not found",
		},
		{
			name: "No EventListener pods",
			args: []string{"port-forward", "eventlistener-no-pods"},
			want: "no pods available for EventListener eventlistener-no-pods",
		},
	}

	for _, td := range tests {
		t.Run(td.name, func(t *testing.T) {
			got, err := test.ExecuteCommand(commandPortforward(t, els, deploys, now), td.args...)

			if err != nil {
				test.AssertOutput(t, td.want, err.Error())
			} else {
				test.AssertOutput(t, td.want, got)
			}
		})
	}
}

func commandPortforward(t *testing.T, els []*v1beta1.EventListener, deploys []*appsv1.Deployment, now time.Time) *cobra.Command {
	// fake clock advanced by 1 hour
	clock := clockwork.NewFakeClockAt(now)
	cs := test.SeedTestResources(t, triggertest.Resources{EventListeners: els, Deployments: deploys})
	cs.Triggers.Resources = cb.TriggersAPIResourceList("v1beta1", []string{"eventlistener"})
	tdc := testDynamic.Options{}
	var utts []runtime.Object
	for _, el := range els {
		utts = append(utts, cb.UnstructuredV1beta1EL(el, "v1beta1"))
	}
	dc, err := tdc.Client(utts...)
	if err != nil {
		t.Errorf("unable to create dynamic client: %v", err)
	}
	p := &test.Params{Tekton: cs.Pipeline, Clock: clock, Kube: cs.Kube, Triggers: cs.Triggers, Dynamic: dc}
	return Command(p)
}
