/*
Copyright 2023 TV 2 DANMARK A/S

Licensed under the Apache License, Version 2.0 (the "License") with the
following modification to section 6. Trademarks:

Section 6. Trademarks is deleted and replaced by the following wording:

6. Trademarks. This License does not grant permission to use the trademarks and
trade names of TV 2 DANMARK A/S, including but not limited to the TV 2® logo and
word mark, except (a) as required for reasonable and customary use in describing
the origin of the Work, e.g. as described in section 4(c) of the License, and
(b) to reproduce the content of the NOTICE file. Any reference to the Licensor
must be made by making a reference to "TV 2 DANMARK A/S", written in capitalized
letters as in this example, unless the format in which the reference is made,
requires lower case letters.

You may not use this software except in compliance with the License and the
modifications set out above.

You may obtain a copy of the license at:

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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
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

	if len(podList.Items) != 1 {
		return fmt.Errorf("multiple PODs found. Use more specific selector")
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
			Command: []string{"sh", "-c", command},
			Stdout:  true,
		}, runtime.NewParameterCodec(scheme.Scheme))

	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", execReq.URL())
	if err != nil {
		return fmt.Errorf("error while creating remote command executor: %v", err)
	}

	return exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdout: stdout,
		Tty:    false,
	})
}
