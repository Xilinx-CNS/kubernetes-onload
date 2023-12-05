
# OpenShift MachineConfig for Day 0/1 sfc

For loading of the out-of-tree `sfc` kernel module on boot.

For context, please see
[Out-of-tree `sfc` kernel module in README](../../README.md#out-of-tree-sfc-kernel-module)
and Red Hat documentation:
[Replacing in-tree modules with out-of-tree modules](https://docs.openshift.com/container-platform/4.14/hardware_enablement/kmm-kernel-module-management.html#kmm-replacing-in-tree-modules-with-out-of-tree-modules_kernel-module-management-operator).

The following steps implement a configuration based on KMM Operator's `mcproducer`, which is documented in [4.13.5. MCO yaml creation](https://access.redhat.com/documentation/en-us/openshift_container_platform/4.14/html/specialized_hardware_and_driver_enablement/kernel-module-management-operator#kmm-day1-mco-yaml-creation_kernel-module-management-operator).

An example Butane configuration file is provided ([99-sfc-machineconfig.bu](99-sfc-machineconfig.bu)) in the
format of a template which a convenience script provides variable substitution for your cluster environment.

Two equal methods are provided. Select the one best suited to your terminal environment.

## Building SFC MachineConfig

Configure a development registry and configure cluster for [insecure registries](README.md#insecure-registries)
if required. Specify the base of the following images:

```sh
export REGISTRY_BASE=image-registry.openshift-image-registry.svc:5000/onload-clusterlocal/
```

### Script method

The following requires:

* [`butane` tool](https://docs.openshift.com/container-platform/4.14/installing/install_config/installing-customizing.html#installation-special-config-butane-about_installing-customizing) in your `PATH`

Run:

```sh
#export OPENSHIFT_VER=4.14.0 # Default is v4.10.0 to 4.13.0
make sfc-mc-build ONLOAD_MODULE_IMAGE=$REGISTRY_BASE/onload-module:$ONLOAD_VERSION-$ONLOAD_KERNEL_VERSION
```

### Docker method

The following requires:

* Docker (or a symbolic link to an API equivalent, e.g. `podman`) in your `$PATH`
* Internet access to `quay.io` to pull the Butane container image

Run:

```sh
#export OPENSHIFT_VER=4.14.0 # Default is v4.10.0 to 4.13.0
make sfc-mc-docker-build ONLOAD_MODULE_IMAGE=$REGISTRY_BASE/onload-module:$ONLOAD_VERSION-$ONLOAD_KERNEL_VERSION
```

## Deploying the SFC MachineConfig

To run `kubectl apply`:

```sh
make sfc-mc-deploy
```

## Footnotes

Copyright (c) 2023 Advanced Micro Devices, Inc.
