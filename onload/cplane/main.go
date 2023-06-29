// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	cplaneServerPathFilePath       = "/sys/module/onload/parameters/cplane_server_path"
	cplaneServerParamsFilePath     = "/sys/module/onload/parameters/cplane_server_params"
	critclFilePath                 = "/usr/bin/crictl"
	crioPrefix                     = "cri-o://"
	cplaneServerParamsFormatString = "exec %s /opt/onload/sbin/onload_cp_server -K"

	timeoutTime = 5 * time.Second
)

var (
	podName      string
	podNamespace string

	containerID string
)

func warnAndWait(message string) {
	glog.Warningf("%s. Will retry in %v", message, timeoutTime)
	time.Sleep(timeoutTime)
}

func main() {

	podName = os.Getenv("POD_NAME")
	podNamespace = os.Getenv("POD_NAMESPACE")

	cfg, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("Failed to get cluster config (%v)", err)
	}

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		glog.Fatalf("Failed to create clientset (%v)", err)
	}
	pod := &corev1.Pod{}

	// The information about the pod may be out of date when it is received,
	// so we enter a "while" loop to keep checking about the pod and container
	// until they report as running.
	for {
		pod, err = clientSet.CoreV1().Pods(podNamespace).Get(
			context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			glog.Fatalf("Failed to find pod %q in namespace %q (%v)",
				podName, podNamespace, err)
		}

		// We only expect a single container within the pod, so error out if a
		// different number is found
		if len(pod.Status.ContainerStatuses) != 1 {
			glog.Fatalf("Expecting 1 containerStatus in pod %q, found %d",
				podName, len(pod.Status.ContainerStatuses))
		}
		containerStatus := pod.Status.ContainerStatuses[0]

		if !containerStatus.Ready {
			warnAndWait("Container is not ready")
			continue
		}

		if !*containerStatus.Started {
			warnAndWait("Container has not started")
			continue
		}

		if containerStatus.State.Running == nil {
			warnAndWait("Container is not running")
			continue
		}

		containerID = containerStatus.ContainerID

		break
	}

	// The ContainerID reported has a "cri-o://" prefix before the actual ID.
	// We need to remove this so that crictl is happy.
	// If this is not present then we are using an unknown container runtime
	// interface, in that case abort.
	containerID, found := strings.CutPrefix(containerID, crioPrefix)
	if !found {
		glog.Fatalf("Could not find %q as a prefix to the containerID %v",
			crioPrefix, containerID)
	}

	err = os.WriteFile(
		cplaneServerPathFilePath,
		[]byte(critclFilePath),
		0644,
	)
	if err != nil {
		glog.Fatalf("Couldn't write cplane_server_path")
	}

	cplaneServerParams :=
		fmt.Sprintf(cplaneServerParamsFormatString, containerID)

	err = os.WriteFile(
		cplaneServerParamsFilePath,
		[]byte(cplaneServerParams),
		0644,
	)
	if err != nil {
		glog.Fatalf("Couldn't write cplane_server_params")
	}

	// Block here indefinitely to keep DaemonSet alive.
	select {}
}
