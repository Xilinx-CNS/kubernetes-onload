# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.

# Base image for the container where to build the Onload and SFC kernel modules.
DTK_AUTO ?= $$(oc adm release info --image-for=driver-toolkit)

ONLOAD_LOCATION := https://github.com/Xilinx-CNS/onload/archive/refs/tags/v8.1.0.tar.gz

KERNEL_FULL_VERSION ?= $$(./dtk-kernel-version $(DTK_AUTO))

# The Onload module and user images are consumed by the Onload CR,
# which is defined at config/samples/onload_v1alpha1_onload.yaml.
ONLOAD_MODULE_IMAGE ?= image-registry.openshift-image-registry.svc:5000/onload-clusterlocal/onload-module:v8.1.0-$(KERNEL_FULL_VERSION)
ONLOAD_USER_IMAGE ?= image-registry.openshift-image-registry.svc:5000/onload-clusterlocal/onload-user:v8.1.0

all: onload-module-push onload-user-push

onload-module-push: onload-module-build
	docker push $(ONLOAD_MODULE_IMAGE)

onload-user-push: onload-user-build
	docker push $(ONLOAD_USER_IMAGE)

onload-module-build:
	docker build \
		-t $(ONLOAD_MODULE_IMAGE) \
		--build-arg DTK_AUTO=$(DTK_AUTO) \
		--build-arg ONLOAD_LOCATION=$(ONLOAD_LOCATION) \
		--build-arg KERNEL_FULL_VERSION=$(KERNEL_FULL_VERSION) \
		onload-module/

onload-user-build:
	docker build \
		-t $(ONLOAD_USER_IMAGE) \
		--build-arg ONLOAD_LOCATION=$(ONLOAD_LOCATION) \
		onload-user/

.PHONY: all onload-module-push onload-user-push onload-module-build onload-user-build
