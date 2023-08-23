// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package main

import (
	"flag"

	"github.com/golang/glog"

	"github.com/Xilinx-CNS/kubernetes-onload/pkg/deviceplugin"
)

func main() {
	flag.Parse()
	err := flag.Lookup("logtostderr").Value.Set("true")
	if err != nil {
		glog.Fatalf("Failed to initialise device plugin: %v", err)
	}
	glog.Info("Starting device plugin")
	manager, err := deviceplugin.NewNicManager()
	if err != nil {
		glog.Fatalf("Failed to initialise device plugin: %v", err)
	}
	glog.Infof("SFC interfaces: %s", manager.GetInterfaces())
	glog.Infof("Device files: %s", manager.GetDeviceFiles())
	manager.Run() /* Blocks forever */
}
