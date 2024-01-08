// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package deviceplugin

import (
	"errors"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

const (
	sysClassNetPath  = "/sys/class/net/"
	solarflareVendor = "0x1924"
)

func isSFCNic(devicePath string) bool {
	deviceDir, err := os.Stat(path.Join(devicePath, "device"))
	if errors.Is(err, os.ErrNotExist) {
		// Not a physical device, so won't have the "vendor" file
		return false
	} else if err != nil {
		glog.Errorf("Failed to stat %s (%v)", devicePath, err)
		return false
	}
	if !deviceDir.IsDir() {
		return false
	}

	data, err := os.ReadFile(path.Join(devicePath, "device", "vendor"))
	if errors.Is(err, os.ErrNotExist) {
		// File doesn't exist but that is fine
		return false
	} else if err != nil {
		glog.Errorf("Error reading %s (%v)",
			path.Join(devicePath, "device", "vendor"), err)
		return false
	}

	vendor := strings.TrimSuffix(string(data), "\n")
	return vendor == solarflareVendor
}

// Get numa node from sysfs files.
// -1 means no specific numa node / unknown
func getNumaNode(devicePath string) int64 {
	data, err := os.ReadFile(path.Join(devicePath, "device", "numa_node"))
	if errors.Is(err, os.ErrNotExist) {
		// File doesn't exist but that is fine, return -1
		return -1
	} else if err != nil {
		glog.Errorf("Error reading %s (%v)",
			path.Join(devicePath, "device", "vendor"), err)
		return -1
	}
	numaString := strings.TrimSuffix(string(data), "\n")
	node, err := strconv.ParseInt(numaString, 10, 64)
	if err != nil {
		glog.Errorf("Error parse int from string %s (%v)",
			numaString, err)
		return -1
	}
	return node
}

func readSysFiles() ([]nic, error) {
	infos, err := os.ReadDir(sysClassNetPath)
	if err != nil {
		glog.Errorf("Error reading %s (%v)", sysClassNetPath, err)
		return []nic{}, err
	}

	interfaces := []nic{}
	for _, info := range infos {
		devicePath := path.Join(sysClassNetPath, info.Name())
		if !isSFCNic(devicePath) {
			continue
		}
		nic := nic{
			name: info.Name(),
			numa: getNumaNode(devicePath),
		}
		interfaces = append(interfaces, nic)
	}
	return interfaces, nil
}

// Returns a list of the Solarflare interfaces present on the node
func queryNics() ([]nic, error) {
	interfaces, err := readSysFiles()
	if err != nil {
		glog.Errorf("Failed to list interfaces (%v)", err)
		return []nic{}, err
	}
	return interfaces, nil
}
