package options

import (
	"fmt"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	versionedResource "github.com/tektoncd/pipeline/pkg/client/resource/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type InteractiveOpts struct {
	Stream          *cli.Stream
	CliParams       cli.Params
	InputResources  []string
	OutputResources []string
	Params          []string
	Workspaces      []string
	AskOpts         survey.AskOpt
	Ns              string
}

type pipelineResources struct {
	git         []string
	image       []string
	cluster     []string
	storage     []string
	pullRequest []string
	cloudEvent  []string
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
	if restype == "cluster" {
		return resources.cluster
	}
	if restype == "storage" {
		return resources.storage
	}
	if restype == "cloudEvent" {
		return resources.cloudEvent
	}
	return []string{}
}

func (intOpts *InteractiveOpts) allPipelineResources(client versionedResource.Interface) (*v1alpha1.PipelineResourceList, error) {
	pres, err := client.TektonV1alpha1().PipelineResources(intOpts.CliParams.Namespace()).List(metav1.ListOptions{})
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
		case "cluster":
			for _, param := range res.Spec.Params {
				if param.Name == "url" {
					output = param.Value + output
				}
				if param.Name == "user" {
					output = output + "#" + param.Value
				}
			}
			ret.cluster = append(ret.cluster, fmt.Sprintf("%s (%s)", res.Name, output))
		case "cloudEvent":
			for _, param := range res.Spec.Params {
				if param.Name == "targetURI" {
					output = param.Value + output
				}
			}
			ret.cloudEvent = append(ret.cloudEvent, fmt.Sprintf("%s (%s)", res.Name, output))
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

func (intOpts *InteractiveOpts) TaskParams(task *v1beta1.Task) error {
	for _, param := range task.Spec.Params {
		var ans, ques, defaultValue string
		ques = fmt.Sprintf("Value for param `%s` of type `%s`?", param.Name, param.Type)
		input := &survey.Input{}
		if param.Default != nil {
			if param.Type == "string" {
				defaultValue = param.Default.StringVal
			} else {
				defaultValue = strings.Join(param.Default.ArrayVal, ",")
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
	return nil
}

func (intOpts *InteractiveOpts) TaskWorkspaces(task *v1beta1.Task) error {
	for _, ws := range task.Spec.Workspaces {
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
					Message: " Type of the Workspace :",
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
			kind, err := askParam("Type of EmtpyDir :", intOpts.AskOpts, "")
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
		it, err := askParam("Item Value :", askOpts, " ")
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
