#!/bin/bash
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.

if podman image exists "$ONLOAD_MODULE_IMAGE"; then
    echo "Image $ONLOAD_MODULE_IMAGE found locally. Nothing to do"
else
    echo "Image $ONLOAD_MODULE_IMAGE not found locally, pulling"
    if podman pull --authfile /var/lib/kubelet/config.json "$ONLOAD_MODULE_IMAGE"; then
        echo "Image $ONLOAD_MODULE_IMAGE has been successfully pulled, rebooting.."
        reboot
    else
        echo "Failed to pull image $ONLOAD_MODULE_IMAGE"
    fi
fi
