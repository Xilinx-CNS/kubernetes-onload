// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package control_plane

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
)

const (
	mockOnloadCPServerPath   = "/mock/sbin/onload_cp_server"
	mockOnloadCPServerParams = "-mock -foo -bar"

	goodContainerID = "0123456789abcdef"

	mockCRIOContainerID       = "cri-o://" + goodContainerID
	mockContainerdContainerID = "containerd://" + goodContainerID
	mockTooShortContainerID   = "cri-o" + goodContainerID
	mockTooLongContainerId    = mockCRIOContainerID + "://suffix"
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

var _ = DescribeTable(
	"Configure Onload to launch the control plane process within a container",
	func(
		serverPath, serverParams, containerID, expectedParams string,
		matcher types.GomegaMatcher,
	) {
		err := Configure(serverPath, serverParams,
			containerID, NewMockKernelParametersWriter(expectedParams))
		Expect(err).Should(matcher)
	},
	Entry("should work with the CRI-O runtime", mockOnloadCPServerPath, mockOnloadCPServerParams, mockCRIOContainerID, "exec 0123456789abcdef /mock/sbin/onload_cp_server -mock -foo -bar", Succeed()),
	Entry("should work with the CRI-O runtime and empty parameter list", mockOnloadCPServerPath, "", mockCRIOContainerID, "exec 0123456789abcdef /mock/sbin/onload_cp_server", Succeed()),
	Entry("should pass with non CRI-O (but still good) runtime, e.g. containerd", mockOnloadCPServerPath, "", mockContainerdContainerID, "exec 0123456789abcdef /mock/sbin/onload_cp_server", Succeed()),
	Entry("should reject a too short ContainerId", mockOnloadCPServerPath, "", mockTooShortContainerID, "", HaveOccurred()),
	Entry("should reject a too long ContainerId", mockOnloadCPServerPath, "", mockTooLongContainerId, "", HaveOccurred()),
)
