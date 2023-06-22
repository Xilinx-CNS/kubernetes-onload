// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package main

import (
	"os"
	"path"
	"strings"

	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	hostPathPrefix = "/opt/onload"
	lib64path      = "/usr/lib64"
	destLibDir     = "/opt/onload"
)

// deviceMounts are the device nodes required to run onload in a pod
var deviceMounts = []string{
	"/dev/onload",
	"/dev/onload_epoll",
	"/dev/sfc_char",
}

// libraryMounts are the shared libraries required to run onload in a pod
var libraryMounts = []string{
	"libonload.so",
	"libonload_ext.so",
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

// addFileMount arranges for this host path to be mounted inside containers
func (manager *NicManager) addFileMount(hostPath, containerPath string) {
	spec := pluginapi.Mount{
		HostPath:      hostPath,
		ContainerPath: containerPath,
		ReadOnly:      true,
	}
	glog.Infof("Mount %s ---> %s", hostPath, containerPath)
	manager.mounts = append(manager.mounts, &spec)
}

// Returns all versioned names of this library file
func findLibraryVersions(filename string) ([]string, error) {
	hostLibDir := path.Join(hostPathPrefix, lib64path)
	infos, err := os.ReadDir(hostLibDir)
	if err != nil {
		return nil, err
	}
	filenames := []string{}

	for _, info := range infos {
		if info.IsDir() {
			continue
		}
		name := info.Name()
		if strings.HasPrefix(name, filename) {
			filenames = append(filenames, name)
		}
	}
	return filenames, nil
}

// addLibraryMounts arranges for this library to be mounted inside the
// container. There are two complications here:
//  1. The library may exist as both name.so and name.so.<version>;
//     if so we must mount both versions
//  2. Different distros look in different directories for libraries. To be
//     compatibile with all distros we must mount the library in multiple
//     directories inside the container
func (manager *NicManager) addLibraryMounts(baseFilename string) error {
	filenames, err := findLibraryVersions(baseFilename)
	if err != nil {
		return err
	}
	for _, filename := range filenames {
		hostPath := path.Join(hostPathPrefix, lib64path, filename)
		manager.addFileMount(hostPath, path.Join(destLibDir, lib64path, filename))
	}
	return nil
}

// Initialises the set of host files to mount in each container
func (manager *NicManager) initMounts() {
	manager.deviceFiles = []*pluginapi.DeviceSpec{}

	for _, path := range deviceMounts {
		manager.addDeviceMount(path)
	}

	for _, path := range libraryMounts {
		err := manager.addLibraryMounts(path)
		if err != nil {
			glog.Warningf("Failed to add library mount for %s (%v)", path, err)
		}
	}

	manager.envs["LD_PRELOAD"] = path.Join(destLibDir, lib64path, "libonload.so")
}
