# Onload® Operator v3.0.1 Releases notes

## Update release of the Onload Operator for Kubernetes®

This software package is the v3.0.1 release of the Onload Operator and
Device Plugin, containing minor fixes to v3.0.0.

Version 3.0 is a complete rewrite of v2.0 of the Onload Operator and
the new versions should be selected by any user running OpenOnload® on
Kubernetes.

This release is supplied in the form of binary container images and scripts for
applying to the target cluster.

## Onload®

Onload is the AMD Solarflare application acceleration platform for low
latency networking and scaling cloud data center workloads.

## Operator features

The Onload Operator automates deployment of Onload in Red Hat® OpenShift®
and other Kubernetes clusters. The operator and associated Onload Device Plugin
ease the creation of pods with interfaces that can run applications accelerated
by Onload.

Application network workloads which are to be accelerated by Onload need only
specify their need for Onload in their pod declarations. There is no need
to modify any application container images, which may be used as provided, such
as by third party vendors.

As an example of the ease with which application workloads may be accelerated,
the following snippet of pod definition YAML requests Onload acceleration by
the installed version of Onload.

```yaml
    resources:
      limits:
        amd.com/onload: 1
```

This version of the Onload Operator expects AMD Solarflare adapters to be
presented to application pods as an additional IPVLAN or MACVLAN interface
by the Multus CNI plugin.

## Contents of this release

This release consists of the following components

* Script downloaded and deployed by hand
  * Onload Operator deployment [scripts](https://github.com/Xilinx-CNS/kubernetes-onload)
* Pre-built container images, downloaded automatically from container registry
  * Onload Operator [controller](https://hub.docker.com/r/onload/onload-operator)
  * Onload Device [plugin](https://hub.docker.com/r/onload/onload-device-plugin)
  * Onload [userspace](https://hub.docker.com/r/onload/onload-user)
  * Onload [source](https://hub.docker.com/r/onload/onload-source) -
    used for building Onload kernel modules
  * [sfnettest](https://hub.docker.com/r/onload/sfnettest)
  * Solarflare Enhanced PTP [Daemon](https://hub.docker.com/r/onload/sfptpd)

This product is initially primarily documented in the
[README](https://github.com/Xilinx-CNS/kubernetes-onload#readme)
of its git repository branch.

## Versions of orchestrated software

The Onload Operator is not tied to the versions of the software which
it orchestrates. Future versions of Onload and the other components can be
used by changing the deployment configuration for the Onload Operator. This
version of the Operator has been tested with the following versions of the
other components and these are the versions that the default configurations
supplied with this release will pull in.

| Component  | Initial version |
| ---------- | --------------- |
| OpenOnload | 8.1.2           |
| sfnettest  | 1.6.0-rc2       |
| sfptpd     | 3.7.1.1007      |

Upgrading to new versions of Onload, for example, can be achieved simply
by adjusting cluster configuration files with the new desired version number.

## Kernel Module Management

Management of the Onload kernel module is improved in v3.0 of the Operator by the use of the
[Kernel Module Management Operator](https://github.com/kubernetes-sigs/kernel-module-management).
The KMM Operator ("KMM") can either build the Onload kernel module within
the cluster automatically when a new Linux kernel is required for a node
or can pull in an appropriate kernel module version that has already been
built from a local or remote container registry.

### Day 0/1 deployment of sfc net driver kernel module

Onload depends on an out-of-tree version of the `sfc` net driver which is
distributed with Onload itself. In normal operation the Onload runtime startup
will involve replacing any in-tree `sfc` driver version installed by the OS
with the appropriate out-of-tree version. However, if Kubernetes or other
infrastructure services such as time synchronisation with PTP are operating
over the AMD Solarflare NIC then special provision is required to ensure
this is done earlier in the node startup process.

For the cases where it is required, the Onload Operator solution includes
scripts to arrange the installation of the `sfc` netdriver earlier in node
startup. These take advantage of the OpenShift
[Machine Config Operator](#machine-config-operator).

## Time synchronisation options

Users may use any NTP or PTP software orchestration supplied by their
platform vendor or may deploy the Solarflare Enhanced PTP Daemon (sfptpd)
which is provided as a container image with KubernetesOnload and for which
instructions are available about how to apply to the user's cluster and
ensure other time sync software does not interfere in that case. The Onload
Operator does not provide any additional capability to orchestrate sfptpd.

## Supported platforms

The Onload Operator v3.0 has been written as a generic solution for Kubernetes
clusters and we welcome feedback from attempted use on any platform but this
release is only qualified and officially supported on a limited range of
platforms. Please contact your sales representative at <nic-sales@amd.com>
with as much detail about your deployment platform and topology as possible
if you require support on a different configuration.

| Kubernetes Platform                        | Version |
| ------------------------------------------ | ------  |
| Red Hat OpenShift Container Platform (OCP) |  4.10   |
| Red Hat OpenShift Container Platform (OCP) |  4.12   |
| Red Hat OpenShift Container Platform (OCP) |  4.13   |
| Red Hat OpenShift Container Platform (OCP) |  4.14   |

| Network plugin | Topology    |
| -------------- | ----------- |
| Multus CNI     | sfc MACVLAN |
| Multus CNI     | sfc IPVLAN  |

| Network adapters           |
| -------------------------- |
| AMD Solarflare X2-series   |
| AMD Solarflare 8000-series |

### Considerations for OpenShift versus other Kubernetes platforms

When used with OpenShift, KubernetesOnload offers additional capabilities
to aid deployment.

#### Machine Config Operator

The Machine Config Operator allows actions to be configured on nodes
participating in an OpenShift cluster at startup. This mechanism provides
an opportunity to load the `sfc` net driver needed for Onload before it gets
used by any other system services and provides a way to halt time
synchronisation services like `chronyd` which would prevent the effective
operation of enhanced services like PTP.

Examples and information for using MachineConfigs to assist with a deployment
are provided in this release documentation.

#### The Red Hat Driver Toolkit (DTK) image

The KMM that builds the `onload`, `sfc` and other kernel modules needed for
Onload uses a base container image called a Driver Toolkit (DTK) to build
the modules for either the running or next Linux kernel.

OpenShift provides a DTK images per release which the KMM can automatically
select for the cluster's current requirements. Other Kubernetes platforms
do not necessarily have such a resource readily available.

Therefore for non-OpenShift platforms it will be necessary to do one of
the following:

* Create one's own container image base for kernel builds
* Build in-cluster with the node-host's own kernel headers
* Build module container images out of cluster with custom scripting.

## Prerequisites

The following components need to be available to install and use Kubernetes
Onload:

| Kubernetes Operator |
| ------------------- |
| KMM 1.1             |

## Documentation

The canonical documentation for the Onload Operator solution is the
[README](https://github.com/Xilinx-CNS/kubernetes-onload#readme) file in the
source tree. Additional documents in the GitHub repository provide advice on
building from source. Further documents are authoritative on the orchestrated
components, namely:

| Component | Documentation                                        |
| --------- | ---------------------------------------------------- |
| Onload    | <https://docs.xilinx.com/r/en-US/ug1586-onload-user> |
| sfnettest | <https://docs.xilinx.com/r/en-US/ug1586-onload-user> |
| sfptpd    | <https://docs.xilinx.com/r/en-US/ug1602-ptp-user>    |

## Limitations

This version of the Onload Operator relies on a MACVLAN or
IPVLAN of the network interface to be accelerated being presented into
application pods as a secondary interface using the Multus CNI and does not
accelerate traffic over the Kubernetes software defined network (SDN).

In particular, the following are not supported:

* Calico CNI
* MetalLB
* pod-to-pod traffic within a node
* upgrading from the Onload Operator v2.0 or earlier without
  first removing it

## Major changes since Onload Operator v2.0

This version of the operator is a complete rewrite of the previous version of
the operator supporting different use cases and platforms and with a simplified
deployment scheme.

Notable changes:

* Injection of and acceleration by Onload of a container workload is
  automatic once declaratively requested in the pod definition, with
  no need to modify the application container.
* Kernel module management is automated
* Onload node manager not required

## Changes since Onload Operator v3.0.0

TODO: List the minor changes

## Open source project and support status

The Onload Operator v3.0 has been developed as an open source project. Only
Onload Operator releases and OpenOnload releases announced at
<https://www.xilinx.com/support/download/onload> are supported by AMD
Solarflare's Application Engineering team but we welcome engagement with any
work in progress at <https://github.com/Xilinx-CNS/kubernetes-onload>.

For assistance deploying the Onload Operator on OpenShift with
AMD Solarflare NICs, please contact <support-nic@amd.com>.

## Footnotes

```yaml
SPDX-License-Identifier: MIT
SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
```

All trademarks are acknowledged as being the property of their respective
owners. Kubernetes® is a trademark of The Linux Foundation. Red Hat® and
OpenShift® are trademarks of Red Hat, Inc..
