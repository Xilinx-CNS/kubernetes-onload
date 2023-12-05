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
    "linux-headers-$KERNEL_VERSION"

COPY --from=onload-source / /opt/onload
WORKDIR /opt/onload
ENV i_prefix=/opt
RUN scripts/onload_build --kernel --kernelver $KERNEL_FULL_VERSION $ONLOAD_BUILD_PARAMS && \
    scripts/onload_install --nobuild --kernelfiles --kernelver $KERNEL_FULL_VERSION


FROM $UBUNTU_IMAGE
COPY --from=builder /opt/lib/modules/ /opt/lib/modules/
RUN /sbin/depmod -b /opt
