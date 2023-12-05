# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
# hadolint global ignore=DL3006,DL3020,DL3059,DL3040,DL3041
ARG DTK_AUTO
ARG UBI_BASE=registry.access.redhat.com/ubi9/ubi-minimal:9.2


FROM $DTK_AUTO as builder
ARG ONLOAD_BUILD_PARAMS
ARG ONLOAD_LOCATION
ARG KERNEL_FULL_VERSION

WORKDIR /build/
ADD $ONLOAD_LOCATION onload.tar.gz
RUN mkdir -p /build/onload
RUN tar xzf onload.tar.gz -C /build/onload --strip-components=1
WORKDIR /build/onload/

ENV i_prefix=/opt
RUN scripts/onload_build --kernel --kernelver $KERNEL_FULL_VERSION $ONLOAD_BUILD_PARAMS && \
    scripts/onload_install --nobuild --kernelfiles --kernelver $KERNEL_FULL_VERSION && \
    make -j8 -C src/driver/linux_net CONFIG_SFC_VDPA= CONFIG_SFC_MTD= KVER=$KERNEL_FULL_VERSION && \
    /sbin/depmod -b /opt $KERNEL_FULL_VERSION


FROM $UBI_BASE
ARG KERNEL_VERSION
RUN microdnf install -y kmod && dnf clean all
COPY --from=builder /opt/lib/modules/ /opt/lib/modules/
