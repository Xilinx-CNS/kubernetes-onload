# SPDX-License-Identifier: MIT
# Copyright (c) 2023 Advanced Micro Devices, Inc.

FROM quay.io/coreos/butane:release AS build-stage
ARG IMG
ARG OPENSHIFT_VER=4.12.0
ARG NODE_TYPE=worker
WORKDIR /sfc-butane
COPY . /sfc-butane
RUN /sfc-butane/compile_butane.sh ${OPENSHIFT_VER} ${IMG} ${NODE_TYPE}

FROM scratch as export-stage
COPY --from=build-stage /sfc-butane/mc/99-sfc-machineconfig.yaml /
