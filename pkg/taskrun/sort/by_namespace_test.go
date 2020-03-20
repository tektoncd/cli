package taskrun

import (
	"github.com/jonboulle/clockwork"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func Test_TaskRunsByNamespace(t *testing.T) {
	tr1 := v1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "abc",
			Name:      "tr0-1",
		},
	}

	tr2 := v1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "def",
			Name:      "tr1-1",
		},
	}

	tr3 := v1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ghi",
			Name:      "tr2-1",
		},
	}

	trs := []v1alpha1.TaskRun{
		tr2,
		tr3,
		tr1,
	}

	SortByNamespace(trs)

	element1 := trs[0].Name
	if element1 != "tr0-1" {
		t.Errorf("SortTaskRunsByNamespace should be tr0-1 but returned: %s", element1)
	}

	element2 := trs[1].Name
	if element2 != "tr1-1" {
		t.Errorf("SortTaskRunsByNamespace should be tr1-1 but returned: %s", element2)
	}

	element3 := trs[2].Name
	if element3 != "tr2-1" {
		t.Errorf("SortTaskRunsByNamespace should be tr2-1 but returned: %s", element3)
	}
}

func Test_PipelineRunsByNamespaceWithStartTime(t *testing.T) {

	clock := clockwork.NewFakeClock()

	tr00Started := clock.Now().Add(10 * time.Second)
	tr01Started := clock.Now().Add(-1 * time.Hour)
	tr10Started := clock.Now().Add(10 * time.Second)
	tr11Started := clock.Now().Add(-1 * time.Hour)
	tr20Started := clock.Now().Add(10 * time.Second)
	tr21Started := clock.Now().Add(-1 * time.Hour)

	tr00 := v1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "abc",
			Name:      "tr0-0",
		},
	}

	tr00.Status.StartTime = &metav1.Time{Time: tr00Started}

	tr01 := v1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "abc",
			Name:      "tr0-1",
		},
	}

	tr01.Status.StartTime = &metav1.Time{Time: tr01Started}

	tr10 := v1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "def",
			Name:      "tr1-0",
		},
	}

	tr10.Status.StartTime = &metav1.Time{Time: tr10Started}

	tr11 := v1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "def",
			Name:      "tr1-1",
		},
	}

	tr11.Status.StartTime = &metav1.Time{Time: tr11Started}

	tr20 := v1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ghi",
			Name:      "tr2-0",
		},
	}

	tr20.Status.StartTime = &metav1.Time{Time: tr20Started}

	tr21 := v1alpha1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ghi",
			Name:      "tr2-1",
		},
	}

	tr21.Status.StartTime = &metav1.Time{Time: tr21Started}

	trs := []v1alpha1.TaskRun{
		tr11,
		tr21,
		tr01,
		tr10,
		tr20,
		tr00,
	}

	SortByNamespace(trs)

	element1 := trs[0].Name
	if element1 != "tr0-0" {
		t.Errorf("SortTaskRunsByNamespaceWithStartTime should be tr0-0 but returned: %s", element1)
	}

	element2 := trs[1].Name
	if element2 != "tr0-1" {
		t.Errorf("SortTaskRunsByNamespaceWithStartTime should be tr0-1 but returned: %s", element2)
	}

	element3 := trs[2].Name
	if element3 != "tr1-0" {
		t.Errorf("SortTaskRunsByNamespaceWithStartTime should be tr1-0 but returned: %s", element3)
	}

	element4 := trs[3].Name
	if element4 != "tr1-1" {
		t.Errorf("SortTaskRunsByNamespaceWithStartTime should be tr1-1 but returned: %s", element4)
	}

	element5 := trs[4].Name
	if element5 != "tr2-0" {
		t.Errorf("SortTaskRunsByNamespaceWithStartTime should be tr2-0 but returned: %s", element5)
	}

	element6 := trs[5].Name
	if element6 != "tr2-1" {
		t.Errorf("SortTaskRunsByNamespaceWithStartTime should be tr2-1 but returned: %s", element6)
	}
}
