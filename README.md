# Onload Operator v3 for Kubernetes and OpenShift

## Deployment guide

This section outlines the steps required to deploy Onload components in an OCP cluster in order to enable Onload acceleration in user pods.

Expected environment:
- (WIP) Onload/master
- (WIP) OCP 4.12 with:
  - Node Feature Discovery (NFD) Operator
  - Kernel Module Management (KMM) Operator

### SFC MachineConfig

To apply the SFC MachineConfig, it is a prerequisite that there exists a local docker registry containing an image with the SFC module built and present in /opt in the node filesystem. This is expected to be present at the following location:

```
image-registry.openshift-image-registry.svc:5000/openshift-kmm/sfc-module
```

To generate this image, please use SFC module:
```
$ oc apply -f sfc/kmm/sfc-module.yaml
```

The generated image must be labelled with the exact kernel version necessary for the worker nodes. All worker nodes must share the same kernel version.

Apply the SFC MachineConfig:

```
$ butane sfc/mco/99-sfc-machineconfig.bu -d sfc/mco/ -o sfc/mco/99-sfc-machineconfig.yaml
$ oc apply -f sfc/mco/99-sfc-machineconfig.yaml
```

Please ensure that the openshift version specified within the above butane file tracks the appropriate specification revision.
https://github.com/coreos/butane/blob/main/docs/specs.md#stable-specification-versions
E.g. OpenShift 4.10.62 -> 4.10.0


### Onload manifests

Create and apply the composed Kustomize manifest that defines Onload resources, including namespaces:

```
$ oc apply [--dry-run=client] -k onload/dev
$ oc start-build onload-device-plugin -n onload-device-plugin --from-dir onload/deviceplugin
```

### Disable chronyd

This will disable the `chronyd.service` on `worker` role nodes.

```
$ butane sfptpd/99-worker-chronyd.bu -o sfptpd/99-worker-chronyd.yaml
$ oc apply -f sfptpd/99-worker-chronyd.yaml
```

Please ensure that the openshift version specified within the above butane file tracks the appropriate specification revision.
https://github.com/coreos/butane/blob/main/docs/specs.md#stable-specification-versions
E.g. OpenShift 4.10.62 -> 4.10.0


### sfptpd

```
$ oc new-project sfptpd
$ oc create -f sfptpd/0000-sfptpd-build.yaml
$ oc create -f sfptpd/1000-sfptpd-daemonset.yaml
```

Please note that `sfptpd/1000-sfptpd-daemonset.yaml` uses the interface name `sf0` as a placeholder for development purposes. This should be modified to use an appropriate value before deploying to the cluster.

### Onloaded application

Here we suggest running the `sfnt-pingpong` from [https://github.com/Xilinx-CNS/cns-sfnettest.git](https://github.com/Xilinx-CNS/cns-sfnettest.git).

Create a new BuildConfig to build the app and a couple of new demo pods with:

```
$ oc apply -f examples/cns-sfnettest.yaml
```

It should create pods `onload-sfnettest-server` and `onload-sfnettest-client` on workers `compute-0` and `compute-1` respectively.

Get the SFC IP address of the `onload-sfnettest-server` pod:
```console
$ oc describe pod onload-sfnettest-server | grep AddedInterface
  Normal  AddedInterface  24s   multus   Add eth0 [192.168.8.114/23] from openshift-sdn
  Normal  AddedInterface  24s   multus   Add net1 [198.19.0.1/16] from default/ipvlan-sf0
```

The server pod is already running the accelerated `sfnt-pingpong` instance.

Run the client:
```
sh-4.4# LD_PRELOAD=/opt/onload/usr/lib64/libonload.so ./sfnt-pingpong udp 198.19.0.1
```

Try running with and without `onload` to compare the reported performance.

Based on [examples/README.md](examples/README.md).

## Uninstall

Use `oc delete` to uninstall Onload and example pods:
```
$ oc delete -f examples/cns-sfnettest.yaml
$ oc delete project sfptpd
$ oc delete -f 99-worker-chronyd.yaml
$ oc delete -k onload/dev
$ oc delete -f sfc/mco/99-sfc-machineconfig.yaml
```

(Applying and removing MachineConfig might reboot the targeted nodes.)

Make sure the Onload kernel modules are unloaded, i.e. by running directly in the worker nodes:
```
# lsmod | grep onload
# rmmod onload sfc_char sfc_resource
```
This is because, by the time of the module removal, the kernel modules may still be in use by userspace, and the early revisions of KMM won't retry the module unloading.

Kustomize will remove build config but not build products. Confirm the state of uploaded Onload images:
```
$ oc get image | grep onload
```
Remove any outstanding manually with `oc delete image`. (Not providing any automated invocation to prevent removal of the false-positive images.)

Copyright (c) 2023 Advanced Micro Devices, Inc.
