// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package main

import (
	"flag"

	"github.com/golang/glog"

	"github.com/Xilinx-CNS/kubernetes-onload/pkg/deviceplugin"
)

func main() {

	config := deviceplugin.DefaultConfig

	flag.IntVar(&config.MaxPodsPerNode, "maxPods",
		deviceplugin.DefaultConfig.MaxPodsPerNode,
		"Number of Onload resources to advertise on each node")
	flag.BoolVar(&config.NeedNic, "needNic",
		deviceplugin.DefaultConfig.NeedNic,
		"Should the Device Plugin fail if no compatible nics are found")
	flag.BoolVar(&config.SetPreload, "setPreload",
		deviceplugin.DefaultConfig.SetPreload,
		"Should the device plugin set the LD_PRELOAD environment variable in the pod")
	flag.BoolVar(&config.MountOnload, "mountOnload",
		deviceplugin.DefaultConfig.MountOnload,
		"Should the device plugin mount the onload script into the pod")
	flag.StringVar(&config.HostPathPrefix, "hostOnloadPath",
		deviceplugin.DefaultConfig.HostPathPrefix,
		"Base location of onload files on the host filesystem")
	flag.StringVar(&config.BaseMountPath, "baseMountPath",
		deviceplugin.DefaultConfig.BaseMountPath,
		"Prefix to be applied to all file mounts in the container's filesystem")
	flag.StringVar(&config.BinMountPath, "binMountPath",
		deviceplugin.DefaultConfig.BinMountPath,
		"Location to mount onload binaries in the container's filesystem")
	flag.StringVar(&config.LibMountPath, "libMountPath",
		deviceplugin.DefaultConfig.LibMountPath,
		"Location to mount onload libraries in the container's filesystem")
	flag.Parse()
	err := flag.Lookup("logtostderr").Value.Set("true")
	if err != nil {
		glog.Fatalf("Failed to initialise device plugin: %v", err)
	}
	glog.Info("Starting device plugin")
	manager, err := deviceplugin.NewNicManager(config)
	if err != nil {
		glog.Fatalf("Failed to initialise device plugin: %v", err)
	}
	glog.Infof("SFC interfaces: %s", manager.GetInterfaces())
	glog.Infof("Device files: %s", manager.GetDeviceFiles())
	manager.Run() /* Blocks forever */
}
