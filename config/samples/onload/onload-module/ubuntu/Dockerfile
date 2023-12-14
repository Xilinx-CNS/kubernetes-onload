# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
# hadolint global ignore=DL3006,DL3008
ARG UBUNTU_IMAGE
ARG ONLOAD_SOURCE=onload-source


FROM $ONLOAD_SOURCE as onload-source


FROM $UBUNTU_IMAGE as builder
ARG ONLOAD_BUILD_PARAMS
ARG KERNEL_FULL_VERSION

RUN apt-get update && apt-get install --no-install-recommends -y \
    bc \
    bison \
    flex \
    libelf-dev \
    gnupg \
    wget \
    git \
    make \
    gcc \
    "linux-headers-$KERNEL_FULL_VERSION"

COPY --from=onload-source / /opt/onload
WORKDIR /opt/onload
RUN scripts/onload_build --kernel --kernelver $KERNEL_FULL_VERSION $ONLOAD_BUILD_PARAMS
RUN scripts/onload_install --nobuild --kernelfiles --kernelver $KERNEL_FULL_VERSION
RUN depmod $KERNEL_FULL_VERSION


FROM $UBUNTU_IMAGE
ARG KERNEL_FULL_VERSION
RUN apt-get update && apt-get install -y kmod && apt-get clean
COPY --from=builder /lib/modules/$KERNEL_FULL_VERSION/modules* /opt/lib/modules/$KERNEL_FULL_VERSION/
COPY --from=builder /lib/modules/$KERNEL_FULL_VERSION/extra /opt/lib/modules/$KERNEL_FULL_VERSION/extra
RUN ln -s /lib/modules/$KERNEL_FULL_VERSION/kernel /opt/lib/modules/$KERNEL_FULL_VERSION/kernel

