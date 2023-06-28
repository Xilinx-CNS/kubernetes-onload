# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
FROM golang:1.20.4 AS builder

WORKDIR /app
RUN git clone --depth 1 https://github.com/Xilinx-CNS/kubernetes-onload.git

WORKDIR /app/kubernetes-onload/onload/deviceplugin
RUN go build -o /app/onload-plugin

FROM image-registry.openshift-image-registry.svc:5000/onload-clusterlocal/onload-user:git27b3826
RUN microdnf install lshw
COPY --from=builder /app/onload-plugin /usr/bin/onload-plugin
CMD ["/usr/bin/onload-plugin"]
