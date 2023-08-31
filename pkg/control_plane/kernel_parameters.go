// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package control_plane

import (
	"os"
	"path/filepath"
)

// Provides access to the Onload sysfs configuration knobs.
type KernelParametersWriter interface {
	SetControlPlaneServerPath(path string) error
	SetControlPlaneServerParams(params string) error
}

type kernelParametersWriter struct {
}

func NewKernelParametersWriter() KernelParametersWriter {
	return &kernelParametersWriter{}
}

func writeOnloadParameterFile(where string, what string) error {
	where = filepath.Join("/sys/module/onload/parameters", where)
	return os.WriteFile(where, []byte(what), 0644)
}

func (kernelParametersWriter) SetControlPlaneServerPath(path string) error {
	return writeOnloadParameterFile("cplane_server_path", path)
}

func (kernelParametersWriter) SetControlPlaneServerParams(params string) error {
	return writeOnloadParameterFile("cplane_server_params", params)
}
