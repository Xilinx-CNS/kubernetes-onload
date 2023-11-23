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

// NicManagerConfig describes the configuration of the NicManager.
type NicManagerConfig struct {
	MaxPodsPerNode int
	SetPreload     bool
	MountOnload    bool
	NeedNic        bool
}

// Ideally this would be const, but go doesn't support const structs.
var DefaultConfig = NicManagerConfig{
	MaxPodsPerNode: 100,
	SetPreload:     true,
	MountOnload:    false,
	NeedNic:        true,
}

// NicManager holds all the state required by the device plugin
type NicManager struct {
	// interfaces is used to check the presence of any sfc nics on the node.
	// Currently it is just used as a check for existence and no additional
	// logic takes place.
	interfaces  []string
	deviceFiles []*pluginapi.DeviceSpec
	mounts      []*pluginapi.Mount
	devices     []*pluginapi.Device
	envs        map[string]string
	rpcServer   *RPCServer
	wg          sync.WaitGroup
	config      NicManagerConfig
}

func (manager *NicManager) GetInterfaces() []string {
	return manager.interfaces
}

func (manager *NicManager) GetDeviceFiles() []*pluginapi.DeviceSpec {
	return manager.deviceFiles
}

// NewNicManager allocates and initialises a new NicManager
func NewNicManager(
	config NicManagerConfig,
) (*NicManager, error) {
	nics, err := queryNics()
	if err != nil {
		return nil, err
	}
	if len(nics) == 0 && config.NeedNic {
		return nil, errors.New("no sfc devices found")
	}
	manager := &NicManager{
		interfaces: nics,
		config:     config,
	}
	manager.envs = make(map[string]string)
	manager.initDevices()

	if manager.config.SetPreload && manager.config.MountOnload {
		return nil, errors.New("setting both usePreload and mountOnload is not supported")
	}
	manager.initMounts()

	manager.rpcServer = NewRPCServer(manager)
	return manager, nil
}

// Initialises the set of devices to advertise to kubernetes
func (manager *NicManager) initDevices() {
	manager.devices = []*pluginapi.Device{}
	fmt.Println(manager.interfaces)
	for i := 0; i < manager.config.MaxPodsPerNode; i++ {
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
