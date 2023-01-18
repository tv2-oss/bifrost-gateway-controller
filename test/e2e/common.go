/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file contain tests that are not e2e per se, however the e2e
// tests assume an external cluster with the controller deployed using
// 'production' means, e.g. a Helm chart and thus we have some basic
// tests here to validate the deployment of the controller on the
// external cluster.

package e2esuite

import (
	"context"
	"fmt"
	"io"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ExecCmdInPodBySelector(cl client.Client, restClient rest.Interface, cfg *rest.Config,
	selector client.ListOption, namespace string,
	command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {

	podList := &corev1.PodList{}
	opts := []client.ListOption{client.InNamespace(namespace), selector}
	err := cl.List(context.Background(), podList, opts...)
	if err != nil {
		return err
	}

	podName := podList.Items[0].ObjectMeta.Name

	return ExecCmdInPod(restClient, cfg,
		podName, namespace,
		command, stdin, stdout, stderr)
}

func ExecCmdInPod(restClient rest.Interface, cfg *rest.Config,
	podName string, namespace string,
	command string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {

	execReq := restClient.
		Post().
		Namespace(namespace).
		Resource("pods").
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			//Container: "container",
			Command:   []string{"sh", "-c", command},
			//Stdin:     true,
			Stdout:    true,
			//Stderr:    true,
		}, runtime.NewParameterCodec(scheme.Scheme))

	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", execReq.URL())
	if err != nil {
		return fmt.Errorf("error while creating remote command executor: %v", err)
	}

	err = exec.Stream(remotecommand.StreamOptions{
		//Stdin:  stdin,
		Stdout: stdout,
		//Stderr: stderr,
		Tty:    false,
	})

	return nil
}
