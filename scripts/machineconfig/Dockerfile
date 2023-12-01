# SPDX-License-Identifier: MIT
# Copyright (c) 2023 Advanced Micro Devices, Inc.

FROM quay.io/coreos/butane:release AS build-stage
ARG ONLOAD_MODULE_IMAGE
ARG OPENSHIFT_VER=4.12.0
ARG NODE_TYPE=worker
WORKDIR /sfc-butane
COPY . /sfc-butane
RUN /sfc-butane/create_machine_config.sh ${OPENSHIFT_VER} ${ONLOAD_MODULE_IMAGE} ${NODE_TYPE}

FROM scratch as export-stage
COPY --from=build-stage /sfc-butane/output/99-sfc-machineconfig.yaml /
