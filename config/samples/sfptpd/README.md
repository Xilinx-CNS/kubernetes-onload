
# Solarflare Enhanced PTP Daemon (sfptpd)

To run `sfptpd` in a Kubernetes cluster, deploy the `sfptpd` privileged
container to all nodes receiving PTP signal and
[disable the node's existing time services](#disable-default-time-source).

This directory provides a Kustomize formatted base
([DaemonSet](base/daemonset.yaml)) and example overlays.

(The Onload Operator does not presently manage `sfptpd`; please refer to
Release Notes for details on expressing interest in this feature.)

## Quickstart

The example Kustomize overlay
[overlays/example-bond0/kustomization.yaml](overlays/example-bond0/kustomization.yaml)
would deploy the following to a namespace called `sfptpd`:

1. A DaemonSet deploying the `sfptpd` container to worker nodes labelled
   `node-role.kubernetes.io/ptp`.
2. A [sfptpd.cfg](overlays/example-bond0/sfptpd.cfg) config file specifying
   the pod's PTP ethernet interface name as `bond0`. (A bond is not required.)

To use in your own cluster, make a copy and apply:

```sh
cp -r overlays/example-bond0 overlays/my-cluster
$EDITOR overlays/my-cluster/kustomization.yaml
$EDITOR overlays/my-cluster/sfptpd.cfg
kubectl label nodes <your-ptp-node> node-role.kubernetes.io/ptp=
kubectl apply -k overlays/my-cluster
```

Ensure you have also disabled the node's existing time services.

## Disable default time source

A node must have one time source only. sfptpd should replace services such
as `ntpd` and `chronyd` if they manage the local host's time.

If you are running OpenShift, follow the [Redhat documentation on disabling chronyd.service](https://docs.openshift.com/container-platform/4.10/post_installation_configuration/machine-configuration-tasks.html#cnf-disable-chronyd_post-install-machine-configuration-tasks).
Ensure the resulting MachineConfigPool exactly selects the nodes specified
above; if using the Quickstart's node label of `node-role.kubernetes.io/ptp`,
use a NodeSelector matching this label.

## Source Code & Documentation

Container images with compiled binaries are provided. To build the container
afresh, refer to
[sfptpd on GitHub](https://github.com/Xilinx-CNS/sfptpd/#building-a-container-image).

Example config (.cfg) files are available from
[sfptpd on GitHub](https://github.com/Xilinx-CNS/sfptpd/tree/master/config).

Full documentation is available in the
[Enhanced PTP User Guide](https://docs.xilinx.com/r/en-US/ug1602-ptp-user)

## Footnotes

(c) Copyright 2023 Advanced Micro Devices, Inc.
