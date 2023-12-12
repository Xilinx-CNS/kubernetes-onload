// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package controllers

import (
	"context"
	"errors"
	"strconv"
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
	"k8s.io/utils/ptr"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	onloadv1alpha1 "github.com/Xilinx-CNS/kubernetes-onload/api/v1alpha1"
	mock_client "github.com/Xilinx-CNS/kubernetes-onload/mocks/client"

	kmm "github.com/kubernetes-sigs/kernel-module-management/api/v1beta1"
)

var _ = Describe("Testing createModule function", func() {
	var onload *onloadv1alpha1.Onload
	var exampleKernelMapper kernelMapperFn

	BeforeEach(func() {
		onload = &onloadv1alpha1.Onload{}

		exampleKernelMapper = func(spec onloadv1alpha1.OnloadKernelMapping) *kmm.KernelMapping {
			return &kmm.KernelMapping{
				Regexp:         spec.Regexp,
				ContainerImage: spec.KernelModuleImage,
			}
		}
	})

	It("Should work with a valid onload CR", func() {
		module, err := createModule(onload, "example", "example.ko", "old-example.ko", exampleKernelMapper)
		Expect(err).Should(Succeed())

		Expect(module.Spec.ModuleLoader.Container.Modprobe.ModuleName).To(Equal("example.ko"))
		Expect(module.Spec.ModuleLoader.Container.InTreeModuleToRemove).To(Equal("old-example.ko"))
	})

	It("Should have the correct number of kernel mappings", func() {
		for i := 0; i < 10; i++ {
			onload.Spec.Onload.KernelMappings = append(
				onload.Spec.Onload.KernelMappings,
				onloadv1alpha1.OnloadKernelMapping{},
			)
		}
		module, err := createModule(onload, "example", "example", "example", exampleKernelMapper)
		Expect(err).Should(Succeed())

		Expect(len(module.Spec.ModuleLoader.Container.KernelMappings)).
			To(Equal(len(onload.Spec.Onload.KernelMappings)))
	})
})

var _ = Describe("Testing onloadUsesSFC predicate", func() {
	var (
		onloadWithSFC    onloadv1alpha1.Onload
		onloadWithoutSFC onloadv1alpha1.Onload
	)

	BeforeEach(func() {
		onloadWithoutSFC = onloadv1alpha1.Onload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "name",
				Namespace: "namespace",
			},
			Spec: onloadv1alpha1.Spec{
				Selector: map[string]string{
					"key": "value",
				},

				Onload: onloadv1alpha1.OnloadSpec{
					KernelMappings: []onloadv1alpha1.OnloadKernelMapping{
						{
							KernelModuleImage: "",
							Regexp:            "",
						},
					},
					UserImage: "image:tag",
					Version:   "foo",
				},
			},
		}

		onloadWithoutSFC.DeepCopyInto(&onloadWithSFC)
		onloadWithSFC.Spec.Onload.KernelMappings[0].SFC = &onloadv1alpha1.SFCSpec{}
	})

	It("Should find SFC", func() {
		Expect(onloadUsesSFC(&onloadWithSFC)).To(BeTrue())
	})

	It("Should not find SFC", func() {
		Expect(onloadUsesSFC(&onloadWithoutSFC)).To(BeFalse())
	})
})

var _ = Describe("Testing kernelmapping conversion functions", func() {
	var (
		onloadKmap  onloadv1alpha1.OnloadKernelMapping
		onloadBuild onloadv1alpha1.OnloadKernelBuild
	)

	BeforeEach(func() {
		onloadKmap = onloadv1alpha1.OnloadKernelMapping{
			KernelModuleImage: "image:label",
			Regexp:            "",
		}

		onloadBuild = onloadv1alpha1.OnloadKernelBuild{
			DockerfileConfigMap: &corev1.LocalObjectReference{Name: "foo"},
		}
	})

	It("should map as expected for onloadKernelMapper", func() {
		Expect(onloadKernelMapper(onloadKmap)).Should(PointTo(
			MatchFields(IgnoreExtras, Fields{
				"Regexp":         Equal(onloadKmap.Regexp),
				"ContainerImage": Equal(onloadKmap.KernelModuleImage),
			})),
		)
	})

	It("shouldn't map anything for sfcKernelMapper with an empty sfc field", func() {
		Expect(sfcKernelMapper(onloadKmap)).Should(BeNil())
	})

	It("should map as expected for sfcKernelMapper with a set sfc field", func() {
		onloadKmap.SFC = &onloadv1alpha1.SFCSpec{}
		Expect(sfcKernelMapper(onloadKmap)).Should(PointTo(
			MatchFields(IgnoreExtras, Fields{
				"Regexp":         Equal(onloadKmap.Regexp),
				"ContainerImage": Equal(onloadKmap.KernelModuleImage),
			})),
		)
	})

	It("should map the build parameters in onloadKernelMapper", func() {
		onloadKmap.Build = &onloadBuild

		kmmKmap := onloadKernelMapper(onloadKmap)

		Expect(kmmKmap).ShouldNot(BeNil())
		Expect(kmmKmap.Build).Should(PointTo(MatchFields(IgnoreExtras, Fields{
			"DockerfileConfigMap": Equal(onloadKmap.Build.DockerfileConfigMap),
		})))
	})

	It("should map the build args in onloadKernelMapper", func() {

		buildArgs := []onloadv1alpha1.BuildArg{
			{Name: "A", Value: "1"},
			{Name: "B", Value: "2"},
			{Name: "C", Value: "3"},
		}
		onloadBuild.BuildArgs = buildArgs
		onloadKmap.Build = &onloadBuild

		kmmKmap := onloadKernelMapper(onloadKmap)

		idFn := func(index int, _ interface{}) string {
			return strconv.Itoa(index)
		}

		Expect(kmmKmap).ShouldNot(BeNil())
		Expect(kmmKmap.Build).Should(PointTo(MatchFields(IgnoreExtras, Fields{
			"DockerfileConfigMap": Equal(onloadKmap.Build.DockerfileConfigMap),
			"BuildArgs": MatchAllElementsWithIndex(idFn, Elements{
				"0": Equal(kmm.BuildArg(buildArgs[0])),
				"1": Equal(kmm.BuildArg(buildArgs[1])),
				"2": Equal(kmm.BuildArg(buildArgs[2])),
			}),
		})))
	})

	It("shouldn't map the build parameters in sfcKernelMapper", func() {
		onloadKmap.Build = &onloadBuild
		onloadKmap.SFC = &onloadv1alpha1.SFCSpec{}
		kmmKmap := sfcKernelMapper(onloadKmap)

		Expect(kmmKmap).ShouldNot(BeNil())
		Expect(kmmKmap.Build).Should(BeNil())
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
						KernelMappings: []onloadv1alpha1.OnloadKernelMapping{
							{
								KernelModuleImage: "",
								Regexp:            "",
							},
						},
						UserImage: "image:tag",
						Version:   "foo",
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

		It("should label Nodes with Onload kmm labels", func() {
			pods := corev1.PodList{}

			// Listing nodes
			listNodesCall := mockClient.EXPECT().
				List(gomock.Any(), &corev1.NodeList{}, gomock.Any()).
				SetArg(1, nodes).
				Return(nil).
				Times(1)

			// Listing pods
			listPodsCalls := mockClient.EXPECT().
				List(gomock.Any(), &corev1.PodList{}, gomock.Any()).
				SetArg(1, pods).
				Return(nil).
				Times(len(nodes.Items)).
				After(listNodesCall)

			// Patching nodes to add the label
			mockClient.EXPECT().
				Patch(gomock.Any(), &nodes.Items[0], gomock.Any()).
				Return(nil).
				Times(1).
				After(listPodsCalls)

			Expect(r.addKmmLabelsToNodes(ctx, &onload)).Should(Equal(&ctrl.Result{Requeue: true}))
		})

		It("should label Nodes with Onload and SFC kmm labels", func() {
			onload.Spec.Onload.KernelMappings[0].SFC = &onloadv1alpha1.SFCSpec{}

			pods := corev1.PodList{}

			// Listing nodes
			listNodesCall := mockClient.EXPECT().
				List(gomock.Any(), &corev1.NodeList{}, gomock.Any()).
				SetArg(1, nodes).
				Return(nil).
				Times(1)

			// This can be either "list nodes" or "patch node"
			previousCall := listNodesCall

			modules := []string{"onload", "sfc"}
			for range modules {
				// Listing pods
				listPodsCalls := mockClient.EXPECT().
					List(gomock.Any(), &corev1.PodList{}, gomock.Any()).
					SetArg(1, pods).
					Return(nil).
					Times(len(nodes.Items)).
					After(previousCall)

				// Patching nodes to add the label
				previousCall = mockClient.EXPECT().
					Patch(gomock.Any(), &nodes.Items[0], gomock.Any()).
					Return(nil).
					Times(1).
					After(listPodsCalls)
			}

			Expect(r.addKmmLabelsToNodes(ctx, &onload)).Should(Equal(&ctrl.Result{Requeue: true}))
		})

		It("should not label Nodes with SFC kmm label while Onload is upgrading", func() {
			// Request the SFC module kind with this Onload CR
			onload.Spec.Onload.KernelMappings[0].SFC = &onloadv1alpha1.SFCSpec{}

			// Label the node with the KMM Onload label that does not match
			// the Onload spec's version "foo" to mimic the upgrade scenario
			nodes.Items[0].Labels[kmmOnloadLabelName(onload.Name, onload.Namespace)] = "bar"

			// Listing nodes thrice: once for the "addition" step,
			// twice for the "removal" step with removeStaleLabels().
			mockClient.EXPECT().
				List(gomock.Any(), &corev1.NodeList{}, gomock.Any()).
				SetArg(1, nodes).
				Return(nil).
				Times(3)

			res, err := r.addKmmLabelsToNodes(ctx, &onload)

			// No reconciliation needed
			Expect(res).Should(BeNil())

			// No errors either
			Expect(err).Should(BeNil())
		})

		It("should label nodes with Onload label", func() {

			// Add the kmm label to the second node so that the onload label
			// will be added
			nodes.Items[0].Labels[kmmOnloadLabelName(onload.Name, onload.Namespace)] = onload.Spec.Onload.Version

			// Add a new node to the list with the kmm label, but with the wrong
			// value. This node should not be patched.
			nodes.Items = append(nodes.Items, corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"key": "value",
						kmmOnloadLabelName(onload.Name, onload.Namespace): "bar",
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
			labelKey := kmmOnloadLabelName(onload.Name, onload.Namespace)

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
				Times(2)

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

		It("should create one Onload module", func() {
			createdModule := kmm.Module{}
			moduleName := types.NamespacedName{
				Name:      onload.Name + "-onload-module",
				Namespace: onload.Namespace,
			}

			By("creating an onload CR")
			Expect(k8sClient.Create(ctx, onload)).To(BeNil())

			By("checking for the existence of the Onload module")
			Eventually(func() bool {
				err := k8sClient.Get(ctx, moduleName, &createdModule)
				return err == nil
			}, timeout, pollingInterval).Should(BeTrue())

			By("checking for existence of only one module")
			Eventually(func() int {
				var moduleList kmm.ModuleList
				err := k8sClient.List(ctx, &moduleList, client.InNamespace(onload.Namespace))
				if err == nil {
					return len(moduleList.Items)
				} else {
					return -1
				}
			}, timeout, pollingInterval).Should(Equal(1))

			By("checking the owner references of the module")
			Expect(createdModule.ObjectMeta.OwnerReferences).
				To(ContainElement(MatchFields(IgnoreExtras, Fields{
					"Name": Equal(onload.Name),
					"UID":  Equal(onload.UID),
				})))
		})

		It("should create one Onload and one SFC module", func() {
			createdModule := kmm.Module{}

			By("creating an onload CR")
			onload.Spec.Onload.KernelMappings[0].SFC = &onloadv1alpha1.SFCSpec{}
			Expect(k8sClient.Create(ctx, onload)).To(BeNil())

			By("checking for the existence of each module")
			moduleName := types.NamespacedName{
				Name:      onload.Name + "-onload-module",
				Namespace: onload.Namespace,
			}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, moduleName, &createdModule)
				return err == nil
			}, timeout, pollingInterval).Should(BeTrue())

			moduleName = types.NamespacedName{
				Name:      onload.Name + "-sfc-module",
				Namespace: onload.Namespace,
			}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, moduleName, &createdModule)
				return err == nil
			}, timeout, pollingInterval).Should(BeTrue())

			By("checking for existence of only two modules")
			Eventually(func() int {
				var moduleList kmm.ModuleList
				err := k8sClient.List(ctx, &moduleList, client.InNamespace(onload.Namespace))
				if err == nil {
					return len(moduleList.Items)
				} else {
					return -1
				}
			}, timeout, pollingInterval).Should(Equal(2))

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

		// Test all four combinations of Onload CR upgrade: with/without SFC before/after
		DescribeTable("Onload upgrade with and without SFC",
			func(sfcBefore *onloadv1alpha1.SFCSpec, sfcAfter *onloadv1alpha1.SFCSpec) {
				By("creating the onload CR")

				// The initial Onload CR doesn't request SFC support
				onload.Spec.Onload.KernelMappings[0].SFC = sfcBefore
				Expect(k8sClient.Create(ctx, onload)).Should(Succeed())

				By("checking Onload Device Plugin")
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

				// The user might have requested SFC support during upgrade
				onload.Spec.Onload.KernelMappings[0].SFC = sfcAfter

				Expect(len(onload.Spec.Onload.KernelMappings)).To(Equal(1))

				onload.Spec.Onload.KernelMappings[0].KernelModuleImage = "kernel-image:tag2"
				Expect(k8sClient.Patch(ctx, onload, client.MergeFrom(oldOnload))).Should(Succeed())

				By("re-checking Onload Device Plugin")
				Eventually(func() bool {
					err := k8sClient.Get(ctx, devicePluginName, &devicePlugin)
					if err != nil {
						return false
					}
					return devicePlugin.Spec.Template.Spec.InitContainers[0].Image == onload.Spec.Onload.UserImage
				}, timeout, pollingInterval).Should(BeTrue())

				Expect(len(devicePlugin.Spec.Template.Spec.InitContainers)).To(Equal(1))

				By("checking the SFC module")
				sfcModule := kmm.Module{}
				moduleName := types.NamespacedName{
					Name:      onload.Name + "-sfc-module",
					Namespace: onload.Namespace,
				}

				if sfcAfter != nil {
					Eventually(func() bool {
						err := k8sClient.Get(ctx, moduleName, &sfcModule)
						if err != nil {
							return false
						}
						return sfcModule.Spec.ModuleLoader.Container.KernelMappings[0].ContainerImage ==
							onload.Spec.Onload.KernelMappings[0].KernelModuleImage
					}, timeout, pollingInterval).Should(BeTrue())

					Expect(sfcModule.Spec.ModuleLoader.Container.Modprobe.ModuleName).
						Should(Equal("sfc"))
				} else {
					Eventually(func() bool {
						err := k8sClient.Get(ctx, moduleName, &sfcModule)
						return apierrors.IsNotFound(err)
					}, timeout, pollingInterval).Should(BeTrue())
				}
			},
			Entry("Upgrade Onload without SFC", nil, nil),
			Entry("Add SFC during Onload upgrade", nil, &onloadv1alpha1.SFCSpec{}),
			Entry("Remove SFC during Onload upgrade", &onloadv1alpha1.SFCSpec{}, nil),
			Entry("Upgrade Onload with SFC", &onloadv1alpha1.SFCSpec{}, &onloadv1alpha1.SFCSpec{}),
		)

		DescribeTable("Testing Device Plugin options",
			func(dev *onloadv1alpha1.DevicePluginSpec, args string) {
				devicePlugin := appsv1.DaemonSet{}
				devicePluginName := types.NamespacedName{
					Name:      onload.Name + "-onload-device-plugin-ds",
					Namespace: onload.Namespace,
				}

				if dev != nil {
					dev.DevicePluginImage = "image:tag"
					onload.Spec.DevicePlugin = *dev
				}

				Expect(k8sClient.Create(ctx, onload)).To(BeNil())

				Eventually(func() bool {
					err := k8sClient.Get(ctx, devicePluginName, &devicePlugin)
					return err == nil
				}, timeout, pollingInterval).Should(BeTrue())

				if args != "" {
					Expect(devicePlugin.Spec.Template.Spec.Containers).Should(
						ContainElement(MatchFields(IgnoreExtras, Fields{
							"Args": ContainElement(args),
						})),
					)
				}

			},
			Entry( /*It*/ "shouldn't add anything when empty", nil, ""),
			Entry( /*It*/ "should pass the value of maxPodsPerNode through",
				&onloadv1alpha1.DevicePluginSpec{MaxPodsPerNode: ptr.To(1)},
				"-maxPods=1",
			),
			Entry( /*It*/ "should pass the value of setPreload through",
				&onloadv1alpha1.DevicePluginSpec{SetPreload: ptr.To(false)},
				"-setPreload=false",
			),
			Entry( /*It*/ "should pass the value of mountOnload through",
				&onloadv1alpha1.DevicePluginSpec{MountOnload: ptr.To(false)},
				"-mountOnload=false",
			),
			Entry( /*It*/ "should pass the value of hostOnloadPath through",
				&onloadv1alpha1.DevicePluginSpec{HostOnloadPath: ptr.To("foo")},
				"-hostOnloadPath=foo",
			),
			Entry( /*It*/ "should pass the value of baseMountPath through",
				&onloadv1alpha1.DevicePluginSpec{BaseMountPath: ptr.To("bar")},
				"-baseMountPath=bar",
			),
			Entry( /*It*/ "should pass the value of binMountPath through",
				&onloadv1alpha1.DevicePluginSpec{BinMountPath: ptr.To("baz")},
				"-binMountPath=baz",
			),
			Entry( /*It*/ "should pass the value of libMountPath through",
				&onloadv1alpha1.DevicePluginSpec{LibMountPath: ptr.To("qux")},
				"-libMountPath=qux",
			),
		)
	})
})

// Please note that each of these tests will spin-up a new envtest cluster
// before each test and destroy it after. This has quite a heavy time cost, so
// if possible please try to test functionality in a "lighter" way.
// Running tests on an Intel(R) Xeon(R) Silver 4216 CPU @ 2.10GHz showed that
// adding an empty test would increase the time taken to run `make test` by ~5s
var _ = Describe("Recovery from a non-clean state", func() {

	// Shadowing variables that relate to the env test cluster.
	var (
		k8sManager manager.Manager
		k8sClient  client.Client
		testEnv    *envtest.Environment
		ctx        context.Context
		cancel     context.CancelFunc
	)

	// General definitions for tests
	var (
		onload           onloadv1alpha1.Onload
		devicePlugin     appsv1.DaemonSet
		devicePluginName types.NamespacedName
		module           kmm.Module
	)

	BeforeEach(func() {

		ctx, cancel, testEnv, k8sClient, k8sManager = createEnvTestCluster()

		onload = onloadv1alpha1.Onload{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "onload-test",
				Namespace: "default",
			},
			Spec: onloadv1alpha1.Spec{
				Selector: map[string]string{
					"key": "value",
				},
				Onload: onloadv1alpha1.OnloadSpec{
					KernelMappings: []onloadv1alpha1.OnloadKernelMapping{
						{
							KernelModuleImage: "",
							Regexp:            "",
						},
					},
					UserImage: "image:tag",
					Version:   "old",
				},
				DevicePlugin: onloadv1alpha1.DevicePluginSpec{
					DevicePluginImage: "image:tag",
				},
				ServiceAccountName: "",
			},
		}

		devicePluginName = types.NamespacedName{
			Name:      onload.Name + "-onload-device-plugin-ds",
			Namespace: onload.Namespace,
		}

		dsLabels := map[string]string{"onload.amd.com/name": devicePluginName.Name}
		devicePlugin = appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      devicePluginName.Name,
				Namespace: devicePluginName.Namespace,
				Labels:    map[string]string{onloadVersionLabel: onload.Spec.Onload.Version},
			},
			Spec: appsv1.DaemonSetSpec{
				Selector: &metav1.LabelSelector{MatchLabels: dsLabels},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: dsLabels,
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "foo", Image: "bar"},
						},
						InitContainers: []corev1.Container{
							{
								Name: "init", Image: onload.Spec.Onload.UserImage,
							},
						},
					},
				},
			},
		}

		module = kmm.Module{
			ObjectMeta: metav1.ObjectMeta{
				Name:      onload.Name + onloadModuleNameSuffix,
				Namespace: onload.Namespace,
			},
			Spec: kmm.ModuleSpec{
				Selector: onload.Spec.Selector,
				ModuleLoader: kmm.ModuleLoaderSpec{
					Container: kmm.ModuleLoaderContainerSpec{
						KernelMappings: []kmm.KernelMapping{{}},
					},
				},
			},
		}
	})

	AfterEach(func() {
		cancel()
		err := testEnv.Stop()
		Expect(err).NotTo(HaveOccurred())
	})

	It("shouldn't remove existing operands", func() {
		Expect(k8sClient.Create(ctx, &onload)).Should(Succeed())

		Expect(ctrl.SetControllerReference(&onload, &devicePlugin, testEnv.Scheme)).Should(Succeed())
		Expect(k8sClient.Create(ctx, &devicePlugin)).Should(Succeed())

		startReconciler(k8sManager, ctx)

		// Create a new object so we can compare the old and new versions of the
		// object
		newDevicePlugin := appsv1.DaemonSet{}

		Eventually(func() bool {
			err := k8sClient.Get(ctx, devicePluginName, &newDevicePlugin)
			return err == nil
		}, timeout, pollingInterval).Should(BeTrue())

		Expect(newDevicePlugin.UID).To(Equal(devicePlugin.UID))
	})

	// These test are restricted to testing the upgrading of the operands. Tests
	// that use Nodes are in a subsequent section.
	It("should continue upgrade process (device plugin)", func() {

		// Fake the upgrade, the device plugin is still using the old values
		onload.Spec.Onload.Version = "new"
		onload.Spec.Onload.UserImage = "image:tag-newer"

		Expect(onload.Spec.Onload.Version).ShouldNot(Equal(devicePlugin.Labels[onloadVersionLabel]))

		Expect(k8sClient.Create(ctx, &onload)).Should(Succeed())

		Expect(ctrl.SetControllerReference(&onload, &devicePlugin, testEnv.Scheme)).Should(Succeed())
		Expect(k8sClient.Create(ctx, &devicePlugin)).Should(Succeed())

		startReconciler(k8sManager, ctx)

		// Create a new object so we can compare the old and new versions of the
		// object
		newDevicePlugin := appsv1.DaemonSet{}

		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&devicePlugin), &newDevicePlugin)).Should(Succeed())
			g.Expect(newDevicePlugin.UID).To(Equal(devicePlugin.UID))
			g.Expect(newDevicePlugin.Labels[onloadVersionLabel]).ShouldNot(Equal(devicePlugin.Labels[onloadVersionLabel]))
			g.Expect(newDevicePlugin.Labels[onloadVersionLabel]).Should(Equal(onload.Spec.Onload.Version))
		}, timeout, pollingInterval).Should(Succeed())

	})

	It("should continue upgrade process (module)", func() {
		// Fake the upgrade, the operand is still using the old values
		onload.Spec.Onload.Version = "new"
		onload.Spec.Onload.UserImage = "image:tag-newer"

		Expect(onload.Spec.Onload.Version).ShouldNot(Equal(module.Spec.ModuleLoader.Container.Version))

		Expect(k8sClient.Create(ctx, &onload)).Should(Succeed())

		Expect(ctrl.SetControllerReference(&onload, &module, testEnv.Scheme)).Should(Succeed())
		Expect(k8sClient.Create(ctx, &module)).Should(Succeed())

		startReconciler(k8sManager, ctx)

		// Create a new object so we can compare the old and new versions of the
		// object
		newModule := kmm.Module{}

		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&module), &newModule)).Should(Succeed())
			g.Expect(newModule.UID).To(Equal(module.UID))
			g.Expect(newModule.Spec.ModuleLoader.Container.Version).ShouldNot(Equal(module.Spec.ModuleLoader.Container.Version))
			g.Expect(newModule.Spec.ModuleLoader.Container.Version).Should(Equal(onload.Spec.Onload.Version))
		}, timeout, pollingInterval).Should(Succeed())

	})

	Describe("Node-based tests", func() {

		var (
			node           corev1.Node
			kmmOnloadLabel string
			dpOnloadLabel  string
		)

		BeforeEach(func() {
			node = corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					Labels: map[string]string{
						"key": "value",
					},
				},
			}

			kmmOnloadLabel = kmmOnloadLabelName(onload.Name, onload.Namespace)
			dpOnloadLabel = onloadLabelName(onload.Name, onload.Namespace)
		})

		It("should add the initial labels", func() {

			Expect(k8sClient.Create(ctx, &node)).Should(Succeed())

			Expect(k8sClient.Create(ctx, &onload)).Should(Succeed())

			startReconciler(k8sManager, ctx)

			newNode := corev1.Node{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&node), &newNode)).Should(Succeed())
				g.Expect(newNode.Labels[kmmOnloadLabel]).Should(Equal(onload.Spec.Onload.Version))
				g.Expect(newNode.Labels[dpOnloadLabel]).Should(Equal(onload.Spec.Onload.Version))
			}, timeout, pollingInterval).Should(Succeed())
		})

		// Want to test 3 different values for each label/version: unset, old and new
		DescribeTable("controller starting at different points during upgrade",
			func(label1, label2 string) {
				if label1 != "" {
					node.Labels[kmmOnloadLabel] = label1
				}

				if label2 != "" {
					node.Labels[dpOnloadLabel] = label2
				}

				Expect(k8sClient.Create(ctx, &node)).Should(Succeed())

				onload.Spec.Onload.Version = "new"
				onload.Spec.Onload.UserImage = "image:tag-newer"
				Expect(k8sClient.Create(ctx, &onload)).Should(Succeed())

				if label1 != "" {
					module.Spec.ModuleLoader.Container.Version = label1
				}
				Expect(ctrl.SetControllerReference(&onload, &module, testEnv.Scheme)).Should(Succeed())

				if label2 != "" {
					devicePlugin.Labels[onloadVersionLabel] = label2
				}
				Expect(ctrl.SetControllerReference(&onload, &devicePlugin, testEnv.Scheme)).Should(Succeed())

				Expect(k8sClient.Create(ctx, &module)).Should(Succeed())
				Expect(k8sClient.Create(ctx, &devicePlugin)).Should(Succeed())

				startReconciler(k8sManager, ctx)

				newNode := corev1.Node{}
				Eventually(func(g Gomega) {
					g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(&node), &newNode)).Should(Succeed())
					g.Expect(newNode.Labels[kmmOnloadLabel]).Should(Equal(onload.Spec.Onload.Version))
					g.Expect(newNode.Labels[dpOnloadLabel]).Should(Equal(onload.Spec.Onload.Version))
				}, timeout, pollingInterval).Should(Succeed())
			},
			Entry("00", "", ""),
			Entry("01", "", "old"), // I don't think this should be able to happen
			Entry("02", "", "new"),
			Entry("10", "old", ""),
			Entry("11", "old", "old"),
			Entry("12", "old", "new"),
			Entry("20", "new", ""),
			Entry("21", "new", "old"), // I don't think this should be able to happen
			Entry("22", "new", "new"),
		)
	})

})
