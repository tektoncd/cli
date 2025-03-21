package options

import (
	"context"
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/tektoncd/cli/pkg/actions"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/task"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var taskrunGroupResource = schema.GroupVersionResource{Group: "tekton.dev", Resource: "taskruns"}

type InteractiveOpts struct {
	Stream                *cli.Stream
	CliParams             cli.Params
	InputResources        []string
	OutputResources       []string
	Params                []string
	Workspaces            []string
	AskOpts               survey.AskOpt
	Ns                    string
	SkipOptionalWorkspace bool
}

type TaskRunOpts struct {
	CliParams  cli.Params
	Last       bool
	UseTaskRun string
	PrefixName string
}

func (taskRunOpts *TaskRunOpts) UseTaskRunFrom(tr *v1beta1.TaskRun, cs *cli.Clients, tname string, taskKind string) error {
	var (
		trUsed *v1beta1.TaskRun
		err    error
	)
	if taskRunOpts.Last {
		name, err := task.LastRunName(cs, tname, taskRunOpts.CliParams.Namespace(), taskKind)
		if err != nil {
			return err
		}

		trUsed, err = getTaskRunV1beta1(taskrunGroupResource, cs, name, taskRunOpts.CliParams.Namespace())
		if err != nil {
			return err
		}

	} else if taskRunOpts.UseTaskRun != "" {
		trUsed, err = getTaskRunV1beta1(taskrunGroupResource, cs, taskRunOpts.UseTaskRun, taskRunOpts.CliParams.Namespace())
		if err != nil {
			return err
		}
	}

	if trUsed.Spec.TaskRef.Kind != v1beta1.TaskKind(taskKind) {
		return fmt.Errorf("%s doesn't belong to %s of kind %s", trUsed.ObjectMeta.Name, tname, taskKind)
	}

	if len(trUsed.ObjectMeta.GenerateName) > 0 && taskRunOpts.PrefixName == "" {
		tr.ObjectMeta.GenerateName = trUsed.ObjectMeta.GenerateName
	} else if taskRunOpts.PrefixName == "" {
		tr.ObjectMeta.GenerateName = trUsed.ObjectMeta.Name + "-"
	}
	// Copy over spec from last or previous TaskRun to use same values for this TaskRun
	tr.Spec = trUsed.Spec
	// Reapply blank status in case TaskRun used was cancelled
	tr.Spec.Status = ""
	return nil
}

func (intOpts *InteractiveOpts) TaskParams(task *v1beta1.Task, skipParams map[string]string, useParamDefaults bool) error {
	for _, param := range task.Spec.Params {
		if param.Default == nil && useParamDefaults || !useParamDefaults {
			if _, toSkip := skipParams[param.Name]; toSkip {
				continue
			}
			var ans, ques, defaultValue string
			ques = fmt.Sprintf("Value for param `%s` of type `%s`?", param.Name, param.Type)
			input := &survey.Input{}
			if param.Default != nil {
				if param.Type == "string" {
					defaultValue = param.Default.StringVal
				}
				if param.Type == "array" {
					defaultValue = strings.Join(param.Default.ArrayVal, ",")
				}
				if param.Type == "object" {
					defaultValue = fmt.Sprintf("%+v", param.Default.ObjectVal)
				}
				ques += fmt.Sprintf(" (Default is `%s`)", defaultValue)
				input.Default = defaultValue
			}
			input.Message = ques

			var qs = []*survey.Question{
				{
					Name:   "pipeline param",
					Prompt: input,
				},
			}

			if err := survey.Ask(qs, &ans, intOpts.AskOpts); err != nil {
				return err
			}

			intOpts.Params = append(intOpts.Params, param.Name+"="+ans)
		}

	}
	return nil
}

func (intOpts *InteractiveOpts) TaskWorkspaces(task *v1beta1.Task) error {
	for _, ws := range task.Spec.Workspaces {
		if ws.Optional && intOpts.SkipOptionalWorkspace {
			continue
		}
		if ws.Optional {
			isOptional, err := askParam(fmt.Sprintf("Do you want to give specifications for the optional workspace `%s`: (y/N)", ws.Name), intOpts.AskOpts)
			if err != nil {
				return err
			}
			if strings.ToLower(isOptional) == "n" {
				continue
			}
		}
		fmt.Fprintf(intOpts.Stream.Out, "Please give specifications for the workspace: %s \n", ws.Name)
		name, err := askParam("Name for the workspace :", intOpts.AskOpts)
		if err != nil {
			return err
		}
		workspace := "name=" + name
		subPath, err := askParam("Value of the Sub Path :", intOpts.AskOpts, " ")
		if err != nil {
			return err
		}
		if subPath != " " {
			workspace = workspace + ",subPath=" + subPath
		}

		var kind string
		var qs = []*survey.Question{
			{
				Name: "workspace param",
				Prompt: &survey.Select{
					Message: "Type of the Workspace :",
					Options: []string{"config", "emptyDir", "secret", "pvc"},
					Default: "emptyDir",
				},
			},
		}
		if err := survey.Ask(qs, &kind, intOpts.AskOpts); err != nil {
			return err
		}
		switch kind {
		case "pvc":
			claimName, err := askParam("Value of Claim Name :", intOpts.AskOpts)
			if err != nil {
				return err
			}
			workspace = workspace + ",claimName=" + claimName
		case "emptyDir":
			kind, err := askParam("Type of EmptyDir :", intOpts.AskOpts, "")
			if err != nil {
				return err
			}
			workspace = workspace + ",emptyDir=" + kind
		case "config":
			config, err := askParam("Name of the configmap :", intOpts.AskOpts)
			if err != nil {
				return err
			}
			workspace = workspace + ",config=" + config
			items, err := getItems(intOpts.AskOpts)
			if err != nil {
				return err
			}
			workspace += items
		case "secret":
			secret, err := askParam("Name of the secret :", intOpts.AskOpts)
			if err != nil {
				return err
			}
			workspace = workspace + ",secret=" + secret
			items, err := getItems(intOpts.AskOpts)
			if err != nil {
				return err
			}
			workspace += items
		}
		intOpts.Workspaces = append(intOpts.Workspaces, workspace)

	}
	return nil
}

func getItems(askOpts survey.AskOpt) (string, error) {
	var items string
	for {
		it, err := askParam("Item Value:", askOpts, " ")
		if err != nil {
			return "", err
		}
		if it != " " {
			items = items + ",item=" + it
		} else {
			return items, nil
		}
	}
}

func askParam(ques string, askOpts survey.AskOpt, def ...string) (string, error) {
	var ans string
	input := &survey.Input{
		Message: ques,
	}
	if len(def) != 0 {
		input.Default = def[0]
	}

	var qs = []*survey.Question{
		{
			Name:   "workspace param",
			Prompt: input,
		},
	}
	if err := survey.Ask(qs, &ans, askOpts); err != nil {
		return "", err
	}

	return ans, nil
}

func getTaskRunV1beta1(gr schema.GroupVersionResource, c *cli.Clients, trName, ns string) (*v1beta1.TaskRun, error) {
	var taskrun v1beta1.TaskRun
	gvr, err := actions.GetGroupVersionResource(gr, c.Tekton.Discovery())
	if err != nil {
		return nil, err
	}

	if gvr.Version == "v1beta1" {
		err := actions.GetV1(gr, c, trName, ns, metav1.GetOptions{}, &taskrun)
		if err != nil {
			return nil, err
		}
		return &taskrun, nil
	}

	var taskrunV1 v1.TaskRun
	err = actions.GetV1(gr, c, trName, ns, metav1.GetOptions{}, &taskrunV1)
	if err != nil {
		return nil, err
	}
	err = taskrun.ConvertFrom(context.Background(), &taskrunV1)
	if err != nil {
		return nil, err
	}
	return &taskrun, nil
}
