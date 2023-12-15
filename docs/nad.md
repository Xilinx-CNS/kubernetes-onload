# Network Attachment Definitions (NADs)

KubernetesOnload's recommended CNI is Multus, which is configured through a `NetworkAttachmentDefinition` *kind* of CR.

## Simple examples

The following examples presume an interface name of `bond0` and address allocation of `whereabouts` but
neither bonding nor a specific address allocation method are required.

Ensure the NAD's `metadata.namespace` matches that of workload pods if Multus namespace isolation is configured.

### IPVLAN

```yaml
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: ipvlan-bond0
spec:
  config: |
    {
      "cniVersion": "0.3.1",
      "type": "ipvlan",
      "name": "ipvlan-bond0",
      "master": "bond0",
      "ipam": {
        "type": "whereabouts",
        "range": "198.19.0.0/16"
      }
    }
```

### MACVLAN

```yaml
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: macvlan-bond0
spec:
  config: |
    {
      "cniVersion": "0.3.1",
      "type": "macvlan",
      "name": "macvlan-bond0",
      "master": "bond0",
      "mode": "bridge",
      "ipam": {
          "type": "whereabouts",
          "range": "198.18.0.0/16"
      }
    }
```

## Footnotes

```yaml
SPDX-License-Identifier: MIT
SPDX-FileCopyrightText: (c) Copyright 2023 Advanced Micro Devices, Inc.
```
