# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
ARG DTK_AUTO

FROM ${DTK_AUTO} as builder
ARG ONLOAD_LOCATION
ARG KERNEL_FULL_VERSION

WORKDIR /build/
ADD ${ONLOAD_LOCATION} onload.tar.gz
RUN mkdir -p /build/onload
RUN tar xzf onload.tar.gz -C /build/onload --strip-components=1
WORKDIR /build/onload/

# Onload
RUN scripts/onload_build --kernel --kernelver ${KERNEL_FULL_VERSION}
RUN scripts/onload_install --nobuild --kernelfiles --kernelver ${KERNEL_FULL_VERSION}

# SFC
RUN make -j8 -C src/driver/linux_net CONFIG_SFC_VDPA= CONFIG_SFC_MTD= KVER=${KERNEL_FULL_VERSION}

FROM ubi8:8.8
ARG KERNEL_FULL_VERSION

RUN yum -y install kmod && yum clean all
RUN mkdir -p /opt/lib/modules/${KERNEL_FULL_VERSION}

# Onload
COPY --from=builder /lib/modules/${KERNEL_FULL_VERSION}/extra/onload.* /opt/lib/modules/${KERNEL_FULL_VERSION}
COPY --from=builder /lib/modules/${KERNEL_FULL_VERSION}/extra/sfc_char.* /opt/lib/modules/${KERNEL_FULL_VERSION}
COPY --from=builder /lib/modules/${KERNEL_FULL_VERSION}/extra/sfc_resource.* /opt/lib/modules/${KERNEL_FULL_VERSION}

# SFC
COPY --from=builder /build/onload/src/driver/linux_net/drivers/net/ethernet/sfc/sfc.ko /opt/lib/modules/${KERNEL_FULL_VERSION}/
COPY --from=builder /build/onload/src/driver/linux_net/drivers/net/ethernet/sfc/sfc_driverlink.ko /opt/lib/modules/${KERNEL_FULL_VERSION}/

RUN /usr/sbin/depmod -b /opt ${KERNEL_FULL_VERSION}
