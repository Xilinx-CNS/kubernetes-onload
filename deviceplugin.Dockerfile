# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod /app/go.mod
COPY go.sum /app/go.sum
RUN go mod download

COPY Makefile /app/Makefile

COPY pkg/client_helper /app/pkg/client_helper
COPY pkg/control_plane /app/pkg/control_plane
COPY pkg/deviceplugin /app/pkg/deviceplugin

COPY cmd/deviceplugin /app/cmd/deviceplugin
COPY cmd/worker /app/cmd/worker

COPY LICENSE /app/LICENSE

RUN CGO_ENABLED=0 make device-plugin-build worker-build

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8
RUN microdnf install lshw
COPY --from=builder /app/bin/onload-device-plugin /app/bin/onload-worker /usr/bin/
COPY --from=builder /app/LICENSE /licenses/LICENSE
CMD ["/usr/bin/onload-device-plugin"]
