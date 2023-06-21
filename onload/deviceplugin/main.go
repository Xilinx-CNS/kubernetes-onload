// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package main

import (
	"flag"

	"github.com/golang/glog"
)

func main() {
	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")
	glog.Info("Starting device plugin")
	manager, err := NewNicManager()
	if err != nil {
		glog.Fatalf("Failed to initialise device plugin: %v", err)
	}
	glog.Infof("SFC interfaces: %s", manager.interfaces)
	glog.Infof("Device files: %s", manager.deviceFiles)
	manager.Run() /* Blocks forever */
}
