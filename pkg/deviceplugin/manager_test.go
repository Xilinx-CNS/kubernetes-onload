// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package deviceplugin

import (
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
			path.Join(destLibDir, lib64path, "libonload.so")),
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
			"ContainerPath": Equal(path.Join(destDirBase, usrBinPath, "onload")),
			"HostPath":      Equal(path.Join(hostPathPrefix, usrBinPath, "onload")),
		}))))
	})
})
