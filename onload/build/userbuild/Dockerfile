# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
ARG UBI8_MINIMAL_IMAGE

FROM ${UBI8_MINIMAL_IMAGE} as builder
ARG ONLOAD_BUILD_PARAMS
ARG ONLOAD_LOCATION

# Install requirements for building onload userland
RUN microdnf install -y binutils \
gettext \
gawk \
gcc \
sed \
make \
bash \
glibc-common \
libpcap-devel \
perl-Test-Harness \ 
gcc-c++ \
git \
make \
which \
python3-devel \
kmod \
tar

# libcap headers are needed to build onload. The installation of the
# headers on RHEL is normally handled by installing the rpm package libcap-devel.
# However, that package is not available in the standard ubi8
# package repositories, so to get around this issue the package is
# downloaded, built and installed manually.
# START
RUN microdnf -y --enablerepo=ubi-8-baseos-source download --source libcap

RUN mkdir libcap
RUN rpm -i libcap-*.src.rpm \
 && tar xzf ~/rpmbuild/SOURCES/libcap-*.tar.gz --strip-components=1 -C libcap \
 && make -C libcap/libcap install
# END

# Build onload userland components
ADD ${ONLOAD_LOCATION} onload.tar.gz
RUN mkdir -p /build/onload
RUN tar xzf onload.tar.gz -C /build/onload --strip-components=1
WORKDIR /build/onload/
RUN scripts/onload_build --user ${ONLOAD_BUILD_PARAMS}

RUN mkdir /opt/onload
ENV i_prefix=/opt/onload

RUN scripts/onload_install --nobuild --userfiles

# Prepare a minimal image with onload userland
FROM ${UBI8_MINIMAL_IMAGE}

COPY --from=builder /opt/onload /opt/onload

ENTRYPOINT cp -TRv /opt/onload /tmp/onload
