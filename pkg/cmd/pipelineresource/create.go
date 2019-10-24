// Copyright © 2019 The Tekton Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pipelineresource

import (
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/AlecAivazis/survey/v2"
	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	validateinput "github.com/tektoncd/cli/pkg/helper/validate"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type resource struct {
	params           cli.Params
	stream           *cli.Stream
	askOpts          survey.AskOpt
	pipelineResource v1alpha1.PipelineResource
}

func createCommand(p cli.Params) *cobra.Command {
	res := &resource{params: p,
		askOpts: func(opt *survey.AskOptions) error {
			opt.Stdio = terminal.Stdio{
				In:  os.Stdin,
				Out: os.Stdout,
				Err: os.Stderr,
			}
			return nil
		},
	}

	eg := `
  # creates new resource as per the given input
    tkn resource create -n namespace

   `
	c := &cobra.Command{
		Use:                   "create",
		DisableFlagsInUseLine: true,
		Short:                 "Creates pipeline resource",
		Example:               eg,
		SilenceUsage:          true,
		Annotations: map[string]string{
			"commandType": "main",
		},

		RunE: func(cmd *cobra.Command, args []string) error {

			res.stream = &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			if err := validateinput.NamespaceExists(p); err != nil {
				return err
			}

			return res.create()
		},
	}

	return c
}

func (res *resource) create() error {
	res.pipelineResource.Namespace = res.params.Namespace()

	// ask for the object meta data name, namespace
	if err := res.askMeta(); err != nil {
		return err
	}

	// below all the question mostly belongs to pipelineresource spec
	// ask for the resource type
	if err := res.askType(); err != nil {
		return err
	}

	resourceTypeParams := map[v1alpha1.PipelineResourceType]func() error{
		v1alpha1.PipelineResourceTypeGit:         res.askGitParams,
		v1alpha1.PipelineResourceTypeStorage:     res.askStorageParams,
		v1alpha1.PipelineResourceTypeImage:       res.askImageParams,
		v1alpha1.PipelineResourceTypeCluster:     res.askClusterParams,
		v1alpha1.PipelineResourceTypePullRequest: res.askPullRequestParams,
		v1alpha1.PipelineResourceTypeCloudEvent:  res.askCloudEventParams,
	}
	if res.pipelineResource.Spec.Type != "" {
		if err := resourceTypeParams[res.pipelineResource.Spec.Type](); err != nil {
			return err
		}
	}

	cls, err := res.params.Clients()
	if err != nil {
		return err
	}

	newRes, err := cls.Tekton.TektonV1alpha1().PipelineResources(res.params.Namespace()).Create(&res.pipelineResource)
	if err != nil {
		return err
	}

	fmt.Fprintf(res.stream.Out, "New %s resource \"%s\" has been created\n", newRes.Spec.Type, newRes.Name)
	return nil
}

func (res *resource) askMeta() error {
	var answer string
	var qs = []*survey.Question{{
		Name: "resource name",
		Prompt: &survey.Input{
			Message: "Enter a name for a pipeline resource :",
		},
		Validate: survey.Required,
	}}

	err := survey.Ask(qs, &answer, res.askOpts)
	if err != nil {
		return Error(err)
	}
	if err := validate(answer, res.params); err != nil {
		return err
	}

	res.pipelineResource.Name = answer

	return nil
}

func (res *resource) askType() error {
	var answer string
	var qs = []*survey.Question{{
		Name: "pipelineResource",
		Prompt: &survey.Select{
			Message: "Select a resource type to create :",
			Options: allResourceType(),
		},
	}}

	err := survey.Ask(qs, &answer, res.askOpts)
	if err != nil {
		return Error(err)
	}

	res.pipelineResource.Spec.Type = cast(answer)

	return nil
}

func (res *resource) askGitParams() error {
	urlParam, err := askParam("url", res.askOpts)
	if err != nil {
		return err
	}
	if urlParam.Name != "" {
		res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, urlParam)
	}

	revisionParam, err := askParam("revision", res.askOpts)
	if err != nil {
		return err
	}
	if revisionParam.Name != "" {
		res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, revisionParam)
	}

	return nil
}

func (res *resource) askStorageParams() error {
	options := []string{"gcs", "build-gcs"}

	storageType, err := askToSelect("Select a storage type", options, res.askOpts)
	if err != nil {
		return err
	}
	param := v1alpha1.ResourceParam{}
	param.Name, param.Value = "type", storageType
	res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, param)

	switch storageType {
	case "gcs":
		locationParam, err := askParam("location", res.askOpts)
		if err != nil {
			return err
		}
		if locationParam.Name != "" {
			res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, locationParam)
		}

		dirParam, err := askParam("dir", res.askOpts)
		if err != nil {
			return err
		}
		if dirParam.Name != "" {
			res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, dirParam)
		}

	case "build-gcs":
		locationParam, err := askParam("location", res.askOpts)
		if err != nil {
			return err
		}
		if locationParam.Name != "" {
			res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, locationParam)
		}

		artifactOpts := []string{"ZipArchive", "TarGzArchive", "Manifest"}
		artifactType, err := askToSelect("Select an artifact type", artifactOpts, res.askOpts)
		if err != nil {
			return err
		}
		artifactParam := v1alpha1.ResourceParam{}
		artifactParam.Name, artifactParam.Value = "artifactType", artifactType
		res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, artifactParam)
	}

	// ask secret
	secret, err := askSecret("GOOGLE_APPLICATION_CREDENTIALS", res.askOpts)
	if err != nil {
		return err
	}
	res.pipelineResource.Spec.SecretParams = append(res.pipelineResource.Spec.SecretParams, secret)

	return nil
}

func (res *resource) askImageParams() error {
	urlParam, err := askParam("url", res.askOpts)
	if err != nil {
		return err
	}
	if urlParam.Name != "" {
		res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, urlParam)
	}

	digestParam, err := askParam("digest", res.askOpts)
	if err != nil {
		return err
	}
	if digestParam.Name != "" {
		res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, digestParam)
	}

	return nil
}

func (res *resource) askClusterParams() error {
	nameParam, err := askParam("name", res.askOpts)
	if err != nil {
		return err
	}
	if nameParam.Name != "" {
		res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, nameParam)
	}

	urlParam, err := askParam("url", res.askOpts)
	if err != nil {
		return err
	}
	if urlParam.Name != "" {
		res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, urlParam)
	}

	usernameParam, err := askParam("username", res.askOpts)
	if err != nil {
		return err
	}
	if usernameParam.Name != "" {
		res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, usernameParam)
	}

	secure, err := askToSelect("Is the cluster secure?", []string{"yes", "no"}, res.askOpts)
	if err != nil {
		return err
	}
	insecureParam := v1alpha1.ResourceParam{}
	insecureParam.Name = "insecure"
	if secure == "yes" {
		insecureParam.Value = "false"
	} else {
		insecureParam.Value = "true"
	}
	res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, insecureParam)

	qs := "Which authentication technique you want to use?"
	qsOpts := []string{
		"password",
		"token",
	}
	ans, err := askToSelect(qs, qsOpts, res.askOpts)
	if err != nil {
		return err
	}
	switch ans {
	case qsOpts[0]: // Using password authentication technique
		passwordParam, err := askPassword(res.askOpts)
		if err != nil {
			return err
		}
		if passwordParam.Name != "" {
			res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, passwordParam)
		}
		if secure == "yes" {
			qs := "How do you want to set cadata?"
			qsOpts := []string{
				"Passing plain text as parameters",
				"Using existing kubernetes secrets",
			}
			ans, err := askToSelect(qs, qsOpts, res.askOpts)
			if err != nil {
				return err
			}
			switch ans {
			case qsOpts[0]: // plain text
				cadataParam, err := askParam("cadata", res.askOpts)
				if err != nil {
					return err
				}
				if cadataParam.Name != "" {
					res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, cadataParam)
				}

			case qsOpts[1]: // kubernetes secrets
				secret, err := askSecret("cadata", res.askOpts)
				if err != nil {
					return err
				}
				res.pipelineResource.Spec.SecretParams = append(res.pipelineResource.Spec.SecretParams, secret)

			}
		} else {
			cadataParam := v1alpha1.ResourceParam{}
			cadataParam.Name = "cadata"
			res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, cadataParam)
		}

	case qsOpts[1]: // Using token authentication technique
		qs := "How do you want to set cluster credentials?"
		qsOpts := []string{
			"Passing plain text as parameters",
			"Using existing kubernetes secrets",
		}
		ans, err := askToSelect(qs, qsOpts, res.askOpts)
		if err != nil {
			return err
		}
		switch ans {
		case qsOpts[0]: // plain text
			tokenParam, err := askParam("token", res.askOpts)
			if err != nil {
				return err
			}
			if tokenParam.Name != "" {
				res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, tokenParam)
			}
			if secure == "yes" {
				cadataParam, err := askParam("cadata", res.askOpts)
				if err != nil {
					return err
				}
				if cadataParam.Name != "" {
					res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, cadataParam)
				}
			} else {
				// doing this as pipeline returns error if cadata is not present.
				param := v1alpha1.ResourceParam{}
				param.Name = "cadata"
				res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, param)
			}

		case qsOpts[1]: // kubernetes secretes
			secret, err := askSecret("token", res.askOpts)
			if err != nil {
				return err
			}
			res.pipelineResource.Spec.SecretParams = append(res.pipelineResource.Spec.SecretParams, secret)

			if secure == "yes" {
				secret, err := askSecret("cadata", res.askOpts)
				if err != nil {
					return err
				}
				res.pipelineResource.Spec.SecretParams = append(res.pipelineResource.Spec.SecretParams, secret)
			} else {
				caSecret := v1alpha1.SecretParam{}
				caSecret.FieldName = "cadata"
				res.pipelineResource.Spec.SecretParams = append(res.pipelineResource.Spec.SecretParams, caSecret)
			}
		}
	}
	return nil
}

func (res *resource) askPullRequestParams() error {
	urlParam, err := askParam("url", res.askOpts)
	if err != nil {
		return err
	}
	if urlParam.Name != "" {
		res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, urlParam)
	}

	//ask for the secrets
	qsOpts := []string{"Yes", "No"}
	qs := "Do you want to set secrets ?"

	ans, e := askToSelect(qs, qsOpts, res.askOpts)
	if e != nil {
		return e
	}
	if ans == qsOpts[1] {
		return nil
	}

	secret, err := askSecret("githubToken", res.askOpts)
	if err != nil {
		return err
	}
	res.pipelineResource.Spec.SecretParams = append(res.pipelineResource.Spec.SecretParams, secret)

	return nil
}

func (res *resource) askCloudEventParams() error {
	targetURIParam, err := askParam("targetURI", res.askOpts)
	if err != nil {
		return err
	}
	if targetURIParam.Name != "" {
		res.pipelineResource.Spec.Params = append(res.pipelineResource.Spec.Params, targetURIParam)
	}
	return nil
}

func askParam(paramName string, askOpts survey.AskOpt) (v1alpha1.ResourceParam, error) {
	var param v1alpha1.ResourceParam
	var qs = []*survey.Question{{
		Name: "value",
		Prompt: &survey.Input{
			Message: fmt.Sprintf("Enter a value for %s : ", paramName),
		},
	}}

	err := survey.Ask(qs, &param, askOpts)
	if err != nil {
		return param, Error(err)
	}

	if param.Value != "" {
		param.Name = paramName
	}

	return param, nil
}

func askSecret(secret string, askOpts survey.AskOpt) (v1alpha1.SecretParam, error) {
	var secrect v1alpha1.SecretParam
	secrect.FieldName = secret
	var qs = []*survey.Question{
		{
			Name: "secretKey",
			Prompt: &survey.Input{
				Message: fmt.Sprintf("Secret Key for %s :", secret),
			},
		},
		{
			Name: "secretName",
			Prompt: &survey.Input{
				Message: fmt.Sprintf("Secret Name for %s :", secret),
			},
		},
	}

	err := survey.Ask(qs, &secrect, askOpts)
	if err != nil {
		return secrect, Error(err)
	}

	return secrect, nil
}

func askToSelect(message string, options []string, askOpts survey.AskOpt) (string, error) {
	var ans string
	var qs1 = []*survey.Question{{
		Name: "params",
		Prompt: &survey.Select{
			Message: message,
			Options: options,
		},
	}}

	err := survey.Ask(qs1, &ans, askOpts)
	if err != nil {
		return "", Error(err)
	}

	return ans, nil
}

func askPassword(askOpts survey.AskOpt) (v1alpha1.ResourceParam, error) {
	var param v1alpha1.ResourceParam
	var qs = []*survey.Question{{
		Name: "value",
		Prompt: &survey.Password{
			Message: fmt.Sprintf("Enter a value for password :"),
		},
	}}

	err := survey.Ask(qs, &param, askOpts)
	if err != nil {
		return param, Error(err)
	}

	param.Name = "password"

	return param, nil
}

func allResourceType() []string {
	var resType []string

	for _, val := range v1alpha1.AllResourceTypes {
		resType = append(resType, string(val))
	}

	sort.Strings(resType)
	return resType
}

func cast(answer string) v1alpha1.PipelineResourceType {
	return v1alpha1.PipelineResourceType(answer)
}

func Error(err error) error {
	switch err.Error() {
	case "interrupt":
		return errors.New("interrupt")
	default:
		return err
	}
}

func validate(name string, p cli.Params) error {
	c, err := p.Clients()
	if err != nil {
		return err
	}

	if _, err := c.Tekton.TektonV1alpha1().PipelineResources(p.Namespace()).Get(name, metav1.GetOptions{}); err == nil {
		return errors.New("resource already exist")
	}

	return nil
}
