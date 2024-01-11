// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package control_plane

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	mockOnloadCPServerPath   = "/mock/sbin/onload_cp_server"
	mockOnloadCPServerParams = "-mock -foo -bar"

	mockCRIOContainerID       = "cri-o://0123456789abcdef"
	mockContainerdContainerID = "containerd://fedcba9876543210"
)

type mockKernelParametersWriter struct {
	expectedParams string
}

func NewMockKernelParametersWriter(expectedParams string) KernelParametersWriter {
	return &mockKernelParametersWriter{
		expectedParams: expectedParams,
	}
}

func (*mockKernelParametersWriter) SetControlPlaneServerPath(path string) error {
	Expect(path).To(Equal("/usr/bin/crictl"))
	return nil
}

func (mkpw *mockKernelParametersWriter) SetControlPlaneServerParams(params string) error {
	Expect(params).To(Equal(mkpw.expectedParams))
	return nil
}

var _ = Describe("Configure Onload to launch the control plane process within a container", func() {
	It("Should work with the CRI-O runtime", func() {
		expectedParams := "exec 0123456789abcdef /mock/sbin/onload_cp_server -mock -foo -bar"
		err := Configure(mockOnloadCPServerPath, mockOnloadCPServerParams,
			mockCRIOContainerID, NewMockKernelParametersWriter(expectedParams))
		Expect(err).Should(Succeed())
	})

	It("Should work with the CRI-O runtime and empty parameter list", func() {
		expectedParams := "exec 0123456789abcdef /mock/sbin/onload_cp_server"
		err := Configure(mockOnloadCPServerPath, "", mockCRIOContainerID,
			NewMockKernelParametersWriter(expectedParams))
		Expect(err).Should(Succeed())
	})

	It("Should fail with non CRI-O runtime, e.g. containerd", func() {
		notUsed := ""
		err := Configure(mockOnloadCPServerPath, mockOnloadCPServerParams,
			mockContainerdContainerID, NewMockKernelParametersWriter(notUsed))
		Expect(err).Should(HaveOccurred())
	})
})
