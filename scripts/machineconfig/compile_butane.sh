#!/bin/bash
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
CONTAININGDIR=$(dirname "$0")
mkdir -p "$CONTAININGDIR"/mc
sed -e "s/SED_OPENSHIFT_VERSION/$1/g" \
    -e "s#SED_ONLOAD_MODULE_IMAGE#$2#g" \
    -e "s/SED_NODE_TYPE/$3/g"         \
    "$CONTAININGDIR"/99-sfc-machineconfig.bu > \
    "$CONTAININGDIR"/mc/99-sfc-machineconfig-compiled.bu
butane -s "$CONTAININGDIR"/mc/99-sfc-machineconfig-compiled.bu -d "$CONTAININGDIR" \
       -o "$CONTAININGDIR"/mc/99-sfc-machineconfig.yaml
rm -f "$CONTAININGDIR"/mc/99-sfc-machineconfig-compiled.bu
