// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package main

import (
	"context"
	"strings"
	"time"

	"github.com/golang/glog"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

// GetDevicePluginOptions is used by the kubernetes device manager to check
// which optional features we implement. Since we don't use either we can just
// return false for both, which should prevent any headaches if the device
// plugin gets requested to do something it doesn't support.
func (rpc *RPCServer) GetDevicePluginOptions(
	context.Context,
	*pluginapi.Empty,
) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		PreStartRequired:                false,
		GetPreferredAllocationAvailable: false,
	}, nil
}

// PreStartContainer is not used here, but required by the device plugin API
func (rpc *RPCServer) PreStartContainer(
	context.Context,
	*pluginapi.PreStartContainerRequest,
) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

// ListAndWatch is called by the kubelet at start of day;
// loops forever sending status updates periodically.
// NOTE: In this version of the device plugin the state we report never
// changes, as the device plugin exits with an error if the set of
// visible NICs changes.
func (rpc *RPCServer) ListAndWatch(
	emtpy *pluginapi.Empty,
	stream pluginapi.DevicePlugin_ListAndWatchServer,
) error {
	glog.Info("ListAndWatch")
	for {
		rpc.manager.CheckNics()
		resp := &pluginapi.ListAndWatchResponse{}
		resp.Devices = rpc.manager.devices
		err := stream.Send(resp)
		if err != nil {
			glog.Errorf("ListAndWatch failed send (%v)", err)
		}
		time.Sleep(time.Second * 60)
	}
}

// Allocate is called by the kubelet when a container on this host requests
// one of our NICs. This is the function that arranges for the onload devices
// and files to be passed through into the container
func (rpc *RPCServer) Allocate(
	ctx context.Context,
	reqs *pluginapi.AllocateRequest,
) (*pluginapi.AllocateResponse, error) {
	glog.Info("Allocate")

	resps := pluginapi.AllocateResponse{}

	for _, req := range reqs.ContainerRequests {
		devIDs := strings.Join(req.DevicesIDs, ",")
		glog.Infof("  Devices: %s", devIDs)
		resp := pluginapi.ContainerAllocateResponse{
			Envs:    rpc.manager.envs,
			Devices: rpc.manager.deviceFiles,
			Mounts:  rpc.manager.mounts,
		}
		resps.ContainerResponses = append(resps.ContainerResponses, &resp)
	}
	return &resps, nil
}

// GetPreferredAllocation is not used here, but required by the device plugin API
func (rpc *RPCServer) GetPreferredAllocation(
	context.Context,
	*pluginapi.PreferredAllocationRequest,
) (*pluginapi.PreferredAllocationResponse, error) {
	return &pluginapi.PreferredAllocationResponse{}, nil
}
