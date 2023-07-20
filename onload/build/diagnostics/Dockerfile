# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
FROM image-registry.openshift-image-registry.svc:5000/onload-clusterlocal/onload-user:v8.1.0 AS onload-user

FROM ubi8

RUN dnf install -y libpcap procps-ng util-linux iptables python3

# Best effort to install tcpdump for onload_tcpdump.
RUN curl http://mirror.centos.org/centos/8-stream/AppStream/x86_64/os/Packages/tcpdump-4.9.3-3.el8.x86_64.rpm -o tcpdump-4.9.3-3.el8.x86_64.rpm
RUN rpm -i tcpdump-4.9.3-3.el8.x86_64.rpm || exit 0

COPY --from=onload-user /opt/onload/usr/bin/ /usr/bin/
