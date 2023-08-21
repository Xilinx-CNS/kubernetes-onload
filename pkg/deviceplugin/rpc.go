// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package main

import (
	"context"
	"net"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

func dialUnix(ctx context.Context, path string) (net.Conn, error) {
	return net.DialTimeout("unix", path, 5*time.Second)
}

// Specialises grpc.Dial to use a domain socket and accept a timeout
func grpcDial(sockPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()
	return grpc.DialContext(
		ctx,
		sockPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithContextDialer(dialUnix),
	)
}

func getRPCSockPath() string {
	sockPath := path.Join(pluginapi.DevicePluginPath, "sfc-deviceplugin.sock")
	glog.Infof("RPC socket path is %s", sockPath)
	return sockPath
}

// RPCServer runs a gRPC server talking the device plugin API
type RPCServer struct {
	manager        *NicManager
	listenSockPath string
}

// NewRPCServer initialises (but does not start) a new RPC server
func NewRPCServer(manager *NicManager) *RPCServer {
	manager.wg.Add(1)
	return &RPCServer{
		manager:        manager,
		listenSockPath: getRPCSockPath(),
	}
}

// Serve Sets up a listening socket and serves grpc on it
func (rpc *RPCServer) Serve() {
	defer rpc.manager.wg.Done()

	err := os.RemoveAll(rpc.listenSockPath)
	if err != nil {
		glog.Fatalf("Failed to delete old socket %s (%v)", rpc.listenSockPath, err)
	}

	err = os.MkdirAll(filepath.Dir(rpc.listenSockPath), os.ModePerm)
	if err != nil {
		glog.Fatalf("Failed to create directory for socket %s (%v)", rpc.listenSockPath, err)
	}

	listenSock, err := net.Listen("unix", rpc.listenSockPath)
	if err != nil {
		glog.Fatalf("Failed to listen on %s (%v)", rpc.listenSockPath, err)
	}

	grpcServer := grpc.NewServer([]grpc.ServerOption{}...)
	pluginapi.RegisterDevicePluginServer(grpcServer, rpc)
	glog.Infof("RPC server listening on %s", rpc.listenSockPath)
	err = grpcServer.Serve(listenSock)
	if err != nil {
		glog.Fatalf("grpcServer.Serve failed (%v)", err)
	}
}

func (rpc *RPCServer) isUp() bool {
	conn, err := grpcDial(rpc.listenSockPath, 100*time.Millisecond)
	if err != nil {
		glog.Infof("RPC server is not up (%v)", err)
		return false
	}
	conn.Close()
	glog.Infof("RPC server is up")
	return true
}

// WaitUntilUp waits until the RPC server is up
func (rpc *RPCServer) WaitUntilUp() {
	for !rpc.isUp() {
		time.Sleep(1 * time.Second)
	}
}

// Register the device plugin with the kubernetes API
func (rpc *RPCServer) Register() {
	glog.Infof("Connecting to kubelet sock %s", pluginapi.KubeletSocket)
	conn, err := grpcDial(pluginapi.KubeletSocket, 5*time.Second)
	if err != nil {
		glog.Fatalf("Failed to connect to kubelet socket %s (%v)", pluginapi.KubeletSocket, err)
	}
	defer conn.Close()

	opts, err := rpc.GetDevicePluginOptions(context.Background(),
		&pluginapi.Empty{})
	if err != nil {
		glog.Warningf("Failed to get DevicePluginOptions (%v)", err)
		// It should be fine to continue, as the device manager will call
		// GetDevicePluginOptions if any optional functions are availible.
		opts = nil
	}

	client := pluginapi.NewRegistrationClient(conn)
	req := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     filepath.Base(rpc.listenSockPath),
		ResourceName: resourceName,
		Options:      opts,
	}

	_, err = client.Register(context.Background(), req)
	if err != nil {
		glog.Fatalf("Failed to register device plugin (%v)", err)
	}
}
