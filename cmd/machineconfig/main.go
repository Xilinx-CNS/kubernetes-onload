// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rh-ecosystem-edge/kernel-module-management/pkg/mcproducer"
)

func main() {

	outPtr := flag.String("out", "",
		`A filepath where the contents of the MachineConfig should be written.
Optional, if empty the results will be printed to stdout`)
	mod := flag.String("module", "", "Name of module to be loaded. Required")
	pool := flag.String("pool", "worker",
		"Name of machineConfigPool to be used. Optional")
	name := flag.String("name", "",
		"Name of the machineConfig to be created. Required")
	image := flag.String("image", "",
		"Name of the container image to be used. Required")

	flag.Parse()

	if *mod == "" || *name == "" || *image == "" {
		flag.Usage()
		os.Exit(1)
	}

	str, err := mcproducer.ProduceMachineConfig(*name, *pool,
		*image, *mod)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not generate MachineConfig (%v)\n", err)
		os.Exit(1)
	}

	if *outPtr != "" {
		err := os.WriteFile(*outPtr, []byte(str), 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write contents to file %s (%v)",
				*outPtr, str)
		}
	} else {
		fmt.Print(str)
	}
}
