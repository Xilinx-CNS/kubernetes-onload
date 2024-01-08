// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package deviceplugin

import (
	"errors"
	"os"
	"path"
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

func readSysFiles() ([]string, error) {
	infos, err := os.ReadDir(sysClassNetPath)
	if err != nil {
		glog.Errorf("Error reading %s (%v)", sysClassNetPath, err)
		return []string{}, err
	}
	interfaces := []string{}
	for _, info := range infos {
		if isSFCNic(path.Join(sysClassNetPath, info.Name())) {
			interfaces = append(interfaces, info.Name())
		}
	}
	return interfaces, nil
}

// Returns a list of the Solarflare interfaces present on the node
func queryNics() ([]string, error) {
	interfaces, err := readSysFiles()
	if err != nil {
		glog.Errorf("Failed to list interfaces (%v)", err)
		return []string{}, err
	}
	return interfaces, nil
}
