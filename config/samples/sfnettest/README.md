# Onload application example: sfnettest

Here, we run a small utility, [`sfnt-pingpong` from sfnettest](https://github.com/Xilinx-CNS/cns-sfnettest), in a client and server pair to demonstrate Onload acceleration.

## Deploy

The example will require customisation to your environment. By default, this will deploy two pods running on nodes named `compute-0` and `compute-1`:

```sh
kubectl apply -f client-server.yaml
```

## Interactive test

Obtain the SFC interface's IP address of the `onload-sfnettest-server` pod:

```sh
$ kubectl describe pod onload-sfnettest-server | grep AddedInterface
  Normal  AddedInterface  24s   multus   Add eth0 [192.168.8.114/23] from openshift-sdn
  Normal  AddedInterface  24s   multus   Add net1 [198.19.0.1/16] from default/ipvlan-sf0
```

The server pod is already running the accelerated `sfnt-pingpong` instance.

Run the client from the `onload-sfnettest-client` pod:

```sh
kubectl exec onload-sfnettest-client -- sfnt-pingpong udp 198.19.0.1
```

You will likely want to [use an Onload profile](../../../README.md#using-onload-profiles).

---

Copyright (c) 2023 Advanced Micro Devices, Inc.
