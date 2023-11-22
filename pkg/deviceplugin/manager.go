// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package deviceplugin

import (
	"errors"
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

func (manager *NicManager) GetInterfaces() []string {
	return manager.interfaces
}

func (manager *NicManager) GetDeviceFiles() []*pluginapi.DeviceSpec {
	return manager.deviceFiles
}

// NewNicManager allocates and initialises a new NicManager
func NewNicManager(maxPods int,
	usePreload, mountOnload, needNic bool,
) (*NicManager, error) {
	nics, err := queryNics()
	if err != nil {
		return nil, err
	}
	if len(nics) == 0 && needNic {
		return nil, errors.New("no sfc devices found")
	}
	manager := &NicManager{
		interfaces:     nics,
		maxPodsPerNode: maxPods,
	}
	manager.envs = make(map[string]string)
	manager.initDevices()

	if usePreload && mountOnload {
		return nil, errors.New("setting both usePreload and mountOnload is not supported")
	}

	manager.initMounts(usePreload, mountOnload)

	manager.rpcServer = NewRPCServer(manager)
	return manager, nil
}

// Initialises the set of devices to advertise to kubernetes
func (manager *NicManager) initDevices() {
	manager.devices = []*pluginapi.Device{}
	fmt.Println(manager.interfaces)
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
		glog.Fatalf("Failed to query nics (%v)", err)
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
