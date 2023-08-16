package controllers

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	onloadv1alpha1 "github.com/Xilinx-CNS/kubernetes-onload/api/v1alpha1"

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

	It("Should fail if a build parameter was supplied", func() {
		onload.Spec = onloadv1alpha1.Spec{
			Onload: onloadv1alpha1.OnloadSpec{
				KernelMappings: []onloadv1alpha1.OnloadKernelMapping{
					{
						Regexp:            "example",
						KernelModuleImage: "example",
						Build:             &onloadv1alpha1.OnloadKernelBuildSpec{},
					},
				},
			},
		}
		_, err := createModule(onload, "example", "example")
		Expect(err).To(HaveOccurred())
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
					Selector: map[string]string{"": ""},
					Onload: onloadv1alpha1.OnloadSpec{
						KernelMappings: []onloadv1alpha1.OnloadKernelMapping{
							{
								KernelModuleImage: "",
								Regexp:            "",
							},
						},
						UserImage: "",
						Version:   "",
					},
					DevicePlugin: onloadv1alpha1.DevicePluginSpec{
						DevicePluginImage: "",
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
	})
})
