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
	onloadCPServerPath string, onloadCPServerParams string,
	containerID string, kpw KernelParametersWriter,
) error {
	// Split the container runtime type from identifier.
	containerID, found := strings.CutPrefix(containerID, "cri-o://")
	if !found {
		return fmt.Errorf("unsupported container runtime in %s", containerID)
	}

	glog.Info("CRI-O container ID is ", containerID)

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
