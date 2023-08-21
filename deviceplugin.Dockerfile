# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
FROM golang:1.20.4 AS builder

COPY go.mod /app/go.mod
COPY go.sum /app/go.sum
COPY pkg/deviceplugin /app/

WORKDIR /app
RUN go build -o /app/onload-plugin

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8
RUN microdnf install lshw
COPY --from=builder /app/onload-plugin /usr/bin/onload-plugin
CMD ["/usr/bin/onload-plugin"]
