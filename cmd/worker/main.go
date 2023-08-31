// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/golang/glog"

	"github.com/Xilinx-CNS/kubernetes-onload/pkg/client_helper"
	"github.com/Xilinx-CNS/kubernetes-onload/pkg/control_plane"
)

func main() {
	// Enable logging to stderr.
	flag.Parse()
	err := flag.Lookup("logtostderr").Value.Set("true")
	if err != nil {
		glog.Fatalf("Failed to initialise Onload worker: %v", err)
	}

	// Get the container identification from env.
	podNamespace := os.Getenv("POD_NAMESPACE")
	podName := os.Getenv("POD_NAME")
	containerName := os.Getenv("CONTAINER_NAME")

	// A mount point with the Onload control plane server.
	onloadCPServerPath := os.Getenv("ONLOAD_CP_SERVER_PATH")

	// There is a race condition when the container is starting, and the
	// Onload kernel module is loading at the same time, resulting in not
	// all files being available in the container.
	_, err = os.Stat("/dev/onload")
	if err != nil {
		glog.Fatalf("/dev/onload is not available: %v", err)
	}

	// Create an API helper to get the container ID.
	clientHelper, err := client_helper.NewClientHelper()
	if err != nil {
		glog.Fatal("Failed to create K8s helper: ", err)
	}

	// Prepare a context with timeout for API calls.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	containerID, err := clientHelper.GetContainerID(ctx,
		podNamespace, podName,
		client_helper.NewDefaultContainerHelper(containerName))
	if err != nil {
		glog.Fatal("Failed to get container ID: ", err)
	}

	glog.Info("Found container ID ", containerID)

	// Configure Onload kernel module to launch
	// Onload control plane within a container.
	err = control_plane.Configure(onloadCPServerPath, containerID,
		control_plane.NewKernelParametersWriter())
	if err != nil {
		glog.Fatal("Failed to configure Onload control plane: ", err)
	}

	// Sleep forever.
	select {}
}
