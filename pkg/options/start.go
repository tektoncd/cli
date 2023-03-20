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
	"github.com/tektoncd/pipeline/pkg/apis/resource/v1alpha1"
	versionedResource "github.com/tektoncd/pipeline/pkg/client/resource/clientset/versioned"
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

type pipelineResources struct {
	git         []string
	image       []string
	storage     []string
	pullRequest []string
}

func resourceByType(resources pipelineResources, restype string) []string {
	if restype == "git" {
		return resources.git
	}
	if restype == "image" {
		return resources.image
	}
	if restype == "pullRequest" {
		return resources.pullRequest
	}
	if restype == "storage" {
		return resources.storage
	}
	return []string{}
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

func (intOpts *InteractiveOpts) allPipelineResources(client versionedResource.Interface) (*v1alpha1.PipelineResourceList, error) {
	pres, err := client.TektonV1alpha1().PipelineResources(intOpts.CliParams.Namespace()).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		fmt.Fprintf(intOpts.Stream.Err, "failed to list pipelineresources from %s namespace \n", intOpts.Ns)
		return nil, err
	}
	return pres, nil
}

func (intOpts *InteractiveOpts) pipelineResourcesByFormat(client versionedResource.Interface) (pipelineResources, error) {
	ret := pipelineResources{}
	resources, err := intOpts.allPipelineResources(client)
	if err != nil {
		return ret, err
	}
	for _, res := range resources.Items {
		output := ""
		switch string(res.Spec.Type) {
		case "git":
			for _, param := range res.Spec.Params {
				if param.Name == "url" {
					output = param.Value + output
				}
				if param.Name == "revision" && param.Value != "master" {
					output = output + "#" + param.Value
				}
			}
			ret.git = append(ret.git, fmt.Sprintf("%s (%s)", res.Name, output))
		case "image":
			for _, param := range res.Spec.Params {
				if param.Name == "url" {
					output = param.Value + output
				}
			}
			ret.image = append(ret.image, fmt.Sprintf("%s (%s)", res.Name, output))
		case "pullRequest":
			for _, param := range res.Spec.Params {
				if param.Name == "url" {
					output = param.Value + output
				}
			}
			ret.pullRequest = append(ret.pullRequest, fmt.Sprintf("%s (%s)", res.Name, output))
		case "storage":
			for _, param := range res.Spec.Params {
				if param.Name == "location" {
					output = param.Value + output
				}
			}
			ret.storage = append(ret.storage, fmt.Sprintf("%s (%s)", res.Name, output))
		}
	}
	return ret, nil
}

func (intOpts *InteractiveOpts) TaskInputResources(task *v1beta1.Task, f func(v1alpha1.PipelineResourceType, survey.AskOpt, cli.Params, *cli.Stream) (*v1alpha1.PipelineResource, error)) error {
	cs, err := intOpts.CliParams.Clients()
	if err != nil {
		return err
	}

	resources, err := intOpts.pipelineResourcesByFormat(cs.Resource)
	if err != nil {
		return err
	}
	for _, res := range task.Spec.Resources.Inputs {
		options := resourceByType(resources, string(res.Type))
		// directly create resource
		if len(options) == 0 {
			fmt.Fprintf(intOpts.Stream.Out, "no pipeline resource of type \"%s\" found in namespace: %s\n", string(res.Type), intOpts.Ns)
			fmt.Fprintf(intOpts.Stream.Out, "Please create a new \"%s\" resource for pipeline resource \"%s\"\n", string(res.Type), res.Name)
			newres, err := f(res.Type, intOpts.AskOpts, intOpts.CliParams, intOpts.Stream)
			if err != nil {
				return err
			}
			if newres.Status != nil {
				fmt.Printf("resource status %s\n\n", newres.Status)
			}
			intOpts.InputResources = append(intOpts.InputResources, res.Name+"="+newres.Name)
			continue
		}

		// shows create option in the resource list
		resCreateOpt := fmt.Sprintf("create new \"%s\" resource", res.Type)
		options = append(options, resCreateOpt)
		var ans string
		var qs = []*survey.Question{
			{
				Name: "pipelineresource",
				Prompt: &survey.Select{
					Message: fmt.Sprintf("Choose the %s resource to use for %s:", res.Type, res.Name),
					Options: options,
				},
			},
		}

		if err := survey.Ask(qs, &ans, intOpts.AskOpts); err != nil {
			return err
		}

		if ans == resCreateOpt {
			newres, err := f(res.Type, intOpts.AskOpts, intOpts.CliParams, intOpts.Stream)
			if err != nil {
				return err
			}
			intOpts.InputResources = append(intOpts.InputResources, res.Name+"="+newres.Name)
			continue
		}
		name := strings.TrimSpace(strings.Split(ans, " ")[0])
		intOpts.InputResources = append(intOpts.InputResources, res.Name+"="+name)
	}
	return nil
}

func (intOpts *InteractiveOpts) TaskOutputResources(task *v1beta1.Task, f func(v1alpha1.PipelineResourceType, survey.AskOpt, cli.Params, *cli.Stream) (*v1alpha1.PipelineResource, error)) error {
	cs, err := intOpts.CliParams.Clients()
	if err != nil {
		return err
	}

	resources, err := intOpts.pipelineResourcesByFormat(cs.Resource)
	if err != nil {
		return err
	}

	for _, res := range task.Spec.Resources.Outputs {
		options := resourceByType(resources, string(res.Type))
		// directly create resource
		if len(options) == 0 {
			fmt.Fprintf(intOpts.Stream.Out, "no pipeline resource of type \"%s\" found in namespace: %s\n", string(res.Type), intOpts.Ns)
			fmt.Fprintf(intOpts.Stream.Out, "Please create a new \"%s\" resource for pipeline resource \"%s\"\n", string(res.Type), res.Name)
			newres, err := f(res.Type, intOpts.AskOpts, intOpts.CliParams, intOpts.Stream)
			if err != nil {
				return err
			}
			if newres.Status != nil {
				fmt.Printf("resource status %s\n\n", newres.Status)
			}
			intOpts.OutputResources = append(intOpts.OutputResources, res.Name+"="+newres.Name)
			continue
		}

		// shows create option in the resource list
		resCreateOpt := fmt.Sprintf("create new \"%s\" resource", res.Type)
		options = append(options, resCreateOpt)
		var ans string
		var qs = []*survey.Question{
			{
				Name: "pipelineresource",
				Prompt: &survey.Select{
					Message: fmt.Sprintf("Choose the %s resource to use for %s:", res.Type, res.Name),
					Options: options,
				},
			},
		}

		if err := survey.Ask(qs, &ans, intOpts.AskOpts); err != nil {
			return err
		}

		if ans == resCreateOpt {
			newres, err := f(res.Type, intOpts.AskOpts, intOpts.CliParams, intOpts.Stream)
			if err != nil {
				return err
			}
			intOpts.OutputResources = append(intOpts.OutputResources, res.Name+"="+newres.Name)
			continue
		}
		name := strings.TrimSpace(strings.Split(ans, " ")[0])
		intOpts.OutputResources = append(intOpts.OutputResources, res.Name+"="+name)
	}
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

func (intOpts *InteractiveOpts) ClusterTaskInputResources(clustertask *v1beta1.ClusterTask, f func(v1alpha1.PipelineResourceType, survey.AskOpt, cli.Params, *cli.Stream) (*v1alpha1.PipelineResource, error)) error {
	cs, err := intOpts.CliParams.Clients()
	if err != nil {
		return err
	}

	resources, err := intOpts.pipelineResourcesByFormat(cs.Resource)
	if err != nil {
		return err
	}
	for _, res := range clustertask.Spec.Resources.Inputs {
		options := resourceByType(resources, string(res.Type))
		// directly create resource
		if len(options) == 0 {
			fmt.Fprintf(intOpts.Stream.Out, "no PipelineResource of type \"%s\" found in namespace: %s\n", string(res.Type), intOpts.Ns)
			fmt.Fprintf(intOpts.Stream.Out, "Please create a new \"%s\" resource for PipelineResource \"%s\"\n", string(res.Type), res.Name)
			newres, err := f(res.Type, intOpts.AskOpts, intOpts.CliParams, intOpts.Stream)
			if err != nil {
				return err
			}
			if newres.Status != nil {
				fmt.Printf("resource status %s\n\n", newres.Status)
			}
			intOpts.InputResources = append(intOpts.InputResources, res.Name+"="+newres.Name)
			continue
		}

		// shows create option in the resource list
		resCreateOpt := fmt.Sprintf("create new \"%s\" resource", res.Type)
		options = append(options, resCreateOpt)
		var ans string
		var qs = []*survey.Question{
			{
				Name: "pipelineresource",
				Prompt: &survey.Select{
					Message: fmt.Sprintf("Choose the %s resource to use for %s:", res.Type, res.Name),
					Options: options,
				},
			},
		}

		if err := survey.Ask(qs, &ans, intOpts.AskOpts); err != nil {
			return err
		}

		if ans == resCreateOpt {
			newres, err := f(res.Type, intOpts.AskOpts, intOpts.CliParams, intOpts.Stream)
			if err != nil {
				return err
			}
			intOpts.InputResources = append(intOpts.InputResources, res.Name+"="+newres.Name)
			continue
		}
		name := strings.TrimSpace(strings.Split(ans, " ")[0])
		intOpts.InputResources = append(intOpts.InputResources, res.Name+"="+name)
	}
	return nil
}

func (intOpts *InteractiveOpts) ClusterTaskOutputResources(clustertask *v1beta1.ClusterTask, f func(v1alpha1.PipelineResourceType, survey.AskOpt, cli.Params, *cli.Stream) (*v1alpha1.PipelineResource, error)) error {
	cs, err := intOpts.CliParams.Clients()
	if err != nil {
		return err
	}

	resources, err := intOpts.pipelineResourcesByFormat(cs.Resource)
	if err != nil {
		return err
	}

	for _, res := range clustertask.Spec.Resources.Outputs {
		options := resourceByType(resources, string(res.Type))
		// directly create resource
		if len(options) == 0 {
			fmt.Fprintf(intOpts.Stream.Out, "no PipelineResource of type \"%s\" found in namespace: %s\n", string(res.Type), intOpts.Ns)
			fmt.Fprintf(intOpts.Stream.Out, "Please create a new \"%s\" resource for PipelineResource \"%s\"\n", string(res.Type), res.Name)
			newres, err := f(res.Type, intOpts.AskOpts, intOpts.CliParams, intOpts.Stream)
			if err != nil {
				return err
			}
			if newres.Status != nil {
				fmt.Printf("resource status %s\n\n", newres.Status)
			}
			intOpts.OutputResources = append(intOpts.OutputResources, res.Name+"="+newres.Name)
			continue
		}

		// shows create option in the resource list
		resCreateOpt := fmt.Sprintf("create new \"%s\" resource", res.Type)
		options = append(options, resCreateOpt)
		var ans string
		var qs = []*survey.Question{
			{
				Name: "pipelineresource",
				Prompt: &survey.Select{
					Message: fmt.Sprintf("Choose the %s resource to use for %s:", res.Type, res.Name),
					Options: options,
				},
			},
		}

		if err := survey.Ask(qs, &ans, intOpts.AskOpts); err != nil {
			return err
		}

		if ans == resCreateOpt {
			newres, err := f(res.Type, intOpts.AskOpts, intOpts.CliParams, intOpts.Stream)
			if err != nil {
				return err
			}
			intOpts.OutputResources = append(intOpts.OutputResources, res.Name+"="+newres.Name)
			continue
		}
		name := strings.TrimSpace(strings.Split(ans, " ")[0])
		intOpts.OutputResources = append(intOpts.OutputResources, res.Name+"="+name)
	}
	return nil
}

func (intOpts *InteractiveOpts) ClusterTaskParams(clustertask *v1beta1.ClusterTask, skipParams map[string]string, useParamDefaults bool) error {
	for _, param := range clustertask.Spec.Params {
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
					Name:   "clustertask param",
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

func (intOpts *InteractiveOpts) ClusterTaskWorkspaces(clustertask *v1beta1.ClusterTask) error {
	for _, ws := range clustertask.Spec.Workspaces {
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
		name, err := askParam("Name for the workspace:", intOpts.AskOpts)
		if err != nil {
			return err
		}
		workspace := "name=" + name
		subPath, err := askParam("Value of the Sub Path:", intOpts.AskOpts, " ")
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
					Message: "Type of the Workspace:",
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
			claimName, err := askParam("Value of Claim Name:", intOpts.AskOpts)
			if err != nil {
				return err
			}
			workspace = workspace + ",claimName=" + claimName
		case "emptyDir":
			kind, err := askParam("Type of EmptyDir:", intOpts.AskOpts, "")
			if err != nil {
				return err
			}
			workspace = workspace + ",emptyDir=" + kind
		case "config":
			config, err := askParam("Name of the configmap:", intOpts.AskOpts)
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
			secret, err := askParam("Name of the secret:", intOpts.AskOpts)
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
