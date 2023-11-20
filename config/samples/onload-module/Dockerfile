# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
# hadolint global ignore=DL3006,DL3059,DL3040,DL3041
ARG DTK_AUTO
ARG ONLOAD_SOURCE=onload-source
ARG UBI_BASE=registry.access.redhat.com/ubi9/ubi-minimal:9.2


FROM $ONLOAD_SOURCE as onload-source


FROM $DTK_AUTO as builder
ARG ONLOAD_BUILD_PARAMS
ARG KERNEL_FULL_VERSION
COPY --from=onload-source / /opt/onload
WORKDIR /opt/onload
RUN scripts/onload_build --kernel --kernelver $KERNEL_FULL_VERSION $ONLOAD_BUILD_PARAMS
RUN scripts/onload_install --nobuild --kernelfiles --kernelver $KERNEL_FULL_VERSION
RUN depmod $KERNEL_FULL_VERSION


# ON-15418: Consider reducing deps
FROM $UBI_BASE
ARG KERNEL_FULL_VERSION
RUN microdnf install -y kmod && microdnf clean all
COPY --from=builder /lib/modules/$KERNEL_FULL_VERSION/modules* /opt/lib/modules/$KERNEL_FULL_VERSION/
COPY --from=builder /lib/modules/$KERNEL_FULL_VERSION/extra /opt/lib/modules/$KERNEL_FULL_VERSION/extra
RUN ln -s /lib/modules/$KERNEL_FULL_VERSION/kernel /opt/lib/modules/$KERNEL_FULL_VERSION/kernel
