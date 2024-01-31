// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package deviceplugin

import (
	"os"
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Testing command line options", func() {
	var config NicManagerConfig

	BeforeEach(func() {
		config = DefaultConfig
		config.NeedNic = false
	})

	It("should mount onload libraries", func() {
		tmpDir, err := os.MkdirTemp("", "tmp-onload-libs")
		Expect(err).Should(Succeed())
		defer os.RemoveAll(tmpDir)

		Expect(os.MkdirAll(path.Join(tmpDir, hostLib64path), os.ModePerm)).Should(Succeed())
		file, err := os.Create(path.Join(tmpDir, hostLib64path, "libonload.so"))
		Expect(err).Should(Succeed())
		defer file.Close()

		config.HostPathPrefix = tmpDir

		man, err := NewNicManager(config)
		Expect(err).Should(Succeed())
		Expect(man.mounts).Should(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
			"HostPath":      Equal(path.Join(tmpDir, hostLib64path, "libonload.so")),
			"ContainerPath": Equal(path.Join(config.BaseMountPath, config.LibMountPath, "libonload.so")),
		}))))
	})

	DescribeTable("It should set maxPodsPerNode correctly", func(num int) {
		config.MaxPodsPerNode = num
		man, err := NewNicManager(config)
		Expect(err).Should(Succeed())
		Expect(man.config.MaxPodsPerNode).Should(Equal(num))
		Expect(len(man.devices)).Should(Equal(num))
	},
		Entry( /*It*/ "should set it to 0", 0),
		Entry( /*It*/ "should set it to 1", 1),
		Entry( /*It*/ "should set it to 100 (default)", 100),
		Entry( /*It*/ "should set it to 1000", 1000),
	)

	It("should set LD_PRELOAD when appropriate", func() {
		config.SetPreload = true
		man, err := NewNicManager(config)
		Expect(err).Should(Succeed())
		Expect(man.envs).Should(HaveKeyWithValue(
			"LD_PRELOAD",
			path.Join("/opt/onload", hostLib64path, "libonload.so")),
		)
	})

	It("should not set LD_PRELOAD if empty", func() {
		config.SetPreload = false
		man, err := NewNicManager(config)
		Expect(err).Should(Succeed())
		Expect(man.envs).ShouldNot(HaveKey("LD_PRELOAD"))
	})

	It("should mount onload when appropriate", func() {
		config.SetPreload = false
		config.MountOnload = true
		man, err := NewNicManager(config)
		Expect(err).Should(Succeed())

		Expect(man.mounts).Should(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
			"ContainerPath": Equal(path.Join(config.BaseMountPath, config.BinMountPath, "onload")),
			"HostPath":      Equal(path.Join(config.HostPathPrefix, hostUsrBinPath, "onload")),
		}))))
	})

	It("should fail if both preload and mountOnload flags are set", func() {
		config.SetPreload = true
		config.MountOnload = true
		_, err := NewNicManager(config)
		Expect(err).ShouldNot(Succeed())
	})
})

var _ = Describe("Testing topology information", func() {
	var manager *NicManager

	BeforeEach(func() {
		manager = &NicManager{
			config: NicManagerConfig{
				MaxPodsPerNode: 1,
			},
		}
	})

	It("shouldn't provide numa information when none are specified", func() {
		manager.interfaces = []nic{
			{name: "A", numa: -1},
			{name: "B", numa: -1},
			{name: "C", numa: -1},
		}
		manager.initDevices()
		Expect(len(manager.devices[0].Topology.GetNodes())).To(Equal(0))
	})

	It("should describe numa information when a single node is present", func() {
		manager.interfaces = []nic{
			{name: "A", numa: 1},
			{name: "B", numa: -1},
			{name: "C", numa: -1},
		}
		manager.initDevices()
		Expect(len(manager.devices[0].Topology.GetNodes())).To(Equal(1))
		Expect(manager.devices[0].Topology.GetNodes()[0].ID).To(Equal(int64(1)))
	})

	It("should describe numa information when multiple nodes are present", func() {
		manager.interfaces = []nic{
			{name: "A", numa: 1},
			{name: "B", numa: 2},
			{name: "C", numa: -1},
		}
		manager.initDevices()
		Expect(len(manager.devices[0].Topology.GetNodes())).To(Equal(2))
		Expect(manager.devices[0].Topology.GetNodes()[0].ID).To(Equal(int64(1)))
		Expect(manager.devices[0].Topology.GetNodes()[1].ID).To(Equal(int64(2)))
	})

	It("shouldn't provide duplicate numa information", func() {
		manager.interfaces = []nic{
			{name: "A", numa: 1},
			{name: "B", numa: 1},
			{name: "C", numa: -1},
		}
		manager.initDevices()
		Expect(len(manager.devices[0].Topology.GetNodes())).To(Equal(1))
		Expect(manager.devices[0].Topology.GetNodes()[0].ID).To(Equal(int64(1)))
	})
})
