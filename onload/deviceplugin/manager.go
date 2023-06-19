// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package main

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// The name of the resource we provide; this is the key that the user
// puts in a pod spec's "resources" section to request onload.
const resourceName = "amd.com/onload"

// NicManager holds all the state required by the device plugin
type NicManager struct {
	// interfaces is used to check the presence of any sfc nics on the node.
	// Currently it is just used as a check for existence and no additional
	// logic takes place.
	interfaces     []string
	deviceFiles    []*pluginapi.DeviceSpec
	mounts         []*pluginapi.Mount
	devices        []*pluginapi.Device
	envs           map[string]string
	maxPodsPerNode int
	rpcServer      *RPCServer
	wg             sync.WaitGroup
}

// Determines the maximum number of pods to allow on each node
func getMaxPodsPerNode() int {
	// Currently an arbitrary number. This may or may not have to be dynamic
	// depending on how the sfc devices are passed through to the pods (VF,
	// ipvlan, macvlan) and the cluster's network configuration (multus,
	// calico, etc.).
	return 100
}

// NewNicManager allocates and initialises a new NicManager
func NewNicManager() (*NicManager, error) {
	nics, err := queryNics()
	if err != nil {
		return nil, err
	}
	manager := &NicManager{
		interfaces:     nics,
		maxPodsPerNode: getMaxPodsPerNode(),
	}
	manager.envs = make(map[string]string)
	manager.initDevices()
	manager.initMounts()

	manager.rpcServer = NewRPCServer(manager)
	return manager, nil
}

// Initialises the set of devices to advertise to kubernetes
func (manager *NicManager) initDevices() {
	manager.devices = []*pluginapi.Device{}
	fmt.Println(manager.interfaces)
	if len(manager.interfaces) == 0 {
		return
	}
	for i := 0; i < manager.maxPodsPerNode; i++ {
		name := fmt.Sprintf("sfc-%v", i)
		device := &pluginapi.Device{
			ID:     name,
			Health: pluginapi.Healthy,
		}
		manager.devices = append(manager.devices, device)
	}
}

// CheckNics checks that the NICs in the system are still healthy
func (manager *NicManager) CheckNics() {
	interfaces, err := queryNics()
	if err != nil {
		glog.Fatal("No sfc nics found")
	}
	if !reflect.DeepEqual(interfaces, manager.interfaces) {
		glog.Fatalf("SFC interfaces on host have changed (%s -> %s)",
			manager.interfaces, interfaces)
	}
}

// Run runs the device plugin, blocking forever
func (manager *NicManager) Run() {
	go manager.rpcServer.Serve()
	manager.rpcServer.WaitUntilUp()
	manager.rpcServer.Register()
	manager.wg.Wait() // Blocks forever or until rpc server hits a fatal error
}
