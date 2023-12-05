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
RUN scripts/onload_build --kernel --kernelver $KERNEL_FULL_VERSION $ONLOAD_BUILD_PARAMS
RUN scripts/onload_install --nobuild --kernelfiles --kernelver $KERNEL_FULL_VERSION
RUN depmod $KERNEL_FULL_VERSION


RUN mkdir -p /opt/lib/modules/$KERNEL_FULL_VERSION
RUN cp -v /lib/modules/$KERNEL_FULL_VERSION/modules* /opt/lib/modules/$KERNEL_FULL_VERSION/
RUN cp -rv /lib/modules/$KERNEL_FULL_VERSION/extra /opt/lib/modules/$KERNEL_FULL_VERSION/extra
RUN ln -s /lib/modules/$KERNEL_FULL_VERSION/kernel /opt/lib/modules/$KERNEL_FULL_VERSION/kernel
