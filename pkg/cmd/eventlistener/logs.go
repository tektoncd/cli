// Copyright © 2020 The Tekton Authors.
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

package eventlistener

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/tektoncd/cli/pkg/cli"
	"github.com/tektoncd/cli/pkg/options"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func logCommand(p cli.Params) *cobra.Command {
	opts := options.NewLogOptions(p)

	eg := `
Show logs of EventListener pods: 
	
    tkn eventlistener logs eventlistenerName

Show 2 lines of most recent logs from all EventListener pods:

    tkn eventlistener logs eventListenerName -t 2`
	c := &cobra.Command{
		Use:                   "logs",
		DisableFlagsInUseLine: true,
		Short:                 "Show EventListener logs",
		Example:               eg,
		SilenceUsage:          true,
		Annotations: map[string]string{
			"commandType": "main",
		},
		Args: cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.Tail <= 0 && opts.Tail != -1 {
				return fmt.Errorf("tail cannot be 0 or less than 0 unless -1 for all pods")
			}

			err := getEventListener(args[0], p)
			if err != nil {
				return err
			}

			s := &cli.Stream{
				Out: cmd.OutOrStdout(),
				Err: cmd.OutOrStderr(),
			}

			return logs(args[0], p, s, opts)
		},
	}
	c.Flags().Int64VarP(&opts.Tail, "tail", "t", 10, "Number of most recent log lines to show. Specify -1 for all logs from each pod.")
	_ = c.MarkZshCompPositionalArgumentCustom(1, "__tkn_get_eventlistener")
	return c
}

func getEventListener(elName string, p cli.Params) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	_, err = cs.Triggers.TriggersV1alpha1().EventListeners(p.Namespace()).Get(elName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get EventListener %s: %v", elName, err)
	}

	return nil
}

func logs(elName string, p cli.Params, s *cli.Stream, opts *options.LogOptions) error {
	cs, err := p.Clients()
	if err != nil {
		return fmt.Errorf("failed to create tekton client")
	}

	elPods, err := cs.Kube.CoreV1().Pods(p.Namespace()).List(metav1.ListOptions{LabelSelector: "eventlistener=" + elName})
	if err != nil {
		return fmt.Errorf("failed to get pods for EventListener %s", elName)
	}

	if len(elPods.Items) == 0 {
		fmt.Fprintf(s.Out, "No pods available for EventListener %s\n", elName)
		return nil
	}

	for _, pod := range elPods.Items {
		podName := pod.Name
		podLopOpts := &corev1.PodLogOptions{}
		// -1 represents getting all logs from each pod. Tail is 10 by default.
		if opts.Tail != -1 {
			podLopOpts.TailLines = &opts.Tail
		}
		podLogReq := cs.Kube.CoreV1().Pods(p.Namespace()).GetLogs(podName, podLopOpts)
		podLogs, err := podLogReq.Stream()
		if err != nil {
			return err
		}
		defer podLogs.Close()

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		if err != nil {
			return err
		}

		fmt.Println()
		scanner := bufio.NewScanner(buf)
		for scanner.Scan() {
			fmt.Printf("[%s-%s]: "+scanner.Text()+"\n", elName, podName)
		}
	}

	return nil
}
