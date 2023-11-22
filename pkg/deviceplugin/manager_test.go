// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package deviceplugin

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing command line options", func() {

	DescribeTable("It should set maxPodsPerNode correctly", func(num int) {
		man, err := NewNicManager(num, false)
		Expect(err).Should(Succeed())
		Expect(man.maxPodsPerNode).Should(Equal(num))
		Expect(len(man.devices)).Should(Equal(num))
	},
		Entry( /*It*/ "should set it to 0", 0),
		Entry( /*It*/ "should set it to 1", 1),
		Entry( /*It*/ "should set it to 100 (default)", 100),
		Entry( /*It*/ "should set it to 1000", 1000),
	)
})
