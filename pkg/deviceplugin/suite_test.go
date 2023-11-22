// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
package deviceplugin

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"testing"
)

func TestOnload(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Onload Device Plugin Suite")
}
