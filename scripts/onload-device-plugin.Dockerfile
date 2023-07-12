# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
ARG ONLOAD_IMAGE
ARG IMAGE_TAG

FROM docker.io/library/golang:1.20.4 AS builder

WORKDIR /app
COPY . /app

RUN go build -o /app/onload-plugin

FROM ${ONLOAD_IMAGE}:${IMAGE_TAG}
RUN microdnf install lshw
COPY --from=builder /app/onload-plugin /usr/bin/onload-plugin
CMD ["/usr/bin/onload-plugin"]
