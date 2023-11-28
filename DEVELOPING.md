# Developing Onload Operator and Onload Device Plugin

## Environment

* Go version 1.21+
* Access to your cluster via `kubectl` or `oc`
* Docker or equivalent (e.g. `podman`)

The Onload Operator is built with the [Operator SDK](https://sdk.operatorframework.io/)
and [Kubebuilder](https://kubebuilder.io/).

## Prerequisites

The Onload Operator and Onload Device Plugin consume Onload container images (`onload-user` and either `onload-source` or `onload-module`). You may wish to pre-populate your cluster's container image registry, either with the [official images provided](README.md#provided-images) or [your own builds](README.md#build).

## Build and deploy Onload Operator from source

Configure a development registry and configure cluster for [insecure registries](README.md#insecure-registries)
if required. Specify the base of the following images:

```sh
export REGISTRY_BASE=image-registry.openshift-image-registry.svc:5000/onload-clusterlocal/
```

Create and push the Onload Operator controller image:

```sh
make docker-build docker-push IMG=$REGISTRY_BASE/onload-operator:latest
```

Create and push the Onload Device Plugin image:

```sh
export DEVICE_IMG=$REGISTRY_BASE/onload-device-plugin:latest
make device-plugin-docker-build device-plugin-docker-push
```

Deploy the Onload Operator:

```sh
make deploy IMG=$REGISTRY_BASE/operator:latest
```
Ensure that `$DEVICE_IMG` is exported when deploying the operator, or append `DEVICE_IMG=...` to the make invocation.

Continue with [deploying the Onload CR](README.md#onload-custom-resource-cr).


## Footnotes

Copyright (c) 2023 Advanced Micro Devices, Inc.
