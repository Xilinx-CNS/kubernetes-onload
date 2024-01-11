// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package control_plane

import (
	"fmt"
	"slices"
	"strings"

	"github.com/golang/glog"
)

// I couldn't find a definitive list anywhere, so this is a list of the
// "graduated" container runtimes from cncf.io
// https://landscape.cncf.io/card-mode?category=container-runtime
var knownGoodContainerRuntimes = []string{"cri-o", "containerd"}

// Configure the loaded Onload kernel module to launch
// the Onload control plane process within a container.
func Configure(
	onloadCPServerPath string, onloadCPServerParams string,
	containerID string, kpw KernelParametersWriter,
) error {
	// Split the container runtime type from identifier.
	runtimeAndIdentifier := strings.Split(containerID, "://")
	if len(runtimeAndIdentifier) != 2 {
		return fmt.Errorf("unexpected format of containerID %s", containerID)
	}

	if !slices.Contains(knownGoodContainerRuntimes, runtimeAndIdentifier[0]) {
		// Warn, but continue
		glog.Warningf("Unexpected container runtime: %s."+
			" Running in a compatible container runtime (CRI) is required."+
			" Execution will continue, but starting the cplane may fail.",
			runtimeAndIdentifier[0])
	}
	containerID = runtimeAndIdentifier[1]

	glog.Info("Container ID found", "runtime",
		runtimeAndIdentifier[0], "id", containerID)

	// Set the Onload control plane path.
	crictlPath := "/usr/bin/crictl"
	err := kpw.SetControlPlaneServerPath(crictlPath)
	if err != nil {
		return err
	}

	glog.Info("Updated Onload control plane server path to ", crictlPath)

	// Prepend the whitespace to non-empty cplane parameters.
	if onloadCPServerParams != "" {
		onloadCPServerParams = " " + onloadCPServerParams
	}

	// Set the Onload control plane params.
	params := fmt.Sprintf("exec %s %s%s", containerID,
		onloadCPServerPath, onloadCPServerParams)
	err = kpw.SetControlPlaneServerParams(params)
	if err != nil {
		return err
	}

	glog.Info("Updated Onload control plane server params to ", params)

	return nil
}
