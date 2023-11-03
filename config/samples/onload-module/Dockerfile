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
ENV i_prefix=/opt
RUN scripts/onload_build --kernel --kernelver $KERNEL_FULL_VERSION $ONLOAD_BUILD_PARAMS
RUN scripts/onload_install --nobuild --kernelfiles --kernelver $KERNEL_FULL_VERSION
RUN make -j8 -C src/driver/linux_net CONFIG_SFC_VDPA= CONFIG_SFC_MTD= KVER=$KERNEL_FULL_VERSION
RUN /sbin/depmod -b /opt $KERNEL_FULL_VERSION


# ON-15418: Consider reducing deps
FROM $UBI_BASE
ARG KERNEL_FULL_VERSION
RUN microdnf install -y kmod && dnf clean all
COPY --from=builder /opt/lib/modules/ /opt/lib/modules/
COPY --from=builder /build/onload/src/driver/linux_net/drivers/net/ethernet/sfc/sfc*.ko /opt/lib/modules/$KERNEL_FULL_VERSION/
COPY --from=builder /build/onload/src/driver/linux_net/drivers/net/ethernet/sfc/sfc_driverlink.ko /opt/lib/modules/$KERNEL_FULL_VERSION/