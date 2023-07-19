#!/bin/bash
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.

echo "before checking podman images"
if podman image exists image-registry.openshift-image-registry.svc:5000/openshift-kmm/sfc-module:v8.1.0-$(uname -r); then
    echo "Image sfc-module found in the local registry, removing in-tree kernel module"

    podman run --privileged --entrypoint modprobe image-registry.openshift-image-registry.svc:5000/openshift-kmm/sfc-module:v8.1.0-$(uname -r) -rd /opt sfc sfc_driverlink
    if [ $? -eq 0 ]; then
            echo "Successfully removed the in-tree kernel module sfc.ko"
    else
            echo "failed to remove in-tree kernel module sfc.ko"
    fi

    echo "Running container image to insert the oot kernel module sfc.ko"
    podman run --privileged --entrypoint modprobe image-registry.openshift-image-registry.svc:5000/openshift-kmm/sfc-module:v8.1.0-$(uname -r) -d /opt sfc
    if [ $? -eq 0 ]; then
            echo "OOT kernel module sfc.ko is inserted"
    else
            echo "failed to insert OOT kernel module sfc.ko"
    fi
else
   echo "Image sfc-module is not present in local registry, will try after reboot"
fi

