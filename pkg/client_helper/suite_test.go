// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package client_helper

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"testing"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

// Must be a variable to get its address.
var mockContainerName string = "mock-container"

func TestK8sHelper(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Go clients Helper Suite")
}

const (
	mockNamespaceName = "mock-namespace"
	mockPodName       = "mock-pod"

	mockImage = "mock-image"
)

var (
	ctx    context.Context
	cancel context.CancelFunc
)

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.TODO())

	By("Bootstrapping test environment")
	testEnv = &envtest.Environment{}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Creating mock resources.

	// Warning! There are no built-in controllers that are running
	// in the test context. It means that the DaemonSet won't create
	// Pods, Pods won't create Containers, etc.
	mockNamespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: mockNamespaceName,
		},
	}
	Expect(k8sClient.Create(ctx, mockNamespace)).To(Succeed())

	mockPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mockPodName,
			Namespace: mockNamespaceName,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				corev1.Container{
					Name:  mockContainerName,
					Image: mockImage,
				},
			},
		},
	}
	Expect(k8sClient.Create(ctx, mockPod)).To(Succeed())
})

var _ = AfterSuite(func() {
	cancel()
	By("Tearing down test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
