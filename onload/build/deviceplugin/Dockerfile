# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
FROM golang:1.20.4 AS builder

ARG KUBERNETES_ONLOAD_LOCATION

WORKDIR /app
ADD ${KUBERNETES_ONLOAD_LOCATION} kubernetes-onload.tar.gz
RUN mkdir -p /app/kubernetes-onload
RUN tar xzf kubernetes-onload.tar.gz -C /app/kubernetes-onload --strip-components=1

WORKDIR /app/kubernetes-onload/onload/build/deviceplugin
RUN go build -o /app/onload-plugin

FROM image-registry.openshift-image-registry.svc:5000/onload-clusterlocal/onload-user:v8.1.0
RUN microdnf install lshw
COPY --from=builder /app/onload-plugin /usr/bin/onload-plugin
CMD ["/usr/bin/onload-plugin"]
