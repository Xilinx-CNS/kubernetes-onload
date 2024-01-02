
# Selecting Nodes with AMD Solarflare hardware using Node Feature Discovery (NFD)

## Cluster configuration

[Node Feature Discovery (NFD)](https://kubernetes-sigs.github.io/node-feature-discovery)
([Redhat documentation](https://docs.openshift.com/container-platform/4.14/hardware_enablement/psap-node-feature-discovery-operator.html#create-cd-cli_node-feature-discovery-operator))
enables the selection of nodes based on hardware features and system configuration.
NFD-Worker runs on each node to detect changes which are then used to label the node.

A `NodeFeatureDiscovery` CR enables the detections you require. A full example is provided in the above documentation
if you do not already have one configured.

To enable detection of AMD Solarflare cards, identified by the PCIe Subsystem Vendor ID '1924',
add the following configuration to your CR's `configData` section:

```yaml
kind: NodeFeatureDiscovery
...
spec:
  ...
  workerConfig:
    configData: |
      sources:
        pci:
          deviceClassWhitelist:
            - "1924"
          deviceLabelFields:
            - "subsystem_vendor"
```

After NFD is deployed, configured, and its daemons have performed detections, verify with:

```sh
kubectl get nodes -l feature.node.kubernetes.io/pci-1924.present=true
```

## Onload Custom Resource (CR) & workload configuration

Now the above is configured, automated build and loading of the out-of-tree `sfc` driver on all AMD Solarflare
hardware nodes can be easily achieved through the addition the following node label selector in
your Onload CR and/or workloads:

```yaml
  selector:
    feature.node.kubernetes.io/pci-1924.present: "true"
```

## Footnotes

```yaml
SPDX-License-Identifier: MIT
SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
```
