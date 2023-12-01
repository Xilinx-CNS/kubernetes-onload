#!/bin/bash
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
DIR=$(dirname "$0")
mkdir -p "$DIR"/output
sed -e "s/SED_OPENSHIFT_VERSION/$1/g"   \
    -e "s#SED_ONLOAD_MODULE_IMAGE#$2#g" \
    -e "s/SED_NODE_TYPE/$3/g"           \
    "$DIR"/99-sfc-machineconfig.bu |    \
butane -d "$DIR" \
       -o "$DIR"/output/99-sfc-machineconfig.yaml
