# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
FROM golang:1.20.4 AS builder

WORKDIR /app

COPY go.mod /app/go.mod
COPY go.sum /app/go.sum
RUN go mod download

COPY pkg/deviceplugin /app/pkg/deviceplugin
COPY cmd/deviceplugin /app/cmd/deviceplugin

RUN go build -o /app/onload-plugin ./cmd/deviceplugin

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8
RUN microdnf install lshw
COPY --from=builder /app/onload-plugin /usr/bin/onload-plugin
CMD ["/usr/bin/onload-plugin"]
