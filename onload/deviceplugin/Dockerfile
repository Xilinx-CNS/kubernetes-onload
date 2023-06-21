# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
FROM golang:1.20.4 AS builder
COPY . /app/
WORKDIR /app
RUN go build -o /app/onload-plugin

FROM registry.redhat.io/ubi8/ubi-minimal
RUN microdnf install lshw
COPY --from=builder /app/onload-plugin /usr/bin/onload-plugin
CMD ["/usr/bin/onload-plugin"]
