# Onload application example: sfnettest

Here, we run a small utility, [`sfnt-pingpong` from sfnettest](https://github.com/Xilinx-CNS/cns-sfnettest), in a
client and server pair to demonstrate Onload acceleration.

The sfnettest image is solely focused on its performance utils and thus has a micro shell environment; in-depth network
inspection should be performed using dedicated software or Onload tools available on the node's host filesystem.

## Deploy

The following example [client-server.yaml](client-server.yaml) will require customisation to your environment. The
manifest utilises two separately deployed resources which are recommended as part of a full Onload Operator deployment:

* A [Multus network](../../../docs/nad.md)
  -- connects the pods to hardware that supports acceleration.
* An [Onload profile](../../../README.md#using-onload-profiles)
  -- sets environment variables for the pod which are then consumed by userland Onload running in the container(s).

Review the defaults and apply:

* Node names: `compute-0` and `compute-1`
* Network: `ipvlan-bond0` (Multus)
* Namespace: `default`
* Onload accelerated: `amd.com/onload` resource
* Onload profile: `onload-latency-profile`

```sh
kubectl apply --validate=true -f client-server.yaml
kubectl describe pods -l app.kubernetes.io/part-of=sfnettest
```

## Interactive test

Obtain the SFC interface's IP address of the `onload-sfnettest-server` pod, here `198.19.0.1`:

```sh
$ kubectl get events --field-selector involvedObject.name=sfnettest --field-selector reason=AddedInterface
LAST SEEN   TYPE     REASON           OBJECT                        MESSAGE
24s         Normal   AddedInterface   pod/onload-sfnettest-client   Add eth0 [192.168.6.203/23] from openshift-sdn
24s         Normal   AddedInterface   pod/onload-sfnettest-client   Add net1 [198.19.1.1/16] from default/ipvlan-bond0
24s         Normal   AddedInterface   pod/onload-sfnettest-server   Add eth0 [192.168.8.143/23] from openshift-sdn
24s         Normal   AddedInterface   pod/onload-sfnettest-server   Add net1 [198.19.0.1/16] from default/ipvlan-bond0
```

The server pod is already running the accelerated `sfnt-pingpong` instance.

Run `sfnt-pingpong` as a client within the `onload-sfnettest-client` pod, which has an accelerated environment:

```sh
kubectl exec onload-sfnettest-client -- sfnt-pingpong udp 198.19.0.1
```

---

Copyright (c) 2023-2024 Advanced Micro Devices, Inc.
