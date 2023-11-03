# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
# hadolint global ignore=DL3006
ARG DTK_AUTO
ARG ONLOAD_SOURCE=onload-source


FROM $ONLOAD_SOURCE as onload-source


FROM $DTK_AUTO
ARG ONLOAD_BUILD_PARAMS
ARG KERNEL_FULL_VERSION
COPY --from=onload-source / /opt/onload
WORKDIR /opt/onload
ENV i_prefix=/opt
RUN scripts/onload_build --kernel --kernelver $KERNEL_FULL_VERSION $ONLOAD_BUILD_PARAMS && \
    scripts/onload_install --nobuild --kernelfiles --kernelver $KERNEL_FULL_VERSION && \
    make -j8 -C src/driver/linux_net CONFIG_SFC_VDPA= CONFIG_SFC_MTD= KVER=$KERNEL_FULL_VERSION && \
    /sbin/depmod -b /opt $KERNEL_FULL_VERSION
