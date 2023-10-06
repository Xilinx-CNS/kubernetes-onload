// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package control_plane

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	mockOnloadCPServerPath = "/mock/sbin/onload_cp_server"

	mockCRIOContainerID       = "cri-o://0123456789abcdef"
	mockContainerdContainerID = "containerd://fedcba9876543210"
)

type mockKernelParametersWriter struct {
}

func NewMockKernelParametersWriter() KernelParametersWriter {
	return &mockKernelParametersWriter{}
}

func (*mockKernelParametersWriter) SetControlPlaneServerPath(path string) error {
	Expect(path).To(Equal("/usr/bin/crictl"))
	return nil
}

func (*mockKernelParametersWriter) SetControlPlaneServerParams(params string) error {
	Expect(params).To(Equal("exec 0123456789abcdef /mock/sbin/onload_cp_server -K"))
	return nil
}

var _ = Describe("Configure Onload to launch the control plane process within a container", func() {
	It("Should work with the CRI-O runtime", func() {
		err := Configure(mockOnloadCPServerPath, mockCRIOContainerID,
			NewMockKernelParametersWriter())
		Expect(err).Should(Succeed())
	})

	It("Should fail with non CRI-O runtime, e.g. containerd", func() {
		err := Configure(mockOnloadCPServerPath, mockContainerdContainerID,
			NewMockKernelParametersWriter())
		Expect(err).Should(HaveOccurred())
	})
})
