// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package client_helper

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

type mockContainerHelper struct {
	mockContainerID *string
}

func (m *mockContainerHelper) GetContainerID(*corev1.Pod) *string {
	return m.mockContainerID
}

var _ = Describe("Test GetContainerID API function", func() {
	It("Should return container ID", func() {
		clientHelper := &ClientHelper{
			Config: cfg,
		}

		// This test does not require a context with the timeout,
		// but we still create it to prevent blocking forever in
		// the case of a software defect.
		timeoutContext, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		containerID, err := clientHelper.GetContainerID(
			timeoutContext, mockNamespaceName, mockPodName,
			&mockContainerHelper{
				mockContainerID: &mockContainerName,
			})

		Expect(err).Should(Succeed())
		Expect(containerID).To(Equal(mockContainerName))
	})

	It("Should timeout because container helper is mocking not-found behaviour", func() {
		clientHelper := &ClientHelper{
			Config: cfg,
		}

		timeoutContext, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		containerID, err := clientHelper.GetContainerID(
			timeoutContext, mockNamespaceName, mockPodName,
			&mockContainerHelper{
				mockContainerID: nil,
			})

		Expect(err).To(Equal(context.DeadlineExceeded))
		Expect(containerID).To(Equal(""))
	})
})

var _ = Describe("Test internal container helper", func() {
	var mockFooContainerName = "foo-running"
	var mockBarContainerName = "bar-not-running"
	var mockBazContainerName = "baz-does-not-exist"

	var mockFooContainerID = "cri-o://0123456789abcdef"

	mockPod := &corev1.Pod{
		Status: corev1.PodStatus{
			ContainerStatuses: []corev1.ContainerStatus{
				{
					Name:    mockFooContainerName,
					Ready:   true,
					Started: ptr.To(true),
					State: corev1.ContainerState{
						Running: &corev1.ContainerStateRunning{},
					},
					ContainerID: mockFooContainerID,
				},
				{
					Name:  mockBarContainerName,
					Ready: false,
				},
			},
		},
	}

	It("Should find running container", func() {
		containerID := NewDefaultContainerHelper(mockFooContainerName).GetContainerID(mockPod)
		Expect(*containerID).To(Equal(mockFooContainerID))
	})

	It("Should find container but return nil because it is not running", func() {
		containerID := NewDefaultContainerHelper(mockBarContainerName).GetContainerID(mockPod)
		Expect(containerID).To(BeNil())
	})

	It("Should not find container", func() {
		containerID := NewDefaultContainerHelper(mockBazContainerName).GetContainerID(mockPod)
		Expect(containerID).To(BeNil())
	})
})
