// Copyright © 2021 The Tekton Authors.
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

package workspaces

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tektoncd/cli/pkg/cli"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8sRes "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/cmd/cp"
	"k8s.io/kubectl/pkg/cmd/exec"
	"k8s.io/kubectl/pkg/scheme"
)

const (
	image          = "nginx:alpine"
	defaultStorage = "1Gi"
)

type LocalToWorkspace struct {
	Kube       k8s.Interface
	Config     *rest.Config
	Stream     *cli.Stream
	Name       string
	DirPath    string
	PvcStorage string
	Namespace  string
	Overwrite  bool
}

func (l *LocalToWorkspace) Create() error {

	var err error
	l.DirPath, err = filepath.Abs(l.DirPath)
	if err != nil {
		return fmt.Errorf("failed to evaluate abs path for the directory: %s", l.DirPath)
	}

	stat, err := os.Stat(l.DirPath)
	if err != nil {
		return fmt.Errorf("%s doesn't exist in local filesystem", l.DirPath)
	}

	if !stat.IsDir() {
		return fmt.Errorf("%s path is not a directory", l.DirPath)
	}

	dirName := l.dirName()
	if l.PvcStorage == "" {
		l.PvcStorage = defaultStorage
	}

	// creates a pvc with the workspace name
	_, pvcErr := l.Kube.CoreV1().PersistentVolumeClaims(l.Namespace).
		Create(context.TODO(), l.pvcTemplate(), metav1.CreateOptions{})
	if pvcErr != nil && !(errors.IsAlreadyExists(pvcErr) && l.Overwrite) {
		return pvcErr
	}

	// creates an nginx pod mounted on the pvc created
	createdPod, err := l.Kube.CoreV1().Pods(l.Namespace).
		Create(context.TODO(), l.podTemplate(l.Name, dirName), metav1.CreateOptions{})
	if err != nil {
		return err
	}

	// wait for pod to be in running state for copying files
	err = l.waitForPodRunning(createdPod.Name, time.Duration(60)*time.Second)
	if err != nil {
		return err
	}

	l.setKubernetesDefault()

	out := l.Stream.Out
	fmt.Fprintln(out, "")

	// if pvc already exists and overwrite is true then clean the dir before copying files
	if errors.IsAlreadyExists(pvcErr) && l.Overwrite {
		fmt.Fprintln(out, "Clearing existing files in workspace..")
		if err = l.cleanWs(createdPod.Name, dirName); err != nil {
			return err
		}
	}

	fmt.Fprintf(out, "Copying files from '%s' to the workspace..\n", l.DirPath)

	if err = l.copyFiles(createdPod.Name); err != nil {
		return err
	}

	fmt.Fprintln(out, "Files copied !!!")

	return l.cleanUp(createdPod.Name)
}

func (l *LocalToWorkspace) dirName() string {
	arr := strings.Split(l.DirPath, "/")
	return arr[len(arr)-1]
}

func (l *LocalToWorkspace) pvcTemplate() *corev1.PersistentVolumeClaim {

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      l.Name,
			Namespace: l.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]k8sRes.Quantity{
					corev1.ResourceStorage: k8sRes.MustParse(l.PvcStorage),
				},
			},
		},
	}
}

func (l *LocalToWorkspace) podTemplate(pvcName string, dirName string) *corev1.Pod {

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: l.Name + "-",
			Namespace:    l.Namespace,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{{
				Name: "vol",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			}},
			Containers: []corev1.Container{
				{
					Name:  l.Name,
					Image: image,
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "vol",
							MountPath: "/workspace/" + dirName,
						},
					},
				},
			},
		},
	}
}

func (l *LocalToWorkspace) waitForPodRunning(podName string, timeout time.Duration) error {
	fmt.Fprintf(l.Stream.Out, "Preparing %s workspace", l.Name)
	return wait.PollImmediate(time.Second, timeout, l.isPodRunning(podName))
}

func (l *LocalToWorkspace) isPodRunning(podName string) wait.ConditionFunc {
	return func() (bool, error) {
		fmt.Fprintf(l.Stream.Out, ".")

		pod, err := l.Kube.CoreV1().Pods(l.Namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		switch pod.Status.Phase {
		case corev1.PodRunning:
			return true, nil
		case corev1.PodFailed, corev1.PodSucceeded:
			return false, fmt.Errorf("pod ran to completion")
		}
		return false, nil
	}
}

func (l *LocalToWorkspace) cleanWs(podName, dirName string) error {

	command := fmt.Sprintf("rm -rf /workspace/%s/*", dirName)

	execOpt := &exec.ExecOptions{
		StreamOptions: exec.StreamOptions{
			IOStreams: genericclioptions.IOStreams{
				Out:    l.Stream.Out,
				ErrOut: l.Stream.Err,
			},
			Namespace:     l.Namespace,
			PodName:       podName,
			ContainerName: l.Name,
		},
		Config:    l.Config,
		PodClient: l.Kube.CoreV1(),
		Command:   []string{"sh", "-c", command},
		Executor:  &exec.DefaultRemoteExecutor{},
	}

	if err := execOpt.Run(); err != nil {
		return err
	}

	return nil
}

func (l *LocalToWorkspace) copyFiles(podName string) error {

	cpCmd := cp.CopyOptions{
		Container:    l.Name,
		Namespace:    l.Namespace,
		ClientConfig: l.Config,
		Clientset:    l.Kube,
		NoPreserve:   false,
		IOStreams: genericclioptions.IOStreams{
			In:     l.Stream.In,
			Out:    l.Stream.Out,
			ErrOut: l.Stream.Err,
		},
	}

	destination := fmt.Sprintf("%s:/workspace", podName)
	ar := []string{l.DirPath, destination}

	err := cpCmd.Run(ar)
	if err != nil {
		return err
	}

	return nil
}

func (l *LocalToWorkspace) setKubernetesDefault() {
	l.Config.GroupVersion = &schema.GroupVersion{Group: "", Version: "v1"}
	l.Config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	if l.Config.APIPath == "" {
		l.Config.APIPath = "/api"
	}
}

func (l *LocalToWorkspace) cleanUp(podName string) error {
	return l.Kube.CoreV1().Pods(l.Namespace).
		Delete(context.TODO(), podName, metav1.DeleteOptions{})
}
