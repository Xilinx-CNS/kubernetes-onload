// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package client_helper

import (
	"context"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClientHelper struct {
	Config *rest.Config
}

// Create the new helper with the in-cluster config.
func NewClientHelper() (*ClientHelper, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return &ClientHelper{
		Config: config,
	}, nil
}

type ContainerHelper interface {
	GetContainerID(pod *corev1.Pod) *string
}

type DefaultContainerHelper struct {
	ContainerName string
}

func (h *DefaultContainerHelper) GetContainerID(pod *corev1.Pod) *string {
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name != h.ContainerName {
			continue
		}

		if status.Ready && *status.Started && status.State.Running != nil {
			return &status.ContainerID
		}

		return nil
	}

	return nil
}

func NewDefaultContainerHelper(containerName string) ContainerHelper {
	return &DefaultContainerHelper{
		ContainerName: containerName,
	}
}

const (
	backOffInitial = 100 * time.Millisecond
	backOffMax     = 1 * time.Second

	backOffID = "defaultBackOff"
)

// Get ID of the container identified by the pod's name, namespace
// and also the container's name if there are more than one in the pod.
// The returned string is in the format "<type>://<container_id>".
func (ch *ClientHelper) GetContainerID(ctx context.Context,
	podNamespace string, podName string, helper ContainerHelper,
) (string, error) {
	cs, err := kubernetes.NewForConfig(ch.Config)
	if err != nil {
		return "", err
	}

	backOff := flowcontrol.NewBackOff(backOffInitial, backOffMax)

	for {
		pod, err := cs.CoreV1().Pods(podNamespace).Get(
			ctx, podName, metav1.GetOptions{})
		if err != nil {
			return "", err
		}

		containerID := helper.GetContainerID(pod)
		if containerID != nil {
			return *containerID, nil
		}

		backOff.Next(backOffID, time.Now())

		select {
		case <-time.After(backOff.Get(backOffID)):
			continue
		case <-ctx.Done():
			return "", context.DeadlineExceeded
		}
	}
}
