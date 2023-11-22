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

	DescribeTable("It should set maxPodsPerNode correctly", func(num int) {
		man, err := NewNicManager(num, true, false, false)
		Expect(err).Should(Succeed())
		Expect(man.maxPodsPerNode).Should(Equal(num))
		Expect(len(man.devices)).Should(Equal(num))
	},
		Entry( /*It*/ "should set it to 0", 0),
		Entry( /*It*/ "should set it to 1", 1),
		Entry( /*It*/ "should set it to 100 (default)", 100),
		Entry( /*It*/ "should set it to 1000", 1000),
	)

	It("should set LD_PRELOAD when appropriate", func() {
		man, err := NewNicManager(100, true, false, false)
		Expect(err).Should(Succeed())
		Expect(man.envs).Should(HaveKeyWithValue(
			"LD_PRELOAD",
			path.Join(destLibDir, lib64path, "libonload.so")),
		)
	})

	It("should not set LD_PRELOAD if empty", func() {
		man, err := NewNicManager(100, false, false, false)
		Expect(err).Should(Succeed())
		Expect(man.envs).ShouldNot(HaveKey("LD_PRELOAD"))
	})

	It("should mount onload when appropriate", func() {
		man, err := NewNicManager(100, false, true, false)
		Expect(err).Should(Succeed())

		Expect(man.mounts).Should(ContainElement(PointTo(MatchFields(IgnoreExtras, Fields{
			"ContainerPath": Equal(path.Join(destDirBase, usrBinPath, "onload")),
			"HostPath":      Equal(path.Join(hostPathPrefix, usrBinPath, "onload")),
		}))))
	})
})
