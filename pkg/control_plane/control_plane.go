// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package control_plane

import (
	"fmt"
	"strings"

	"github.com/golang/glog"
)

// Configure the loaded Onload kernel module to launch
// the Onload control plane process within a container.
func Configure(
	onloadCPServerPath string, containerID string, kpw KernelParametersWriter,
) error {
	// Split the container runtime type from identifier.
	containerID, found := strings.CutPrefix(containerID, "cri-o://")
	if !found {
		return fmt.Errorf("Unsupported container runtime in %s", containerID)
	}

	glog.Info("CRI-O container ID is ", containerID)

	// Set the Onload control plane path.
	crictlPath := "/usr/bin/crictl"
	err := kpw.SetControlPlaneServerPath(crictlPath)
	if err != nil {
		return err
	}

	glog.Info("Updated Onload control plane server path to ", crictlPath)

	// Set the Onload control plane params.
	params := fmt.Sprintf("exec %s %s -K", containerID, onloadCPServerPath)
	err = kpw.SetControlPlaneServerParams(params)
	if err != nil {
		return err
	}

	glog.Info("Updated Onload control plane server params to ", params)

	return nil
}
