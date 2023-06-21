# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
ARG DTK_AUTO
ARG KERNEL_VERSION

FROM ${DTK_AUTO} as builder
ARG ONLOAD_BUILD_PARAMS

WORKDIR /build/
RUN git clone --depth 1 https://github.com/Xilinx-CNS/onload.git

WORKDIR /build/onload

RUN scripts/onload_build --kernel ${ONLOAD_BUILD_PARAMS}
RUN scripts/onload_install --nobuild --kernelfiles

FROM docker.io/redhat/ubi8:latest
ARG KERNEL_VERSION

RUN yum -y install kmod && yum clean all
RUN mkdir -p /opt/lib/modules/${KERNEL_VERSION}

COPY --from=builder /lib/modules/${KERNEL_VERSION}/extra/onload.* /opt/lib/modules/${KERNEL_VERSION}
COPY --from=builder /lib/modules/${KERNEL_VERSION}/extra/sfc_char.* /opt/lib/modules/${KERNEL_VERSION}
COPY --from=builder /lib/modules/${KERNEL_VERSION}/extra/sfc_resource.* /opt/lib/modules/${KERNEL_VERSION}

RUN /usr/sbin/depmod -b /opt
