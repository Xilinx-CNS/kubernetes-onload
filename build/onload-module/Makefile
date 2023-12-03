# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
.PHONY: all onload-module-dtk dtk_vars onload_vars

OCP_VERSION :=
DTK_AUTO ?= $$(oc adm release info $(OCP_VERSION) --image-for=driver-toolkit)
KERNEL_VERSION ?= $$(./dtk-release-vars --kernel-version $(DTK_AUTO))

ONLOAD_SOURCE_IMAGE ?= onload-source
ONLOAD_SOURCE_IMAGE_REGISTRY ?= localhost
ONLOAD_SOURCE_IMAGE_TAG ?= latest
ONLOAD_SOURCE ?= $(ONLOAD_SOURCE_IMAGE_REGISTRY)/$(ONLOAD_SOURCE_IMAGE):$(ONLOAD_SOURCE_IMAGE_TAG)
ONLOAD_VERSION ?= $$(./onload-source-vars --onload-version $(ONLOAD_SOURCE))

ONLOAD_MODULE_IMAGE_REGISTRY ?= localhost
ONLOAD_MODULE_IMAGE ?= onload-module
ONLOAD_MODULE ?= $(ONLOAD_MODULE_IMAGE_REGISTRY)/$(ONLOAD_MODULE_IMAGE):$(ONLOAD_VERSION)-$(KERNEL_VERSION)

all: onload-module-dtk

dtk_vars:
	@echo DTK_AUTO=$(DTK_AUTO)
	@echo KERNEL_VERSION=$(KERNEL_VERSION)

onload_vars:
	@echo ONLOAD_VERSION=$(ONLOAD_VERSION)
	@echo ONLOAD_MODULE=$(ONLOAD_MODULE)

onload-module-dtk: dtk_vars onload_vars
	docker build \
		-t $(ONLOAD_MODULE) \
		--build-arg DTK_AUTO=$(DTK_AUTO) \
		--build-arg ONLOAD_SOURCE=$(ONLOAD_SOURCE) \
		--build-arg KERNEL_FULL_VERSION=$(KERNEL_VERSION) \
		../../config/samples/onload-module/dtk-ubi/
