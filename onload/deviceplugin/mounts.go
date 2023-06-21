// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package main

import (
	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// DeviceMounts are the device nodes required to run onload in a pod
var DeviceMounts = []string{
	"/dev/onload",
	"/dev/onload_epoll",
	"/dev/sfc_char",
}

// addDeviceMount arranges for this device file to be mounted inside containers
func (manager *NicManager) addDeviceMount(path string) {
	spec := pluginapi.DeviceSpec{
		HostPath:      path,
		ContainerPath: path,
		Permissions:   "mrw",
	}
	glog.Infof("Mount %s ---> %s", path, path)
	manager.deviceFiles = append(manager.deviceFiles, &spec)
}

// Initialises the set of host files to mount in each container
func (manager *NicManager) initMounts() {
	manager.deviceFiles = []*pluginapi.DeviceSpec{}

	for _, path := range DeviceMounts {
		manager.addDeviceMount(path)
	}
}
