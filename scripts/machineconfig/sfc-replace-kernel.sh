#!/bin/bash
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.

echo "checking if the appropriate podman images exist"
echo "Looking for the image: $ONLOAD_MODULE_IMAGE"
if podman image exists "$ONLOAD_MODULE_IMAGE"; then
    echo "Image $ONLOAD_MODULE_IMAGE found locally, removing in-tree kernel module"

    if podman run --privileged --entrypoint modprobe "$ONLOAD_MODULE_IMAGE" -rd /opt sfc sfc_driverlink; then
            echo "Successfully removed the in-tree kernel module sfc.ko"
    else
            echo "failed to remove in-tree kernel module sfc.ko"
    fi

    echo "Running container image to insert the oot kernel module sfc.ko"
    if podman run --privileged --entrypoint modprobe "$ONLOAD_MODULE_IMAGE" -d /opt sfc; then
            echo "OOT kernel module sfc.ko is inserted"
    else
            echo "failed to insert OOT kernel module sfc.ko"
    fi
else
   echo "Image $ONLOAD_MODULE_IMAGE is not present locally, will try after reboot"
fi
