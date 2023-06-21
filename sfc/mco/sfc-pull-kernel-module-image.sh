#!/bin/bash
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.

if podman image exists image-registry.openshift-image-registry.svc:5000/openshift-kmm/sfc-module:$(uname -r); then
    echo "Image sfc-module found in the local registry.Nothing to do"
else
    echo "Image sfc-module not found in the local registry, pulling"
    podman pull --authfile /var/lib/kubelet/config.json image-registry.openshift-image-registry.svc:5000/openshift-kmm/sfc-module:$(uname -r)
    if [ $? -eq 0 ]; then
        echo "Image sfc-module has been successfully pulled, rebooting.."
        reboot
    else
        echo "Failed to pull image sfc-module"
    fi
fi

