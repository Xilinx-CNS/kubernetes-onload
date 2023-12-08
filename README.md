# Onload Operator and Onload Device Plugin for Kubernetes and OpenShift

Use Onload to accelerate your workloads in Kubernetes and OpenShift clusters.

## Installation requirements

### Supported environment

* [OpenOnload](https://github.com/Xilinx-CNS/onload) (including EnterpriseOnload) 8.1+
* [AMD Solarflare](https://www.solarflare.com) hardware (`sfc`)
* OpenShift Container Platform (OCP) 4.10+ with
  * [Kernel Module Management (KMM) Operator](https://kmm.sigs.k8s.io/) 1.1+ ([OpenShift documentation](https://docs.openshift.com/container-platform/4.14/hardware_enablement/kmm-kernel-module-management.html)
* Both restricted network or internet-connected clusters

Deployment can also be performed on Kubernetes 1.23+ but full implementation details are not currently provided.
The Onload Device Plugin is not currently designed for standalone deployment.

Please see Release Notes for further detail on version compatibility and feature availability.

### Access to container images & configuration files

#### Terminal

Your terminal requires access to:

* Your cluster via `kubectl` or `oc`
* [This repository](https://github.com/Xilinx-CNS/kubernetes-onload)

This documentation standardises on `kubectl` but both are compatible: `alias kubectl=oc`.

Most users can benefit from the [provided container images](#provided-images) along with
[KMM's in-cluster `onload-module` builds](#onload-module-in-cluster-builds).
A more comprehensive development environment is required for special use cases, namely:

* [building bespoke `onload-module` images](#onload-module-pre-built-images) outside the cluster,
* [OpenShift MachineConfig for Day 0/1 sfc](#openshift-machineconfig-for-sfc),
* [developing Onload](#onload-source--onload-user), and/or
* [developing Onload Operator or Onload Device Plugin](#onload-operator--onload-device-plugin).

#### Cluster

Your cluster requires access to the following [provided container images](#provided-images):

* `onload-operator`
* `onload-device-plugin`
* `onload-user`
* `onload-source` (if in-cluster builds)
* `sfptpd` (optional)
* `sfnettest` (optional)
* KMM Operator & dependents
* DTK (if in-cluster builds on OpenShift)
  * OpenShift includes a `driver-toolkit` (DTK) image in each release. No action should be required.

The cluster also requires access to the following node-specific kernel module container image(s) which may be provided
externally or internally. If using [in-cluster builds](#onload-module-in-cluster-builds), push access to an internal
registry will be required. Otherwise, only pull access is required if these images are
[pre-built](#onload-module-pre-built-images).
Please see Release Notes for further detail on feature availability.

* `onload-module`

When using [in-cluster builds](#onload-module-in-cluster-builds), other dependencies may be required depending on
the method selected. These may include `ubi-minimal` container image and
[UBI RPM repositories](https://access.redhat.com/articles/4238681).

### Provided Images

This repository's YAML configuration uses the following images by default:

* [`docker.io/onload/onload-operator`](https://hub.docker.com/r/onload/onload-operator)
* [`docker.io/onload/onload-device-plugin`](https://hub.docker.com/r/onload/onload-device-plugin)
* [`docker.io/onload/onload-source`](https://hub.docker.com/r/onload/onload-source)
* [`docker.io/onload/onload-user`](https://hub.docker.com/r/onload/onload-user)
* [`docker.io/onload/sfptpd`](https://hub.docker.com/r/onload/sfptpd)
* [`docker.io/onload/sfnettest`](https://hub.docker.com/r/onload/sfnettest)

For restricted networks these container images can be mirrored.

## Deployment

To accelerate a pod:

* Configure the [Onload Operator](#onload-operator)
* Configure an [Onload Custom Resource (CR)](#onload-custom-resource-cr)
* Configure a pod network with AMD Solarflare interfaces, ie. Multus IPVLAN or MACVAN
* Configure the [out-of-tree `sfc` module](#out-of-tree-sfc-kernel-module)
* [Configure your pods](#run-onloaded-applications) to use the resource provided by
  the [Onload Device Plugin](#onload-device-plugin) and the network

### Onload Operator

The Onload Operator follows the [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)
pattern which links a [Kubernetes Controller](https://kubernetes.io/docs/concepts/architecture/controller/),
implemented here in the `onload-operator` container image, to one or more Custom Resource Definitions (CRD),
implemented here in the `Onload` *kind* of CRD.

To deploy the Onload Operator, its controller container and CRD, run:

```sh
kubectl apply -k https://github.com/Xilinx-CNS/kubernetes-onload/config/default
```

This deploys the following by default:

* In Namespace [`onload-operator-system`](./config/default/kustomization.yaml) with prefix
  [`onload-operator-`](./config/default/kustomization.yaml):
  * [Onload CRD](config/crd/bases/onload.amd.com_onloads.yaml)
  * [Operator](config/manager/kustomization.yaml) version from DockerHub.
  * [RBAC](config/rbac) for these components

The Onload Operator will not deploy the components necessary for accelerating workload pods without
an `Onload` *kind* of Custom Resource (CR).

#### Local Onload Operator images in restricted networks

For restricted networks, the `onload-operator` image location will require changing from its DockerHub default.
To run the above command using locally hosted container images, open this repository locally and use the
[following overlay](config/samples/default-clusterlocal/kustomization.yaml):

```sh
git clone https://github.com/Xilinx-CNS/kubernetes-onload && cd kubernetes-onload

cp -r config/samples/default-clusterlocal config/samples/my-operator
$EDITOR config/samples/my-operator/kustomization.yaml
kubectl apply -k config/samples/my-operator
```

### Onload Device Plugin

The Onload Device Plugin implements the [Kubernetes Device Plugin API](https://kubernetes.io/docs/concepts/extend-kubernetes/compute-storage-net/device-plugins/)
to expose a [Kubernetes Resource](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
named `amd.com/onload`.

It is distributed as the container image `onload-device-plugin` and is deployed and configured entirely by
the Onload Operator. You can configure the image location and its settings via the Onload Operator within
a [Onload Custom Resource (CR)](#onload-custom-resource-cr).

### Onload Custom Resource (CR)

Instruct the Onload Operator to deploy the components necessary for accelerating workload pods by deploying
a `Onload` *kind* of Custom Resource (CR).

If your cluster is internet-connected OpenShift and you want to use in-cluster builds with the current version
of OpenOnload, run:

```sh
kubectl apply -k https://github.com/Xilinx-CNS/kubernetes-onload/config/samples/onload/overlays/in-cluster-build-ocp
```

This takes a [base `Onload` CR template](config/samples/onload/base/onload_v1alpha1_onload.yaml) and adds the
appropriate [image versions](config/samples/onload/overlays/in-cluster-build-ocp/kustomization.yaml) and
[in-cluster build configuration](config/samples/onload/overlays/in-cluster-build-ocp/patch-onload.yaml). To customise
this recommended overlay further, see the variant steps below.

The above overlay configures KMM to `modprobe onload` but `modprobe sfc` is also required.
Please see [Out-of-tree `sfc` module](#out-of-tree-sfc-kernel-module) for options.

#### In-cluster builds in restricted networks

In restricted networks or on other versions of Kubernetes, change the container image locations and build method(s)
to suit your environment. For example, to adapt the overlay
[in-cluster build on OpenShift in restricted network](config/samples/onload/overlays/in-cluster-build-ocp-clusterlocal):

```sh
git clone https://github.com/Xilinx-CNS/kubernetes-onload && cd kubernetes-onload

cd config/samples/onload
cp -r overlays/in-cluster-build-ocp-clusterlocal overlays/my-onload
$EDITOR overlays/my-onload/kustomization.yaml
$EDITOR overlays/my-onload/patch-onload.yaml
kubectl apply -k overlays/my-onload
```

Consider configuring:

* Onload Operator & Onload Device Plugin container image tags (recommended to match)
  * In above `kustomization.yaml`
* Onload Source & Onload User container image tags and Onload version (all must match)
  * In above `kustomization.yaml` & `version` attribute in `patch-onload.yaml`
* [Onload Module build method](#onload-module-in-cluster-builds) and tag (match tag to Onload version for clarity)
  * In above `kustomization.yaml` & `build` section in `patch-onload.yaml`

#### Onload Module in-cluster builds

The Onload Operator supports all of KMM's core methods for providing compiled kernel modules to the nodes.

Some working examples are provided for use with the [Onload CR](#onload-custom-resource-cr):

* [dtk-ubi](config/samples/onload/onload-module/dtk-ubi) -- currently **recommended** for OpenShift,
  depends on DTK & UBI
* [dtk-only](config/samples/onload/onload-module/dtk-only) -- for OpenShift in very restricted networks,
  depends only on official OpenShift DTK
* [mkdist-direct](config/samples/onload/onload-module/mkdist-direct) -- for consistency with non-containerised
  Onload deployments (not recommended)
* [ubuntu](config/samples/onload/onload-module/ubuntu) -- representative sample for non-OpenShift clusters

Please see [Onload Module pre-built images](#onload-module-pre-built-images) for the alternative to building in-cluster.

### Out-of-tree `sfc` kernel module

The out-of-tree `sfc` kernel module is currently required when using the provided `onload` kernel module
with a Solarflare card.

The following methods may be used:

* Configure the Onload Operator to deploy a KMM Module for `sfc`. Please see the example comment in
  [in-cluster build configuration](config/samples/onload/overlays/in-cluster-build-ocp/patch-onload.yaml).

* [OpenShift MachineConfig for Day 0/1 sfc](#openshift-machineconfig-for-sfc). This is for when newer driver features
  are required at boot time while using OpenShift, or when Solarflare NICs are used for OpenShift machine traffic, so
  as to avoid kernel module reloads disconnecting nodes.

* A user-supported method beyond the scope of this document, such as a custom kernel build or in-house OS image.

> [!TIP]
> Network interface names can be fixed with UDEV rules.
>
> On a RHCOS node within OpenShift, the directory `/etc/udev/rules.d/` can be written to with a `MachineConfig` CR.

### sfptpd

The Solarflare Enhanced PTP Daemon (sfptpd) is not managed by Onload Operator but deployment instructions are included
in this repository.

Please see [config/samples/sfptpd/](config/samples/sfptpd/) for documentation and examples.

## Operation

After you have completed the [Deployment](#deployment) steps your cluster is configured with the capability to
accelerate workloads using Onload.

An easy test to verify everything is correctly configured is the
[sfnettest example](#example-client-server-with-sfnettest).

### Run Onloaded applications

To accelerate your workload, configure a pod with a AMD Solarflare network interface and
[`amd.com/onload` resource](#resource-amdcomonload):

```yaml
kind: Pod
metadata:
  annotations:
    k8s.v1.cni.cncf.io/networks: ipvlan-sf0
spec:
  ...
  containers:
  - ...
    resources:
      limits:
        amd.com/onload: 1
```

All applications started within the pod environment will be accelerated due to the `LD_PRELOAD` environment variable.

### Resource `amd.com/onload`

This Kubernetes Resource automatically exposes the following to a requesting pod:

Device mounts:

* `/dev/onload`
* `/dev/onload_epoll`
* `/dev/sfc_char`

Library mounts (by default in `/opt/onload/usr/lib64/`):

* `libonload.so`
* `libonload_ext.so`

Environment variables (if `setPreload` is true):

* `LD_PRELOAD=<library-mount>/libonload.so`

Binary mounts (if `mountOnload` is true, by default in `/opt/onload/usr/bin/`)

* `onload`

If you wish to customise where files are mounted in the container's filesystem this can be configured with the fields
of `spec.devicePlugin` in an Onload CR.

### Example client-server with sfnettest

Please see [config/samples/sfnettest](config/samples/sfnettest).

### Using Onload profiles

If you want to run your onloaded application with a runtime profile we suggest
using a ConfigMap to set the environment variables in the pod(s).
We have included an example definition for the 'latency' profile in
[`config/samples/profiles/`](config/samples/profiles/) directory.

To deploy a ConfigMap named `onload-latency-profile` in the current namespace:

```sh
kubectl apply -k https://github.com/Xilinx-CNS/kubernetes-onload/config/samples/profiles
```

To use this in your pod, add the following to the container spec in your pod definition:

```yaml
kind: Pod
...
spec:
  ...
  containers:
  - ...
    envFrom:
      - configMapRef:
          name: onload-latency-profile
```

#### Converting an existing profile

If you have an existing profile defined as a `.opf` file you can generate a new
ConfigMap definition from this using the [`scripts/profile_to_configmap.sh`](scripts/profile_to_configmap.sh)
script.

`profile_to_configmap.sh` takes in a comma separated list of profiles and will
output the text definition of the ConfigMap which can be saved into a file, or
sent straight to the cluster. To apply the generated ConfigMap straight away
run:

```sh
./scripts/profile_to_configmap.sh -p /path/to/profile.opf | kubectl apply -f -
```

Currently the script produces ConfigMaps with a fixed naming structure,
for example if you want to create a ConfigMap from a profile called
`name.opf` the generated name will be `onload-name-profile`.

## Build

### Onload Module pre-built images

Developing Onload Operator does not require building the `onload-module` image as they can be built in-cluster by KMM.

To build these images outside the cluster, please see [./build/onload-module/](build/onload-module/)
for documentation and examples.

### OpenShift MachineConfig for sfc

Please see [scripts/machineconfig/](scripts/machineconfig/) for documentation and examples
to deploy an out-of-tree `sfc` module in Day 0/1 (on boot).

### Onload Operator & Onload Device Plugin

Using Onload Operator does not require building these images as official images are available.

Please see [DEVELOPING](DEVELOPING.md) documentation.

### Onload Source & Onload User

Developing Onload Operator does not require building these images as official images are available.

If you wish to build these images, please follow ['Distributing as container image' in Onload repository's DEVELOPING](https://github.com/Xilinx-CNS/onload/blob/master/DEVELOPING.md#distributing-as-container-image).

### Insecure registries

If your registry is not running with TLS configured, additional configuration may be necessary for accessing
and pushing images. For example:

```sh
$ oc edit image.config cluster
...
spec:
  registrySources:
    insecureRegistries:
    - image-registry.openshift-image-registry.svc:5000
```

## Caveats

* The Onload Operator manages KMM resources on behalf of the user but does not provide feature parity with KMM. Examples
  of features not included are: in-cluster container image build signing, node version freezing during ordered upgrade
  (Onload Operator manages these labels), miscellaneous DevicePlugin configuration, configuration of registry
  credentials (beyond existing cluster configuration), customisation of kernel module parameters and soft dependencies,
  and customisation of Namespace and Service Account for dependent resources (instead inherited from
  [Onload CR](#onload-custom-resource-cr)). Configuring `PreflightValidation` can be performed independently while
  the Onload Operator is running.

* Reloading of the kernel modules `onload` (and optionally `sfc`) will occur on first deployment and under certain
  reconfigurations. When using AMD Solarflare interfaces for Kubernetes control plane traffic, ensure node network
  interface configuration and workloads will regain correct configuration and cluster connectivity after reload.

* Interface names may change when switching from an in-tree to out-of-tree `sfc` kernel module. This is due to
  [changes in default interface names](https://support.xilinx.com/s/article/000034471) between versions 4 and 5.
  Ensure [appropriate measures](#out-of-tree-sfc-kernel-module) have been taken for any additional network
  configurations that depend on this information.

## Footnotes

Copyright (c) 2023 Advanced Micro Devices, Inc.
