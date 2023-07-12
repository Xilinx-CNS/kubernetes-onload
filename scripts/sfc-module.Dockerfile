# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
ARG DTK_AUTO
FROM ${DTK_AUTO} as builder

ARG KERNEL_VERSION

# This is a bit of a hack using $ONLOAD_VERSION to pass in the filename of the
# release tarball.
ARG ONLOAD_VERSION

WORKDIR /build/
RUN mkdir -p /build/onload
COPY ${ONLOAD_VERSION} /build
RUN tar xzf ${ONLOAD_VERSION} -C /build/onload --strip-components=1
WORKDIR /build/onload/

# Currently there are issues regarding when building the sfc driver due to
# differences between the DTK image used for building and the ubi image used
# for loading the drivers.
#
# Issues found are with:
# * vdpa
# * mtd
#
# To prevent building the problematic things we have to pass additional
# parameters to the driver build. Unfortunately it is currently possible to do
# this with onload's build scripts, so the driver's Makefile is used directly.
RUN make -j8 -C src/driver/linux_net CONFIG_SFC_VDPA= CONFIG_SFC_MTD= KVER=${KERNEL_VERSION}

FROM registry.access.redhat.com/ubi8/ubi
ARG KERNEL_VERSION

RUN yum -y install kmod && yum clean all
RUN mkdir -p /opt/lib/modules/${KERNEL_VERSION}
COPY --from=builder /build/onload/src/driver/linux_net/drivers/net/ethernet/sfc/sfc.ko /opt/lib/modules/${KERNEL_VERSION}/
COPY --from=builder /build/onload/src/driver/linux_net/drivers/net/ethernet/sfc/sfc_driverlink.ko /opt/lib/modules/${KERNEL_VERSION}/
RUN /usr/sbin/depmod -b /opt ${KERNEL_VERSION}
