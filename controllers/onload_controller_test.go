// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package controllers

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	onloadv1alpha1 "github.com/Xilinx-CNS/kubernetes-onload/api/v1alpha1"
	mock_client "github.com/Xilinx-CNS/kubernetes-onload/mocks/client"

	kmm "github.com/kubernetes-sigs/kernel-module-management/api/v1beta1"
)

var _ = Describe("Testing createModule function", func() {
	var onload *onloadv1alpha1.Onload

	BeforeEach(func() {
		onload = &onloadv1alpha1.Onload{}
	})

	It("Should work with a valid onload CR", func() {
		_, err := createModule(onload, "example", "example")
		Expect(err).Should(Succeed())
	})

	It("Should have the correct number of kernel mappings", func() {
		for i := 0; i < 10; i++ {
			onload.Spec.Onload.KernelMappings = append(
				onload.Spec.Onload.KernelMappings,
				onloadv1alpha1.OnloadKernelMapping{},
			)
		}
		module, err := createModule(onload, "example", "example")
		Expect(err).Should(Succeed())

		Expect(len(module.Spec.ModuleLoader.Container.KernelMappings)).
			To(Equal(len(onload.Spec.Onload.KernelMappings)))
	})
})

var _ = Describe("Testing using mocked client", func() {
	var (
		r                     *OnloadReconciler
		mockClient            *mock_client.MockClient
		mockSubResourceClient *mock_client.MockSubResourceClient
	)

	BeforeEach(func() {

		mockCtrl := gomock.NewController(GinkgoT())
		defer mockCtrl.Finish()

		mockClient = mock_client.NewMockClient(mockCtrl)
		mockSubResourceClient = mock_client.NewMockSubResourceClient(mockCtrl)

		r = &OnloadReconciler{
			Client: mockClient,
		}
	})

	It("should evict pods using Onload", func() {

		onloadResource := resource.NewQuantity(1, resource.DecimalSI)

		node := corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: "bar"}}

		// PodList to be return when trying to list pods
		allPods := corev1.PodList{
			Items: []corev1.Pod{
				{ObjectMeta: metav1.ObjectMeta{Name: "A"}, Spec: corev1.PodSpec{NodeName: "foo"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "B"},
					Spec: corev1.PodSpec{
						NodeName: "bar", Containers: []corev1.Container{
							{},
							{Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{"amd.com/onload": *onloadResource}}},
						},
					},
				},
				{ObjectMeta: metav1.ObjectMeta{Name: "C"}, Spec: corev1.PodSpec{NodeName: "foo"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "D"}, Spec: corev1.PodSpec{NodeName: "baz"}},
			},
		}

		// Listing pods
		mockClient.EXPECT().
			List(gomock.Any(), &corev1.PodList{}, gomock.Any()).
			SetArg(1, allPods).
			Return(nil).
			Times(1)

		// Creating the eviction for the pod
		mockSubResourceClient.EXPECT().
			Create(gomock.Any(), &allPods.Items[1], gomock.Any()).
			Return(nil).
			Times(1)

		// Create the subresource client
		mockClient.EXPECT().
			SubResource("eviction").
			Return(mockSubResourceClient).
			Times(1)

		Expect(r.evictOnloadedPods(ctx, node)).Should(Equal(&ctrl.Result{RequeueAfter: 5 * time.Second}))
	})

	Context("Node label management", func() {
		var (
			onload onloadv1alpha1.Onload
			nodes  corev1.NodeList
		)

		BeforeEach(func() {
			onload = onloadv1alpha1.Onload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: onloadv1alpha1.Spec{
					Selector: map[string]string{
						"key": "value",
					},
					Onload: onloadv1alpha1.OnloadSpec{
						Version: "foo",
					},
				},
			}

			// NodeList to return when listing nodes
			nodes = corev1.NodeList{
				Items: []corev1.Node{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"key": "value"},
						},
					},
				},
			}
		})

		It("should label Nodes with kmm label", func() {
			pods := corev1.PodList{}

			// Listing nodes
			listNodesCall := mockClient.EXPECT().
				List(gomock.Any(), &corev1.NodeList{}, gomock.Any()).
				SetArg(1, nodes).
				Return(nil).
				Times(1)

			// Listing pods
			mockClient.EXPECT().
				List(gomock.Any(), &corev1.PodList{}, gomock.Any()).
				SetArg(1, pods).
				Return(nil).
				Times(len(nodes.Items)).After(listNodesCall)

			// Patching nodes to add the label
			mockClient.EXPECT().
				Patch(gomock.Any(), &nodes.Items[0], gomock.Any()).
				Return(nil).
				Times(1)

			Expect(r.addKmmLabelsToNodes(ctx, &onload)).Should(Equal(&ctrl.Result{Requeue: true}))
		})

		It("should label nodes with Onload label", func() {

			// Add the kmm label to the second node so that the onload label
			// will be added
			nodes.Items[0].Labels[kmmLabelName(onload.Name, onload.Namespace)] = onload.Spec.Onload.Version

			// Add a new node to the list with the kmm label, but with the wrong
			// value. This node should not be patched.
			nodes.Items = append(nodes.Items, corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"key": "value",
						kmmLabelName(onload.Name, onload.Namespace): "bar",
					},
				},
			})

			// Listing nodes
			mockClient.EXPECT().
				List(gomock.Any(), &corev1.NodeList{}, gomock.Any()).
				SetArg(1, nodes).
				Return(nil).
				Times(1)

			// Patching nodes to add the label
			mockClient.EXPECT().
				Patch(gomock.Any(), &nodes.Items[0], gomock.Any()).
				Return(nil).
				Times(1)

			Expect(r.addOnloadLabelsToNodes(ctx, &onload)).Should(Equal(&ctrl.Result{Requeue: true}))
		})

		It("should remove stale kmm labels from nodes that no longer match the selector", func() {
			labelKey := kmmLabelName(onload.Name, onload.Namespace)

			// Add the kmm label to the node, and remove "key" so that it
			// doesn't match onload.Spec.Selector
			nodes.Items[0].Labels[labelKey] = onload.Spec.Onload.Version
			delete(nodes.Items[0].Labels, "key")

			// Listing nodes
			mockClient.EXPECT().
				List(gomock.Any(), &corev1.NodeList{}, gomock.Any()).
				SetArg(1, nodes).
				Return(nil).
				Times(1)

			// Patching nodes to remove the label
			mockClient.EXPECT().
				Patch(gomock.Any(), &nodes.Items[0], gomock.Any()).
				Return(nil).
				Times(1)

			Expect(r.removeStaleLabels(ctx, &onload, labelKey)).Should(Equal(&ctrl.Result{Requeue: true}))
		})

	})

	Context("Testing node updates", func() {
		var (
			onload onloadv1alpha1.Onload
			node   corev1.Node
		)

		BeforeEach(func() {
			onload = onloadv1alpha1.Onload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "name",
					Namespace: "namespace",
				},
				Spec: onloadv1alpha1.Spec{
					Selector: map[string]string{
						"key": "value",
					},
					Onload: onloadv1alpha1.OnloadSpec{
						Version: "foo",
					},
				},
			}
			node = corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "node",
					Labels: map[string]string{},
				},
			}

		})

		It("should requeue after removing Onload label", func() {
			node.Labels[onloadLabelName(onload.Name, onload.Namespace)] = "bar"

			// Patch call for the removal of the onload label from the node
			mockClient.EXPECT().
				Patch(gomock.Any(), &node, gomock.Any()).
				Return(nil).
				Times(1)

			Expect(r.handleNodeUpdate(ctx, &onload, node)).Should(Equal(&ctrl.Result{Requeue: true}))
		})

		It("should requeue if the Device Plugin Pod still exists", func() {
			// This isn't a test of the label selector functionality, only the
			// logic present in the controller. This means that we don't have
			// to label the pods.
			pods := corev1.PodList{
				Items: []corev1.Pod{
					{Spec: corev1.PodSpec{NodeName: node.Name}},
					{Spec: corev1.PodSpec{NodeName: "foo"}},
					{Spec: corev1.PodSpec{NodeName: node.Name}},
				},
			}

			mockClient.EXPECT().
				List(gomock.Any(), &corev1.PodList{}, gomock.Any()).
				SetArg(1, pods).
				Return(nil).Times(1)

			Expect(r.handleNodeUpdate(ctx, &onload, node)).Should(Equal(&ctrl.Result{RequeueAfter: 5 * time.Second}))
		})

		It("should requeue after removing kmm label", func() {
			// No device plugin pods
			listDPPodsCall := mockClient.EXPECT().
				List(gomock.Any(), &corev1.PodList{}, gomock.Any()).
				Return(nil).
				Times(1)

			// No evictions needed, tests for that logic is handled in another
			// unit test.
			mockClient.EXPECT().
				List(gomock.Any(), &corev1.PodList{}, gomock.Any()).
				Return(nil).
				Times(1).
				After(listDPPodsCall)

			// Patch call for the removal of the kmm label from the node
			mockClient.EXPECT().
				Patch(gomock.Any(), &node, gomock.Any()).
				Return(nil).
				Times(1)

			Expect(r.handleNodeUpdate(ctx, &onload, node)).Should(Equal(&ctrl.Result{Requeue: true}))
		})

	})

})

func generateNamespaceName() (string, error) {
	name := "test-ns-" + rand.SafeEncodeString(rand.String(5))
	out := corev1.Namespace{}
	err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, &out)
	if err != nil && !apierrors.IsNotFound(err) {
		return "", err
	} else if err == nil {
		return "", errors.New("Error starting test: Namespace already exists")
	}
	return name, nil
}

const (
	timeout         = 10 * time.Second
	pollingInterval = 250 * time.Millisecond
)

var _ = Describe("onload controller", func() {
	Context("testing onload controller", func() {
		var onload *onloadv1alpha1.Onload
		var testNamespace *corev1.Namespace

		BeforeEach(func() {
			namespaceName, err := generateNamespaceName()
			Expect(err).Should(Succeed())

			onload = &onloadv1alpha1.Onload{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "onload-test",
					Namespace: namespaceName,
				},
				Spec: onloadv1alpha1.Spec{
					Selector: map[string]string{
						"key": "",
					},
					Onload: onloadv1alpha1.OnloadSpec{
						KernelMappings: []onloadv1alpha1.OnloadKernelMapping{
							{
								KernelModuleImage: "",
								Regexp:            "",
							},
						},
						UserImage: "image:tag",
						Version:   "",
					},
					DevicePlugin: onloadv1alpha1.DevicePluginSpec{
						DevicePluginImage: "image:tag",
					},
					ServiceAccountName: "",
				},
			}

			testNamespace = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{Name: namespaceName},
			}
			Expect(k8sClient.Create(ctx, testNamespace)).Should(Succeed())
		})

		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, onload)).Should(Succeed())
			Expect(k8sClient.Delete(ctx, testNamespace)).Should(Succeed())
		})

		It("should create an onload kind", func() {
			Expect(k8sClient.Create(ctx, onload)).To(BeNil())
		})

		It("should create a module", func() {
			createdModule := kmm.Module{}
			moduleName := types.NamespacedName{
				Name:      onload.Name + "-onload-module",
				Namespace: onload.Namespace,
			}

			By("creating an onload CR")
			Expect(k8sClient.Create(ctx, onload)).To(BeNil())

			By("checking for the existence of the module")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, moduleName, &createdModule)
				return err == nil
			}, timeout, pollingInterval).Should(BeTrue())

			By("checking the owner references of the module")
			Expect(createdModule.ObjectMeta.OwnerReferences).
				To(ContainElement(MatchFields(IgnoreExtras, Fields{
					"Name": Equal(onload.Name),
					"UID":  Equal(onload.UID),
				})))
		})

		It("should create a device plugin daemonset", func() {
			devicePlugin := appsv1.DaemonSet{}
			devicePluginName := types.NamespacedName{
				Name:      onload.Name + "-onload-device-plugin-ds",
				Namespace: onload.Namespace,
			}

			Expect(k8sClient.Create(ctx, onload)).To(BeNil())

			Eventually(func() bool {
				err := k8sClient.Get(ctx, devicePluginName, &devicePlugin)
				return err == nil
			}, timeout, pollingInterval).Should(BeTrue())

			Expect(devicePlugin.ObjectMeta.OwnerReferences).
				To(ContainElement(MatchFields(IgnoreExtras, Fields{
					"Name": Equal(onload.Name),
					"UID":  Equal(onload.UID),
				})))
		})

		It("should handle the update process", func() {
			By("creating the onload CR")
			Expect(k8sClient.Create(ctx, onload)).Should(Succeed())

			By("checking the operands")
			devicePlugin := appsv1.DaemonSet{}
			devicePluginName := types.NamespacedName{
				Name:      onload.Name + "-onload-device-plugin-ds",
				Namespace: onload.Namespace,
			}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, devicePluginName, &devicePlugin)
				return err == nil
			}, timeout, pollingInterval).Should(BeTrue())

			Expect(len(devicePlugin.Spec.Template.Spec.InitContainers)).To(Equal(1))
			Expect(devicePlugin.Spec.Template.Spec.InitContainers[0].Image).To(Equal(onload.Spec.Onload.UserImage))

			By("patching the onload CR definition")
			oldOnload := onload.DeepCopy()
			onload.Spec.Onload.UserImage = "image:tag2"
			onload.Spec.Onload.Version = "upgraded"
			Expect(len(onload.Spec.Onload.KernelMappings)).To(Equal(1))
			onload.Spec.Onload.KernelMappings[0].KernelModuleImage = "kernel-image:tag2"
			k8sClient.Patch(ctx, onload, client.MergeFrom(oldOnload))

			By("re-checking the operands")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, devicePluginName, &devicePlugin)
				if err != nil {
					return false
				}
				return devicePlugin.Spec.Template.Spec.InitContainers[0].Image == onload.Spec.Onload.UserImage
			}, timeout, pollingInterval).Should(BeTrue())

			Expect(len(devicePlugin.Spec.Template.Spec.InitContainers)).To(Equal(1))
			Expect(devicePlugin.Spec.Template.Spec.InitContainers[0].Image).To(Equal(onload.Spec.Onload.UserImage))
		})
	})
})
