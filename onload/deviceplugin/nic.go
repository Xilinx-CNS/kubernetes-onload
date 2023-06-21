// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package main

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/golang/glog"
)

// Takes the output from lshw and returns the device name for each solarflare
// device.
func parseOutput(output string) []string {
	// "lshw -short -class network" sample output:
	// H/W path            Device     Class          Description
	// =========================================================
	// /0/100/1b/0         enp2s0f0   network        XtremeScale SFC9250 10/25/40/50/100G Ethernet Controller
	// /0/100/1b/0.1       enp2s0f1   network        XtremeScale SFC9250 10/25/40/50/100G Ethernet Controller
	// /0/100/1c.1/0       eno1       network        NetXtreme BCM5720 Gigabit Ethernet PCIe
	// /0/100/1c.1/0.1     eno2       network        NetXtreme BCM5720 Gigabit Ethernet PCIe

	lines := strings.Split(output, "\n")

	var interfaces []string

	// Assume that we are running as root, if not then we would have to skip
	// an additional line at the start of the output
	skip_lines := 2
	end_lines := 1
	if os.Geteuid() != 0 {
		skip_lines = 3
		end_lines = 2
	}

	for _, line := range lines[skip_lines : len(lines)-end_lines] {
		// This regex makes the assumption that all interface names only
		// contain either lowercase letters or numbers. If that is not true,
		// then this should be updated to reflect that.
		r := regexp.MustCompile("([a-z0-9]+) *network *.*SFC")
		out := r.FindStringSubmatch(line)
		if out != nil {
			// It is safe to access out[1] here since the return value of
			// FindStringSubmatch is an array where the first value is the
			// whole string and any subsequent values are the submatches.
			// In this case since there is a submatch that should match the
			// device name if FindStringSubmatch returns non-nil then there
			// will be at least 2 elements in the return array.
			interfaces = append(interfaces, out[1])
		}
	}
	return interfaces
}

// Returns a list of the Solarflare interfaces present on the node
func queryNics() ([]string, error) {
	// Depending on what information we are looking for in the output I think it
	// is quite tempting to retrieve the information in a json format, then
	// parse this using golang's built-in features.
	bytes, err := exec.Command("lshw", "-short", "-class", "network").CombinedOutput()
	output := string(bytes)
	if err != nil {
		glog.Error(output)
		glog.Fatalf("error while listing sfc devices : %v", err)
	}
	interfaces := parseOutput(output)
	if len(interfaces) == 0 {
		return nil, errors.New("no sfc devices found")
	}
	return interfaces, nil
}
